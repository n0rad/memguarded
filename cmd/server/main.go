package main

import (
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/n0rad/memguarded"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// very simple daemon example server
func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	if err := execute(); err != nil {
		logs.WithE(err).Fatal("Command failed")
	}
}

func execute() error {
	app := filepath.Base(os.Args[0])
	config := memguarded.CliConfig{
		CertPassphrase:       &memguarded.Service{},
		Secret:               &memguarded.Service{},
		StopOnAnyClientError: true,
		SocketPath:           "/etc/" + app + "/" + app + ".sock",
		ServerKey:            "/etc/" + app + "/" + app + ".key",
		ServerPem:            "/etc/" + app + "/" + app + ".pem",
	}

	logs.Info("Start")
	return memguarded.StartServer(config)
}
