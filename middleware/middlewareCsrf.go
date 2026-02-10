package middleware

import (
	"net/http"

	"github.com/siherrmann/queuerManager/handler"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v5"
)

func (r Middleware) CsrfMiddleware() echo.MiddlewareFunc {
	// TODO remove csrf.Secure(false) in production
	csrfMiddleware := csrf.Protect(
		r.csrfKey,
		csrf.Path("/"),
		csrf.Secure(false),
		csrf.SameSite(csrf.SameSiteLaxMode), // Set to Lax instead of default Strict
		csrf.ErrorHandler(http.HandlerFunc(handler.HandleCSRFErrorView)),
		csrf.TrustedOrigins([]string{"localhost:3000", "127.0.0.1:3000"}),
	)
	return echo.WrapMiddleware(csrfMiddleware)
}
