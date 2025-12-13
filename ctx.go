package nie

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
)

const (
	CtxUidKey          = "uid"
	CtxNickNameKey     = "nickName"
	CtxEnterpriseIdKey = "enterpriseId"
	CtxUnameKey        = "uname"
	CtxRoleKey         = "role"
	CtxOfficeIdKey     = "officeId"
)

// CtxGlobalInt 从上下文中获取元数据
func CtxGlobalInt(ctx context.Context, name string) (int64, error) {
	if md, ok := metadata.FromServerContext(ctx); ok {
		value, _ := strconv.ParseInt(md.Get(name), 10, 64)
		return value, nil
	}
	return 0, errors.BadRequest("FAIL_VALIDATE", "认证错误")
}

// CtxGlobalString 从上下文中获取元数据
func CtxGlobalString(ctx context.Context, name string) (string, error) {
	if md, ok := metadata.FromServerContext(ctx); ok {
		return md.Get(name), nil
	}
	return "", errors.BadRequest("FAIL_VALIDATE", "认证错误")
}

// ctxInt 从上下文中获取整型元数据
// 入参：ctx 为上下文；name 为键名
// 出参：返回从上下文中获取到的 int64，取不到或类型不匹配时返回 0
func ctxInt(ctx context.Context, name string) int64 {
	// 从上下文中获取值
	val := ctx.Value(name)
	if val == nil {
		// 未设置该键，返回默认值 0，避免 panic
		return 0
	}

	// 类型断言为 int64，并检查是否成功
	if v, ok := val.(int64); ok {
		return v
	}

	// 类型不匹配，同样返回默认值 0，避免 panic
	return 0
}

// ctxString 从上下文中获取元数据
func ctxString(ctx context.Context, name string) string {
	// 从上下文中获取值
	val := ctx.Value(name)
	if val == nil {
		// 未设置该键，返回默认值空字符串，避免 panic
		return ""
	}

	// 类型断言为 string，并检查是否成功
	if v, ok := val.(string); ok {
		return v
	}

	// 类型不匹配，同样返回默认值空字符串，避免 panic
	return ""

}

// ctxArr 从上下文中获取元数据
func ctxArr(ctx context.Context, name string) []string {
	value := ctx.Value(name).(string)
	arr := strings.Split(value, ",")
	return arr
}

// CtxUid 从上下文中获取用户ID
func CtxUid(ctx context.Context) int64 {
	return ctxInt(ctx, CtxUidKey)
}

// CtxNickName 从上下文中获取用户昵称
func CtxNickName(ctx context.Context) string {
	return ctxString(ctx, CtxNickNameKey)
}

// CtxEnterpriseId 从上下文中获取企业ID
func CtxEnterpriseId(ctx context.Context) int64 {
	return ctxInt(ctx, CtxEnterpriseIdKey)
}

// CtxUname 从上下文中获取用户名
func CtxUname(ctx context.Context) string {
	return ctxString(ctx, CtxUnameKey)
}

// CtxRoleKeys 从上下文中获取角色
func CtxRoleKeys(ctx context.Context) []string {
	return ctxArr(ctx, CtxRoleKey)
}

// CtxOfficeId 从上下文中获取单位ID
func CtxOfficeId(ctx context.Context) int64 {
	return ctxInt(ctx, CtxOfficeIdKey)
}
