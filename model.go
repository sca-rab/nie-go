package main

import (
	"gorm.io/gorm"
	"time"
)

// TimeModel 时间模型
type TimeModel struct {
	CreatedAt time.Time      `gorm:"type:datetime;column:create_time;comment:创建时间" json:"createTime" copier:"CreateTime"`    // 创建时间
	UpdatedAt time.Time      `gorm:"type:datetime;column:update_time;comment:更新时间" json:"updateTime" copier:"UpdateTime"`    // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"type:datetime;column:delete_time;comment:删除时间" sql:"index" json:"-" copier:"DeleteTime"` // 删除时间
}

// BaseModel 基础模型
//
// 带增删改时间、ID、姓名
type BaseModel struct {
	CreatedAt time.Time      `gorm:"type:datetime;column:create_time;comment:创建时间" json:"createTime" copier:"CreateTime"`    // 创建时间
	UpdatedAt time.Time      `gorm:"type:datetime;column:update_time;comment:更新时间" json:"updateTime" copier:"UpdateTime"`    // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"type:datetime;column:delete_time;comment:删除时间" sql:"index" json:"-" copier:"DeleteTime"` // 删除时间
	CreateId  int64          `gorm:"type:bigint;column:create_id;comment:创建人id" json:"createId"`                             // 创建人id
	UpdateId  int64          `gorm:"type:bigint;column:update_id;comment:更新人id" json:"updateId"`                             // 更新人id
	DeleteId  int64          `gorm:"type:bigint;column:delete_id;comment:删除人id" json:"deleteId"`                             // 删除人id
	CreateBy  string         `gorm:"type:varchar(64);column:create_by;comment:创建人" json:"createBy"`                          // 创建人
	UpdateBy  string         `gorm:"type:varchar(64);column:update_by;comment:更新人" json:"updateBy"`                          // 更新人
	DeleteBy  string         `gorm:"type:varchar(64);column:delete_by;comment:删除人" json:"deleteBy"`                          // 删除人
}

// FullModel 完整模型
//
// 带增删改时间、ID、姓名、允许字段
type FullModel struct {
	CreatedAt   time.Time      `gorm:"type:datetime;column:create_time;comment:创建时间" json:"createTime" copier:"CreateTime"`    // 创建时间
	UpdatedAt   time.Time      `gorm:"type:datetime;column:update_time;comment:更新时间" json:"updateTime" copier:"UpdateTime"`    // 更新时间
	DeletedAt   gorm.DeletedAt `gorm:"type:datetime;column:delete_time;comment:删除时间" sql:"index" json:"-" copier:"DeleteTime"` // 删除时间
	CreateId    int64          `gorm:"type:bigint;column:create_id;comment:创建人id" json:"createId"`                             // 创建人id
	UpdateId    int64          `gorm:"type:bigint;column:update_id;comment:更新人id" json:"updateId"`                             // 更新人id
	DeleteId    int64          `gorm:"type:bigint;column:delete_id;comment:删除人id" json:"deleteId"`                             // 删除人id
	CreateBy    string         `gorm:"type:varchar(64);column:create_by;comment:创建人" json:"createBy"`                          // 创建人
	UpdateBy    string         `gorm:"type:varchar(64);column:update_by;comment:更新人" json:"updateBy"`                          // 更新人
	DeleteBy    string         `gorm:"type:varchar(64);column:delete_by;comment:删除人" json:"deleteBy"`                          // 删除人
	AllowFields []string       `gorm:"-" json:"allowFields"`                                                                   // 允许修改的字段
}

// HardDModel 硬删除
type HardDModel struct {
	CreatedAt time.Time `gorm:"column:create_time" json:"createTime"`
	UpdatedAt time.Time `gorm:"column:update_time" json:"updateTime"`
}

// SelectUpdateFields 构建允许更新字段
func SelectUpdateFields(db *gorm.DB, allowFields []string) *gorm.DB {
	if len(allowFields) == 0 {
		return db
	}
	// 将UpdateId、UpdateBy加入允许更新字段
	allowFields = append(allowFields, "UpdateId", "UpdateBy")
	return db.Select(allowFields)
}

// SelectCreateFields 构建允许创建字段
func SelectCreateFields(db *gorm.DB, allowFields []string) *gorm.DB {
	if len(allowFields) == 0 {
		return db
	}
	// 将CreateId、CreateBy加入允许创建字段
	allowFields = append(allowFields, "CreateId", "CreateBy")
	return db.Select(allowFields)
}
