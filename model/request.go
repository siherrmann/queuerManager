package model

import (
	"context"

	"github.com/labstack/echo/v5"
)

type ContextKey string

const REQUEST_CONTEXT_KEY ContextKey = "request_context"

type RequestContext struct {
	Url       string `json:"url"`
	HxRequest bool   `json:"hx_request"`
}

func SetRequestContext(c *echo.Context, value any) {
	ctx := context.WithValue(c.Request().Context(), REQUEST_CONTEXT_KEY, value)
	c.SetRequest(c.Request().WithContext(ctx))
}

func GetRequestContext(c interface{}) RequestContext {
	var ctx context.Context
	if goCtx, ok := c.(context.Context); ok {
		ctx = goCtx
	} else if echoCtx, ok := c.(*echo.Context); ok {
		ctx = echoCtx.Request().Context()
	} else {
		panic("invalid context, must be echo.Context or context.Context")
	}
	value, ok := ctx.Value(REQUEST_CONTEXT_KEY).(RequestContext)
	if !ok {
		return RequestContext{}
	}
	return value
}
