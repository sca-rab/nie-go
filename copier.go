package nie

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

// Copier4Bff 使用 copier 深拷贝
//
// 用于将 *structpb.Struct 和 []*structpb.Struct 字段转换为自定义结构体
func Copier4Bff(to interface{}, from interface{}) error {
	err := copier.CopyWithOption(to, from, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})
	if err != nil {
		return err
	}
	// 然后，使用反射处理 *structpb.Struct 和 []*structpb.Struct 字段
	err = convertStructPBFields(to, from)
	if err != nil {
		return err
	}
	return nil
}

// Copier4Ent 使用 copier 深拷贝，加载自定义转换器
//
// 用于entity结构体和pb结构体之间的转换
func Copier4Ent(to interface{}, from interface{}) error {
	// 首先，使用 copier 进行基本字段的复制
	err := copier.CopyWithOption(to, from, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
		Converters:  CopierConverters,
	})
	if err != nil {
		return err
	}
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
	return nil
}

// CopierConverters 定义 copier.Option 中 Converters 转换器列表
var CopierConverters = getAllConverters()

// getAllConverters 定义 copier.Option 中 Converters 转换器列表
func getAllConverters() []copier.TypeConverter {
	converterFuncList := []func() []copier.TypeConverter{
		GetNullTimeConverters,
		GetJSONConverters,
		GetStructPBSliceConverters,
		GetStructPBConverters,
		GetTimeConverters,
	}

	var allConverters []copier.TypeConverter
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
					return nt.Time.Format("2006-01-02"), nil
				}
				return nt.Time.Format("2006-01-02 15:04:05"), nil
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
				layouts := []string{
					time.DateTime,
					time.DateOnly,
					"2006-01", // 兼容仅到月份的场景，如 "2025-10"
				}
				for _, layout := range layouts {
					if t, err := time.Parse(layout, s); err == nil {
						// 若为按月字符串，则归一为该月最后一天 00:00:00
						if layout == "2006-01" {
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
				var result []string
				err := json.Unmarshal(src.(datatypes.JSON), &result)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
		},
		// []string -> datatypes.JSON
		{
			SrcType: []string{},
			DstType: datatypes.JSON{},
			Fn: func(src interface{}) (interface{}, error) {
				jsonData, err := json.Marshal(src.([]string))
				if err != nil {
					return nil, err
				}
				return datatypes.JSON(jsonData), nil
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
				var result []*structpb.Struct
				err := json.Unmarshal(src.(datatypes.JSON), &result)
				return result, err
			},
		},
		// []*structpb.Struct -> datatypes.JSON
		{
			SrcType: []*structpb.Struct{},
			DstType: datatypes.JSON{},
			Fn: func(src interface{}) (interface{}, error) {
				jsonData, err := json.Marshal(src)
				return datatypes.JSON(jsonData), err
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
				return timeVal.Format("2006-01-02 15:04:05"), nil
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
				parsedTime, err := time.Parse("2006-01-02 15:04:05", dateStr)
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

// convertStructPBFields 递归处理任意层级中的 *structpb.Struct / []*structpb.Struct
func convertStructPBFields(to interface{}, from interface{}) error {
	tv := reflect.ValueOf(to)
	fv := reflect.ValueOf(from)
	// 顶层：source=*structpb.Struct，target=struct
	if tv.Kind() == reflect.Ptr && fv.Kind() == reflect.Ptr &&
		!tv.IsNil() && !fv.IsNil() &&
		fv.Type() == typeStructPB &&
		tv.Elem().Kind() == reflect.Struct {

		ps := fv.Interface().(*structpb.Struct)
		b, err := ps.MarshalJSON()
		if err != nil {
			return err
		}
		// 直接 JSON 反序列化到强类型结构体
		if err = json.Unmarshal(b, to); err != nil {
			return err
		}
		return nil
	}
	// 顶层：[]*structpb.Struct → []*T
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
			b, err := ps.MarshalJSON()
			if err != nil {
				return err
			}
			newElem := reflect.New(baseType).Interface()
			if err = json.Unmarshal(b, newElem); err != nil {
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
	for i := 0; i < fromVal.NumField(); i++ {
		fromField := fromVal.Field(i)
		fieldInfo := fromType.Field(i)
		toField := toVal.FieldByName(fieldInfo.Name)
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
