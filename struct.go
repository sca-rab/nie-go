package nie

import "reflect"

// TimeStruct 公共时间结构体
type TimeStruct struct {
	CreateTime string // 创建时间
	UpdateTime string // 更新时间
	DeleteTime string // 删除时间
}

// BaseStruct 公共完整解构体
//
// 带增删改时间、ID、姓名
type BaseStruct struct {
	CreateId   int64  // 创建人ID
	UpdateId   int64  // 更新人ID
	DeleteId   int64  // 删除人ID
	CreateBy   string // 创建人
	UpdateBy   string // 更新人
	DeleteBy   string // 删除人
	CreateTime string // 创建时间
	UpdateTime string // 更新时间
	DeleteTime string // 删除时间
}

// FullStruct 公共完整解构体
//
// 带增删改时间、ID、姓名、允许字段
type FullStruct struct {
	AllowFields []string // 允许字段
	CreateId    int64    // 创建人ID
	UpdateId    int64    // 更新人ID
	DeleteId    int64    // 删除人ID
	CreateBy    string   // 创建人
	UpdateBy    string   // 更新人
	DeleteBy    string   // 删除人
	CreateTime  string   // 创建时间
	UpdateTime  string   // 更新时间
	DeleteTime  string   // 删除时间
}

// CreateStruct 公共创建结构体
//
// 带创建时间、ID、姓名、允许字段
type CreateStruct struct {
	AllowFields []string // 允许字段
	CreateId    int64    // 创建人ID
	CreateBy    string   // 创建人
	CreateTime  string   // 创建时间
}

// UpdateStruct 公共更新结构体
//
// 带更新时间、ID、姓名、允许字段
type UpdateStruct struct {
	AllowFields []string // 允许字段
	UpdateId    int64    // 更新人ID
	UpdateBy    string   // 更新人
	UpdateTime  string   // 更新时间
}

// DeleteStruct 公共删除结构体
//
// 带删除时间、ID、姓名

// FieldOptions 定义 GetAllowFields 参数结构体，用于传递可选参数
type FieldOptions struct {
	Adds                interface{} // 可追加的字段，string 或 []string
	Filters             interface{} // 可过滤的字段，string 或 []string
	AddEnterpriseId     bool        // 是否添加EnterpriseId字段
	FiltersEnterpriseId bool        // 是否过滤EnterpriseId字段
}

// GetAllowFields 提取结构体字段名的函数，返回字段名切片
//
// model.SelectCreateFields 方法中已添加 CreateId, CreateBy
//
// model.SelectUpdateFields 方法中已添加 UpdateId, UpdateBy
func GetAllowFields(obj interface{}, options ...FieldOptions) []string {
	var fieldNames []string
	typ := reflect.TypeOf(obj)
	// 需要检查是否是指针类型，若是，则获取其所指向的类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// 默认添加结构体所有字段
	for i := 0; i < typ.NumField(); i++ {
		fieldNames = append(fieldNames, typ.Field(i).Name)
	}
	// 如果有传入选项
	if len(options) > 0 {
		option := options[0]

		// 处理 'Adds' 选项
		if option.Adds != nil {
			switch v := option.Adds.(type) {
			case string:
				// 只有在字段不存在时才追加
				if !contains(fieldNames, v) {
					fieldNames = append(fieldNames, v)
				}
			case []string:
				for _, addField := range v {
					// 只有在字段不存在时才追加
					if !contains(fieldNames, addField) {
						fieldNames = append(fieldNames, addField)
					}
				}
			}
		}

		// 处理 'Filters' 选项
		if option.Filters != nil {
			var filteredFields []string
			switch v := option.Filters.(type) {
			case string:
				for _, field := range fieldNames {
					if field != v {
						filteredFields = append(filteredFields, field)
					}
				}
			case []string:
				for _, field := range fieldNames {
					if !contains(v, field) {
						filteredFields = append(filteredFields, field)
					}
				}
			}
			fieldNames = filteredFields
		}
	}
	return fieldNames
}

// 检查切片中是否包含指定字符串
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
