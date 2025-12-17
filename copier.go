package nie

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

const (
	layoutDateTime  = "2006-01-02 15:04:05"
	layoutDateOnly  = "2006-01-02"
	layoutYearMonth = "2006-01"
)

var nullTimeParseLayouts = []string{
	time.DateTime,
	time.DateOnly,
	layoutYearMonth, // 兼容仅到月份的场景，如 "2025-10"
}

var (
	errInvalidJSON        = errors.New("invalid json")
	errExpectedJSONArray  = errors.New("expected json array")
	errExpectedJSONObject = errors.New("expected json object")
	errExpectedJSONString = errors.New("expected json string")
)

func isEmptyJSONObjectBytes(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	i := 0
	j := len(b) - 1
	for i <= j {
		c := b[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		i++
	}
	for j >= i {
		c := b[j]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		j--
	}
	// trimmed must be exactly "{}"
	return j-i == 1 && b[i] == '{' && b[j] == '}'
}

var copier4BffOption = copier.Option{
	IgnoreEmpty: true,
	DeepCopy:    true,
}

// Copier4Bff 使用 copier 深拷贝
//
// 用于将 *structpb.Struct 和 []*structpb.Struct 字段转换为自定义结构体
func Copier4Bff(to interface{}, from interface{}) error {
	if err := copier.CopyWithOption(to, from, copier4BffOption); err != nil {
		return err
	}
	// 然后，使用反射处理 *structpb.Struct 和 []*structpb.Struct 字段
	return maybeConvertStructPBFields(to, from)
}

// Copier4Ent 使用 copier 深拷贝，加载自定义转换器
//
// 用于entity结构体和pb结构体之间的转换
func Copier4Ent(to interface{}, from interface{}) error {
	// 首先，使用 copier 进行基本字段的复制
	return copier.CopyWithOption(to, from, copier4EntOption)
	//FieldNameMapping: []copier.FieldNameMapping{
	//		{
	//			SrcType: time.Time{},
	//			DstType: copier.String,
	//			Mapping: map[string]string{
	//				"CreatedAt": "CreateTime",
	//				"UpdatedAt": "UpdateTime",
	//				"DeletedAt": "DeleteTime",
	//			},
	//		},
	//	}
}

// CopierConverters 定义 copier.Option 中 Converters 转换器列表
var CopierConverters = getAllConverters()

var copier4EntOption = copier.Option{
	IgnoreEmpty: true,
	DeepCopy:    true,
	Converters:  CopierConverters,
}

// getAllConverters 定义 copier.Option 中 Converters 转换器列表
func getAllConverters() []copier.TypeConverter {
	converterFuncList := []func() []copier.TypeConverter{
		GetNullTimeConverters,
		GetJSONConverters,
		GetStructPBSliceConverters,
		GetStructPBConverters,
		GetTimeConverters,
	}

	// 目前固定为 2 + 2 + 2 + 2 + 3 = 11 个转换器
	allConverters := make([]copier.TypeConverter, 0, 11)
	for _, fn := range converterFuncList {
		allConverters = append(allConverters, fn()...)
	}
	return allConverters
}

// GetNullTimeConverters 获取 sql.NullTime ←→ string 转换器
func GetNullTimeConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// sql.NullTime -> string
		{
			SrcType: sql.NullTime{},
			DstType: copier.String,
			Fn: func(src interface{}) (interface{}, error) {
				nt := src.(sql.NullTime)
				if !nt.Valid {
					return "", nil
				}
				/// 整天输出 YYYY-MM-DD，否则输出 YYYY-MM-DD HH:MM:SS
				if nt.Time.Hour() == 0 && nt.Time.Minute() == 0 && nt.Time.Second() == 0 {
					return nt.Time.Format(layoutDateOnly), nil
				}
				return nt.Time.Format(layoutDateTime), nil
			},
		},
		// string -> sql.NullTime
		{
			SrcType: copier.String,
			DstType: sql.NullTime{},
			Fn: func(src interface{}) (interface{}, error) {
				s := strings.TrimSpace(src.(string))
				if s == "" {
					return sql.NullTime{Valid: false}, nil
				}
				for _, layout := range nullTimeParseLayouts {
					if t, err := time.Parse(layout, s); err == nil {
						// 若为按月字符串，则归一为该月最后一天 00:00:00
						if layout == layoutYearMonth {
							loc := t.Location()
							lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, loc)
							return sql.NullTime{Time: lastDay, Valid: true}, nil
						}
						return sql.NullTime{Time: t, Valid: true}, nil
					}
				}
				return sql.NullTime{Valid: false}, nil
			},
		},
	}
}

