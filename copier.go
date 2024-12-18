package nie

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"reflect"
	"time"
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
			DstType: "",
			Fn: func(src interface{}) (interface{}, error) {
				nullTime := src.(sql.NullTime)
				if !nullTime.Valid {
					return "", nil
				}
				return nullTime.Time.Format("2006-01-02"), nil
			},
		},
		// string -> sql.NullTime
		{
			SrcType: "",
			DstType: sql.NullTime{},
			Fn: func(src interface{}) (interface{}, error) {
				dateStr := src.(string)
				if dateStr == "" {
					return sql.NullTime{Valid: false}, nil
				}
				parsedTime, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					return nil, err
				}
				return sql.NullTime{Time: parsedTime, Valid: true}, nil
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

// convertStructPBFields 使用反射处理特殊字段
func convertStructPBFields(to interface{}, from interface{}) error {
	fromVal := reflect.ValueOf(from)
	toVal := reflect.ValueOf(to)

	if fromVal.Kind() != reflect.Ptr || toVal.Kind() != reflect.Ptr {
		return errors.New("from and to must be pointers")
	}

	fromVal = fromVal.Elem()
	toVal = toVal.Elem()

	if fromVal.Kind() != reflect.Struct || toVal.Kind() != reflect.Struct {
		return errors.New("from and to must point to structs")
	}

	fromType := fromVal.Type()

	for i := 0; i < fromVal.NumField(); i++ {
		fromField := fromVal.Field(i)
		fromFieldType := fromType.Field(i)
		toField := toVal.FieldByName(fromFieldType.Name)

		if !toField.IsValid() || !toField.CanSet() {
			continue
		}

		// 如果 from 和 to 都是*structpb.Struct 类型，跳过
		if fromField.Type() == reflect.TypeOf(&structpb.Struct{}) && toField.Type() == reflect.TypeOf(&structpb.Struct{}) {
			continue
		}
		// 如果 from 和 to 都是[]*structpb.Struct 类型，跳过
		if fromField.Type() == reflect.TypeOf([]*structpb.Struct{}) && toField.Type() == reflect.TypeOf([]*structpb.Struct{}) {
			continue
		}

		// 处理 *structpb.Struct -> *TargetStruct
		if fromField.Type() == reflect.TypeOf(&structpb.Struct{}) && !fromField.IsNil() {
			// 获取目标字段的类型
			targetType := toField.Type().Elem()

			// 创建目标类型的实例
			targetInstance := reflect.New(targetType).Interface()

			// 使用 mapstructure 进行解码
			err := mapstructure.Decode(fromField.Interface().(*structpb.Struct).AsMap(), targetInstance)
			if err != nil {
				fmt.Println("========================")
				fmt.Println("fromField:", fromField)
				return err
			}

			// 设置目标字段
			toField.Set(reflect.ValueOf(targetInstance))
		}

		// 处理 []*structpb.Struct -> []*TargetStruct
		if fromField.Type() == reflect.TypeOf([]*structpb.Struct{}) && fromField.Len() > 0 {
			// 获取目标切片的元素类型
			elemType := toField.Type().Elem().Elem()

			// 创建一个新的切片
			newSlice := reflect.MakeSlice(toField.Type(), 0, fromField.Len())

			for j := 0; j < fromField.Len(); j++ {
				protoStruct, ok := fromField.Index(j).Interface().(*structpb.Struct)
				if !ok || protoStruct == nil {
					continue
				}
				mapData := protoStruct.AsMap()

				// 创建目标元素的实例
				elemInstance := reflect.New(elemType).Interface()

				// 使用 mapstructure 进行解码
				err := mapstructure.Decode(mapData, elemInstance)
				if err != nil {
					return err
				}

				// 将元素追加到切片
				newSlice = reflect.Append(newSlice, reflect.ValueOf(elemInstance))
			}

			// 设置目标字段
			toField.Set(newSlice)
		}
	}

	return nil
}
