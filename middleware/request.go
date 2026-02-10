package middleware

import (
	"github.com/siherrmann/queuerManager/model"

	"github.com/labstack/echo/v4"
)

func (r *Middleware) RequestContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		rc := model.GetRequestContext(c)

		rc.Url = c.Request().URL.Path
		rc.HxRequest = c.Request().Header.Get("hx-request") == "true"

		model.SetRequestContext(c, rc)

		return next(c)
	}
}