// GetJSONConverters 获取 datatypes.JSON ←→ []string 转换器
func GetJSONConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// datatypes.JSON -> []string
		{
			SrcType: datatypes.JSON{},
			DstType: []string{},
			Fn: func(src interface{}) (interface{}, error) {
				jsonBytes := []byte(src.(datatypes.JSON))
				if len(jsonBytes) == 0 {
					return []string(nil), nil
				}
				if !gjson.ValidBytes(jsonBytes) {
					return nil, errInvalidJSON
				}
				if isEmptyJSONObjectBytes(jsonBytes) {
					// 兼容脏数据：字段被写成 "{}" 时按空/未设置处理
					return []string(nil), nil
				}
				res := gjson.ParseBytes(jsonBytes)
				if res.Type == gjson.Null {
					// 保持与 json.Unmarshal("null", &[]string{}) 一致：返回 nil slice
					return []string(nil), nil
				}
				if !res.IsArray() {
					return nil, errExpectedJSONArray
				}
				arr := res.Array()
				out := make([]string, 0, len(arr))
				var typeErr error
				for _, v := range arr {
					if v.Type == gjson.String {
						out = append(out, v.String())
						continue
					}
					typeErr = errExpectedJSONString
					break
				}
				if typeErr != nil {
					return nil, typeErr
				}
				return out, nil
			},
		},
		// []string -> datatypes.JSON
		{
			SrcType: []string{},
			DstType: datatypes.JSON{},
			Fn: func(src interface{}) (interface{}, error) {
				arr := src.([]string)
				// 保持与 json.Marshal([]string(nil)) 一致：nil slice -> "null"
				if arr == nil {
					return datatypes.JSON([]byte("null")), nil
				}
				var buf bytes.Buffer
				buf.Grow(2 + len(arr)*2)
				buf.WriteByte('[')
				for i, s := range arr {
					if i > 0 {
						buf.WriteByte(',')
					}
					b, err := json.Marshal(s)
					if err != nil {
						return nil, err
					}
					buf.Write(b)
				}
				buf.WriteByte(']')
				return datatypes.JSON(buf.Bytes()), nil
			},
		},
	}
}

// GetStructPBSliceConverters 定义 []*structpb.Struct ←→ datatypes.JSON 的转换器
func GetStructPBSliceConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// datatypes.JSON -> []*structpb.Struct
		{
			SrcType: datatypes.JSON{},
			DstType: []*structpb.Struct{},
			Fn: func(src interface{}) (interface{}, error) {
				jsonBytes := []byte(src.(datatypes.JSON))
				if len(jsonBytes) == 0 {
					return []*structpb.Struct(nil), nil
				}
				if !gjson.ValidBytes(jsonBytes) {
					return nil, errInvalidJSON
				}
				if isEmptyJSONObjectBytes(jsonBytes) {
					// 兼容脏数据：字段被写成 "{}" 时按空/未设置处理
					return []*structpb.Struct(nil), nil
				}
				res := gjson.ParseBytes(jsonBytes)
				if res.Type == gjson.Null {
					// 保持与 json.Unmarshal("null", &[]*structpb.Struct{}) 一致：返回 nil slice
					return []*structpb.Struct(nil), nil
				}
				if !res.IsArray() {
					return nil, errExpectedJSONArray
				}
				arr := res.Array()
				out := make([]*structpb.Struct, 0, len(arr))
				for _, v := range arr {
					if v.Type == gjson.Null {
						out = append(out, nil)
						continue
					}
					if v.Type != gjson.JSON {
						return nil, errExpectedJSONObject
					}
					ps := &structpb.Struct{}
					if err := ps.UnmarshalJSON([]byte(v.Raw)); err != nil {
						return nil, err
					}
					out = append(out, ps)
				}
				return out, nil
			},
		},
		// []*structpb.Struct -> datatypes.JSON
		{
			SrcType: []*structpb.Struct{},
			DstType: datatypes.JSON{},
			Fn: func(src interface{}) (interface{}, error) {
				arr := src.([]*structpb.Struct)
				var buf bytes.Buffer
				buf.Grow(2 + len(arr)*2)
				buf.WriteByte('[')
				for i, ps := range arr {
					if i > 0 {
						buf.WriteByte(',')
					}
					if ps == nil {
						buf.WriteString("null")
						continue
					}
					b, err := ps.MarshalJSON()
					if err != nil {
						return nil, err
					}
					buf.Write(b)
				}
				buf.WriteByte(']')
				return datatypes.JSON(buf.Bytes()), nil
			},
		},
	}
}

