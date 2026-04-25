package queuerManager

import (
	"context"

	"github.com/labstack/echo/v5"
	"github.com/siherrmann/queuerManager/handler"
	"github.com/siherrmann/queuerManager/model"
)

// Extension defines the interface that all queuerManager extensions must implement.
type Extension interface {
	// Init is called during the application startup, after the base handler and database are initialized.
	// Extensions can use this to initialize their own databases or models using the Queuer DB instance.
	Init(ctx context.Context, mh *handler.ManagerHandler) error

	// SetupRoutes is called after base routes are configured. Extensions can register custom API or view routes.
	SetupRoutes(e *echo.Echo, mh *handler.ManagerHandler)

	// SidebarItems returns a list of items that should be appended to the UI sidebar.
	SidebarItems() []model.SidebarItem
}
