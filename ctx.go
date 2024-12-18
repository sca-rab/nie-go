package nie

import (
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"strconv"
)

const (
	CtxUidKey          = "uid"
	CtxNickNameKey     = "nickName"
	CtxEnterpriseIdKey = "enterpriseId"
	CtxUnameKey        = "uname"
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

// CtxInt 从上下文中获取元数据
func CtxInt(ctx context.Context, name string) int64 {
	value := ctx.Value(name).(int64)
	return value
}

// CtxString 从上下文中获取元数据
func CtxString(ctx context.Context, name string) string {
	value := ctx.Value(name).(string)
	return value
}

// CtxUid 从上下文中获取用户ID
func CtxUid(ctx context.Context) int64 {
	return CtxInt(ctx, CtxUidKey)
}

// CtxNickName 从上下文中获取用户昵称
func CtxNickName(ctx context.Context) string {
	return CtxString(ctx, CtxNickNameKey)
}

// CtxEnterpriseId 从上下文中获取企业ID
func CtxEnterpriseId(ctx context.Context) int64 {
	return CtxInt(ctx, CtxEnterpriseIdKey)
}

// CtxUname 从上下文中获取用户名
func CtxUname(ctx context.Context) string {
	return CtxString(ctx, CtxUnameKey)
}