// GetStructPBConverters 返回 *structpb.Struct ←→ datatypes.JSON 的转换器
func GetStructPBConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// *structpb.Struct -> datatypes.JSON
		{
			SrcType: &structpb.Struct{},
			DstType: datatypes.JSON{},
			Fn: func(src interface{}) (interface{}, error) {
				protoStruct, ok := src.(*structpb.Struct)
				if !ok {
					return nil, errors.New("source is not *structpb.Struct")
				}

				// 转换为 JSON 字节
				jsonData, err := protoStruct.MarshalJSON()
				if err != nil {
					return nil, err
				}

				return datatypes.JSON(jsonData), nil
			},
		},
		// datatypes.JSON -> *structpb.Struct
		{
			SrcType: datatypes.JSON{},
			DstType: &structpb.Struct{},
			Fn: func(src interface{}) (interface{}, error) {
				jsonData := src.(datatypes.JSON)

				// 创建 *structpb.Struct 并解析 JSON
				protoStruct := &structpb.Struct{}
				err := protoStruct.UnmarshalJSON(jsonData)
				if err != nil {
					return nil, err
				}

				return protoStruct, nil
			},
		},
	}
}

// GetTimeConverters 获取 time.Time ←→ string 转换器
func GetTimeConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// 新增：string→string 直接返回，优先级最高
		{
			SrcType: copier.String,
			DstType: copier.String,
			Fn: func(src interface{}) (interface{}, error) {
				return src.(string), nil // 直接返回源字符串，不处理
			},
		},
		// time.Time -> string
		{
			SrcType: time.Time{},
			DstType: copier.String,
			Fn: func(src interface{}) (interface{}, error) {
				timeVal := src.(time.Time)
				// 如果是零值，则返回空字符串
				if timeVal.IsZero() {
					return "", nil
				}
				// 格式化时间为字符串
				return timeVal.Format(layoutDateTime), nil
			},
		},
		// string -> time.Time
		{
			SrcType: copier.String,
			DstType: time.Time{},
			Fn: func(src interface{}) (interface{}, error) {
				dateStr := src.(string)
				// 如果字符串为空，则返回零值的 time.Time
				if dateStr == "" {
					return time.Time{}, nil
				}
				// 解析字符串为时间类型
				parsedTime, err := time.Parse(layoutDateTime, dateStr)
				if err != nil {
					return nil, err
				}
				return parsedTime, nil
			},
		},
	}
}

var (
	typeStructPB      = reflect.TypeOf(&structpb.Struct{})
	typeSliceStructPB = reflect.TypeOf([]*structpb.Struct{})
)

var structPBTypeScanCache sync.Map // map[reflect.Type]bool

type fieldIndexPathCacheKey struct {
	from reflect.Type
	to   reflect.Type
}

var fieldIndexPathCache sync.Map // map[fieldIndexPathCacheKey][][]int

func maybeConvertStructPBFields(to interface{}, from interface{}) error {
	if !shouldConvertStructPBFields(to, from) {
		return nil
	}
	return convertStructPBFields(to, from)
}

