package handler

import (
	"context"
	"fmt"
	"manager/view/components"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func render(ctx echo.Context, t templ.Component, status ...int) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	if len(status) > 0 {
		return ctx.HTML(status[0], buf.String())
	}
	return ctx.HTML(http.StatusOK, buf.String())
}

func renderPopup(c echo.Context, component templ.Component) error {
	c.Response().Header().Add("HX-Retarget", "#body")
	c.Response().Header().Add("HX-Reswap", "beforeend")
	return render(c, component)
}

func renderHTTP(writer http.ResponseWriter, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(context.Background(), buf); err != nil {
		return err
	}

	writer.WriteHeader(http.StatusOK)
	fmt.Fprint(writer, buf.String())
	return nil
}

func renderPopupHTTP(writer http.ResponseWriter, component templ.Component) error {
	writer.Header().Add("HX-Retarget", "#body")
	writer.Header().Add("HX-Reswap", "beforeend")
	return renderHTTP(writer, component)
}

func renderPopupOrJson(c echo.Context, status int, value ...any) error {
	// No value to render
	if len(value) == 0 {
		return c.NoContent(status)
	}

	// If HTMX request, render popup
	if c.Request().Header.Get("HX-Request") != "" {
		messageStr := ""
		if messageTemp, ok := value[0].(string); ok {
			messageStr = messageTemp
		} else {
			messageStr = fmt.Sprintf("%v", value[0])
		}

		if status >= 200 && status < 300 {
			return renderPopup(c, components.PopupSuccess("Info", messageStr))
		} else {
			return renderPopup(c, components.PopupError("Error", messageStr))
		}
	}

	// Otherwise, return JSON with message if first value is string
	values := map[string]any{}
	for i, v := range value {
		if message, ok := value[0].(string); ok && i == 0 {
			values["message"] = message
		} else {
			values["value"] = v
		}
	}

	return c.JSON(status, value)
}
