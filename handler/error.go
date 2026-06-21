package handler

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/siherrmann/queuerManager/view/components"

	"github.com/labstack/echo/v5"
)

func HandleErrorView(err error, c *echo.Context) {
	code := http.StatusInternalServerError
	var message interface{}
	message = err.Error()
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = he.Message
	}
	c.Logger().Error(fmt.Sprintf("failed with code %d", code), slog.String("error", err.Error()))

	err = renderPopup(c, components.PopupError("Error", fmt.Sprint(message)))
	if err != nil {
		c.Logger().Error("failed to render error popup", slog.String("error", err.Error()))
	}
}

func HandleCSRFErrorView(w http.ResponseWriter, r *http.Request) {
	// #nosec G706 -- r.URL.EscapedPath() is used to prevent log injection/splitting
	log.Printf("CSRF/CrossOrigin error: request to %s rejected", r.URL.EscapedPath())
	err := renderPopupHTTP(w, components.PopupError("Error", "Security verification failed. Please reload the page."))
	if err != nil {
		log.Printf("Failed to render CSRF error popup: %v", err)
	}
}