func shouldConvertStructPBFields(to interface{}, from interface{}) bool {
	toType := reflect.TypeOf(to)
	fromType := reflect.TypeOf(from)
	if toType == nil || fromType == nil {
		return false
	}

	// 顶层：source=*structpb.Struct → target=*T(struct)
	if fromType == typeStructPB && toType.Kind() == reflect.Ptr && toType.Elem().Kind() == reflect.Struct {
		return true
	}

	// 顶层：source=*T(struct) → target=*structpb.Struct
	if toType == typeStructPB && fromType.Kind() == reflect.Ptr && fromType.Elem().Kind() == reflect.Struct {
		return true
	}

	// 顶层：source=*[]*structpb.Struct → target=*[]T / *[]*T
	if fromType.Kind() == reflect.Ptr && toType.Kind() == reflect.Ptr &&
		fromType.Elem().Kind() == reflect.Slice && fromType.Elem() == typeSliceStructPB &&
		toType.Elem().Kind() == reflect.Slice {
		return true
	}

	// 顶层：source=*[]T / *[]*T → target=*[]*structpb.Struct
	if toType.Kind() == reflect.Ptr && fromType.Kind() == reflect.Ptr &&
		toType.Elem().Kind() == reflect.Slice && toType.Elem() == typeSliceStructPB &&
		fromType.Elem().Kind() == reflect.Slice && fromType.Elem() != typeSliceStructPB {
		return true
	}

	// 普通结构体：仅当任一侧类型树包含 structpb.Struct 相关字段时才需要进入反射递归
	if toType.Kind() != reflect.Ptr || fromType.Kind() != reflect.Ptr {
		return false
	}
	if toType.Elem().Kind() != reflect.Struct || fromType.Elem().Kind() != reflect.Struct {
		return false
	}

	return typeTreeContainsStructPB(toType.Elem()) || typeTreeContainsStructPB(fromType.Elem())
}

func typeTreeContainsStructPB(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if v, ok := structPBTypeScanCache.Load(t); ok {
		return v.(bool)
	}
	visited := make(map[reflect.Type]struct{}, 16)
	result := typeTreeContainsStructPBNoCache(t, visited)
	structPBTypeScanCache.Store(t, result)
	return result
}

func typeTreeContainsStructPBNoCache(t reflect.Type, visited map[reflect.Type]struct{}) bool {
	if t == nil {
		return false
	}
	if t == typeStructPB || t == typeSliceStructPB {
		return true
	}
	if _, ok := visited[t]; ok {
		return false
	}
	visited[t] = struct{}{}

	switch t.Kind() {
	case reflect.Ptr:
		return typeTreeContainsStructPBNoCache(t.Elem(), visited)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			ft := t.Field(i).Type
			if typeTreeContainsStructPBNoCache(ft, visited) {
				return true
			}
		}
		return false
	case reflect.Slice, reflect.Array:
		return typeTreeContainsStructPBNoCache(t.Elem(), visited)
	default:
		return false
	}
}

func getFieldIndexPaths(fromType, toType reflect.Type) [][]int {
	if fromType == nil || toType == nil || fromType.Kind() != reflect.Struct || toType.Kind() != reflect.Struct {
		return nil
	}
	key := fieldIndexPathCacheKey{from: fromType, to: toType}
	if v, ok := fieldIndexPathCache.Load(key); ok {
		return v.([][]int)
	}

	paths := make([][]int, fromType.NumField())
	for i := 0; i < fromType.NumField(); i++ {
		sf := fromType.Field(i)
		if tf, ok := toType.FieldByName(sf.Name); ok {
			idx := make([]int, len(tf.Index))
			copy(idx, tf.Index)
			paths[i] = idx
		}
	}

	fieldIndexPathCache.Store(key, paths)
	return paths
}

func decodeStructPBInto(dst interface{}, src *structpb.Struct) error {
	if src == nil {
		return nil
	}
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "json",
		WeaklyTypedInput: true,
		Result:           dst,
	})
	if err != nil {
		return err
	}
	return dec.Decode(src.AsMap())
}

