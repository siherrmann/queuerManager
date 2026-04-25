package main

import (
	"context"
	"log"
	"os"

	queuerHelper "github.com/siherrmann/queuer/helper"
	"github.com/siherrmann/queuerManager"
	"github.com/siherrmann/queuerManager/helper"
)

// main is the entry point of the manager service. It initializes the manager handler,
// sets up routes, and starts the Echo server with the port from the environment variable QUEUER_MANAGER_PORT.
func main() {
	// Start Postgres container
	teardown, dbPort, err := queuerHelper.MustStartPostgresContainer()
	if err != nil {
		log.Fatalf("error starting postgres container: %v", err)
	}
	defer func() {
		if err := teardown(context.Background()); err != nil {
			log.Printf("error tearing down postgres container: %v", err)
		}
	}()

	// Set database configuration env vars manually since we are not in a *testing.T context
	os.Setenv("QUEUER_DB_HOST", "localhost")
	os.Setenv("QUEUER_DB_PORT", dbPort)
	os.Setenv("QUEUER_DB_DATABASE", "database") // matches helper.dbName
	os.Setenv("QUEUER_DB_USERNAME", "user")     // matches helper.dbUser
	os.Setenv("QUEUER_DB_PASSWORD", "password") // matches helper.dbPwd
	os.Setenv("QUEUER_DB_SCHEMA", "public")
	os.Setenv("QUEUER_DB_SSLMODE", "disable")

	app := queuerManager.NewManagerApp(helper.GetEnvOrDefault("QUEUER_MANAGER_PORT", "3000"), 1)
	app.StaticDir = "../view/static"

	// Register the custom example extension
	app.RegisterExtension(&ExampleExtension{})

	// Start the application
	app.Start()
}
