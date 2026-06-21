package middleware

import (
	"net/http"

	"github.com/siherrmann/queuerManager/handler"
	"github.com/labstack/echo/v5"
)

func (r Middleware) CsrfMiddleware() echo.MiddlewareFunc {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(handler.HandleCSRFErrorView))

	_ = cop.AddTrustedOrigin("http://localhost:3000")
	_ = cop.AddTrustedOrigin("http://127.0.0.1:3000")

	return echo.WrapMiddleware(cop.Handler)
}
