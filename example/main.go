package example

import (
	"github.com/siherrmann/queuerManager"
	"github.com/siherrmann/queuerManager/helper"
)

// main is the entry point of the manager service. It initializes the manager handler,
// sets up routes, and starts the Echo server with the port from the environment variable QUEUER_MANAGER_PORT.
func main() {
	queuerManager.ManagerServer(helper.GetEnvOrDefault("QUEUER_MANAGER_PORT", "3000"), 1)
}
