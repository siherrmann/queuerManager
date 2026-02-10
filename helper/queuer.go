package helper

import "github.com/siherrmann/queuer"

var Queuer *queuer.Queuer

func InitQueuer(maxConcurrency int) {
	Queuer = queuer.NewQueuer("manager-server", maxConcurrency)
}
