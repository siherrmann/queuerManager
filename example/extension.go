package main

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuerManager/handler"
	"github.com/siherrmann/queuerManager/model"
)

type ExampleExtension struct{}

func (e *ExampleExtension) Init(ctx context.Context, mh *handler.ManagerHandler) error {
	// Dummy initialization
	return nil
}

func (e *ExampleExtension) SetupRoutes(echoApp *echo.Echo, mh *handler.ManagerHandler) {
	echoApp.GET("/custom", func(c *echo.Context) error {
		return CustomView().Render(c.Request().Context(), c.Response())
	})
	
	api := echoApp.Group("/api/custom")
	api.GET("/data", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Custom API works!"})
	})
}

func (e *ExampleExtension) SidebarItems() []model.SidebarItem {
	return []model.SidebarItem{
		{
			Title:        "Custom View",
			MaterialIcon: "star",
			Href:         "/custom",
		},
	}
}