// convertStructPBFields 递归处理任意层级中的 *structpb.Struct / []*structpb.Struct
func convertStructPBFields(to interface{}, from interface{}) error {
	tv := reflect.ValueOf(to)
	fv := reflect.ValueOf(from)

	// 顶层：source=*structpb.Struct → target=struct
	if tv.Kind() == reflect.Ptr && fv.Kind() == reflect.Ptr &&
		!tv.IsNil() && !fv.IsNil() &&
		fv.Type() == typeStructPB &&
		tv.Elem().Kind() == reflect.Struct {
		ps := fv.Interface().(*structpb.Struct)
		return decodeStructPBInto(to, ps)
	}

	// 顶层 source=*T(struct) → target=*structpb.Struct
	if tv.Kind() == reflect.Ptr && fv.Kind() == reflect.Ptr &&
		!tv.IsNil() && !fv.IsNil() &&
		tv.Type() == typeStructPB &&
		fv.Elem().Kind() == reflect.Struct {
		b, err := json.Marshal(fv.Interface())
		if err != nil {
			return err
		}
		if err := tv.Interface().(*structpb.Struct).UnmarshalJSON(b); err != nil {
			return err
		}
		return nil
	}

	// 顶层：[]*structpb.Struct → []*T / []T
	if tv.Kind() == reflect.Ptr && fv.Kind() == reflect.Ptr &&
		!tv.IsNil() && !fv.IsNil() &&
		fv.Elem().Kind() == reflect.Slice &&
		fv.Elem().Type() == typeSliceStructPB &&
		tv.Elem().Kind() == reflect.Slice {
		arr := fv.Elem()
		outSlice := reflect.MakeSlice(tv.Elem().Type(), 0, arr.Len())
		elemType := tv.Elem().Type().Elem()
		ptrElem := elemType.Kind() == reflect.Ptr
		baseType := elemType
		if ptrElem {
			baseType = elemType.Elem()
		}
		for i := 0; i < arr.Len(); i++ {
			ps, _ := arr.Index(i).Interface().(*structpb.Struct)
			if ps == nil {
				continue
			}
			newElem := reflect.New(baseType).Interface()
			if err := decodeStructPBInto(newElem, ps); err != nil {
				return err
			}
			if ptrElem {
				outSlice = reflect.Append(outSlice, reflect.ValueOf(newElem))
			} else {
				outSlice = reflect.Append(outSlice, reflect.ValueOf(newElem).Elem())
			}
		}
		tv.Elem().Set(outSlice)
		return nil
	}

	// 顶层 []*T / []T → []*structpb.Struct
	if tv.Kind() == reflect.Ptr && fv.Kind() == reflect.Ptr &&
		!tv.IsNil() && !fv.IsNil() &&
		tv.Elem().Kind() == reflect.Slice &&
		tv.Elem().Type() == typeSliceStructPB &&
		fv.Elem().Kind() == reflect.Slice &&
		fv.Elem().Type() != typeSliceStructPB {
		srcSlice := fv.Elem()
		outSlice := reflect.MakeSlice(tv.Elem().Type(), 0, srcSlice.Len())
		for i := 0; i < srcSlice.Len(); i++ {
			elem := srcSlice.Index(i)
			var src interface{}
			switch elem.Kind() {
			case reflect.Ptr:
				if elem.IsNil() {
					continue
				}
				if elem.Elem().Kind() != reflect.Struct {
					continue
				}
				src = elem.Interface()
			case reflect.Struct:
				src = elem.Interface()
			default:
				continue
			}
			b, err := json.Marshal(src)
			if err != nil {
				return err
			}
			ps := &structpb.Struct{}
			if err := ps.UnmarshalJSON(b); err != nil {
				return err
			}
			outSlice = reflect.Append(outSlice, reflect.ValueOf(ps))
		}
		tv.Elem().Set(outSlice)
		return nil
	}

	return walkAndConvert(tv, fv)
}
func walkAndConvert(toVal reflect.Value, fromVal reflect.Value) error {
	if fromVal.Kind() != reflect.Ptr || toVal.Kind() != reflect.Ptr {
		return nil
	}
	if fromVal.IsNil() || toVal.IsNil() {
		return nil
	}

	fromVal = fromVal.Elem()
	toVal = toVal.Elem()
	if fromVal.Kind() != reflect.Struct || toVal.Kind() != reflect.Struct {
		return nil
	}

	fromType := fromVal.Type()
	toType := toVal.Type()
	indexPaths := getFieldIndexPaths(fromType, toType)
	for i := 0; i < fromVal.NumField(); i++ {
		fromField := fromVal.Field(i)
		var toField reflect.Value
		if indexPaths != nil {
			idx := indexPaths[i]
			if idx == nil {
				continue
			}
			toField = toVal.FieldByIndex(idx)
		} else {
			fieldInfo := fromType.Field(i)
			toField = toVal.FieldByName(fieldInfo.Name)
		}
		if !toField.IsValid() || !toField.CanSet() {
			continue
		}

		// 1. 直接转换 *structpb.Struct -> *TargetStruct
		if fromField.Type() == typeStructPB && !fromField.IsNil() &&
			toField.Kind() == reflect.Ptr && toField.Type() != typeStructPB {
			targetType := toField.Type().Elem()
			inst := reflect.New(targetType).Interface()
			if err := mapstructure.Decode(fromField.Interface().(*structpb.Struct).AsMap(), inst); err != nil {
				return err
			}
			toField.Set(reflect.ValueOf(inst))
			continue
		}

		// 2. 直接转换 []*structpb.Struct -> []*TargetStruct
		if fromField.Type() == typeSliceStructPB && fromField.Len() > 0 &&
			toField.Kind() == reflect.Slice && toField.Type() != typeSliceStructPB {
			// 目标元素类型（支持 []*T / []T）
			var elemType reflect.Type
			if toField.Type().Elem().Kind() == reflect.Ptr {
				elemType = toField.Type().Elem().Elem()
			} else {
				elemType = toField.Type().Elem()
			}

			newSlice := reflect.MakeSlice(toField.Type(), 0, fromField.Len())
			for j := 0; j < fromField.Len(); j++ {
				ps, ok := fromField.Index(j).Interface().(*structpb.Struct)
				if !ok || ps == nil {
					continue
				}
				inst := reflect.New(elemType).Interface()
				if err := mapstructure.Decode(ps.AsMap(), inst); err != nil {
					return err
				}
				// 统一追加指针或值
				if toField.Type().Elem().Kind() == reflect.Ptr {
					newSlice = reflect.Append(newSlice, reflect.ValueOf(inst))
				} else {
					newSlice = reflect.Append(newSlice, reflect.ValueOf(inst).Elem())
				}
			}
			toField.Set(newSlice)
			continue
		}

		// 3. *TargetStruct / TargetStruct -> *structpb.Struct
		if toField.Type() == typeStructPB {
			var src interface{}
			switch fromField.Kind() {
			case reflect.Ptr:
				if fromField.IsNil() {
					continue
				}
				if fromField.Elem().Kind() != reflect.Struct {
					continue
				}
				src = fromField.Interface()
			case reflect.Struct:
				src = fromField.Interface()
			default:
				continue
			}
			data, err := json.Marshal(src)
			if err != nil {
				return err
			}
			ps := &structpb.Struct{}
			if err := ps.UnmarshalJSON(data); err != nil {
				return err
			}
			toField.Set(reflect.ValueOf(ps))
			continue
		}

		// 4. 任意 []X -> []*structpb.Struct
		if toField.Type() == typeSliceStructPB && fromField.Kind() == reflect.Slice {
			b, err := json.Marshal(fromField.Interface())
			if err != nil {
				return err
			}
			var arr []*structpb.Struct
			if err := json.Unmarshal(b, &arr); err != nil {
				return err
			}
			toField.Set(reflect.ValueOf(arr))
			continue
		}

		// 5. 递归：结构体 / 指针结构体
		switch fromField.Kind() {
		case reflect.Ptr:
			if !fromField.IsNil() && toField.Kind() == reflect.Ptr && !toField.IsNil() {
				if err := walkAndConvert(toField, fromField); err != nil {
					return err
				}
			}
		case reflect.Struct:
			if toField.Kind() == reflect.Struct {
				// 创建可寻址副本再处理（避免不可寻址导致的问题）
				fFrom := fromField.Addr()
				fTo := toField.Addr()
				if err := walkAndConvert(fTo, fFrom); err != nil {
					return err
				}
			}
		case reflect.Slice:
			// 递归切片元素
			if toField.Kind() == reflect.Slice && fromField.Len() == toField.Len() {
				for idx := 0; idx < fromField.Len(); idx++ {
					fElem := fromField.Index(idx)
					tElem := toField.Index(idx)
					if fElem.Kind() == reflect.Ptr && tElem.Kind() == reflect.Ptr {
						_ = walkAndConvert(tElem, fElem)
					} else if fElem.Kind() == reflect.Struct && tElem.Kind() == reflect.Struct {
						_ = walkAndConvert(tElem.Addr(), fElem.Addr())
					}
				}
			}
		}
	}
	return nil
}
