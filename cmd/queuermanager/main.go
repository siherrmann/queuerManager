package main

import (
	"github.com/siherrmann/queuerManager"
	"github.com/siherrmann/queuerManager/helper"
)

func main() {
	port := helper.GetEnvOrDefault("QUEUER_MANAGER_PORT", "3000")
	staticDir := helper.GetEnvOrDefault("QUEUER_STATIC_DIR", "./view/static")
	app := queuerManager.NewManagerApp(port, 1)
	app.StaticDir = staticDir

	// Start the application
	app.Start()
}
