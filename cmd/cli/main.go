package main

import (
	"flag"
	"fmt"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/n0rad/memguarded"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

var Version = ""
var SocketPath = "/tmp/" + app + ".sock"

var app = filepath.Base(os.Args[0])

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	if err := execute(); err != nil {
		logs.WithE(err).Fatal("Command failed")
	}
}

func execute() error {
	if len(os.Args) < 2 {
		return errs.WithF(data.WithField("commands", "get|set|server|version"), "command required")
	}

	flags := flag.NewFlagSet("command", flag.ExitOnError)
	socketPath := flags.String("socket", "/tmp/memguarded.sock", "socket path")
	clientKey := flags.String("client-key", "certs/client.key", "client key")
	clientPem := flags.String("client-pem", "certs/client.pem", "client pem")
	serverKey := flags.String("server-key", "certs/server.key", "server key")
	serverPem := flags.String("server-pem", "certs/server.pem", "server pem")
	debug := flags.Bool("debug", false, "debug")
	continueOnError := flags.Bool("continue-on-error", false, "Do not stop the server on any error")

	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}

	if *socketPath == "" {
		*socketPath = SocketPath
	}

	if *debug {
		logs.SetLevel(logs.TRACE)
	}

	if os.Args[1] == "version" {
		fmt.Println(app)
		fmt.Println("version : ", Version)
		return nil
	}

	certPassphrase := &memguarded.Service{}
	certPassphrase.Init()
	config := memguarded.CliConfig{
		CertPassphrase:       &memguarded.Service{},
		Secret:               &memguarded.Service{},
		StopOnAnyClientError: *continueOnError,
		SocketPath:           *socketPath,
		ClientKey:            *clientKey,
		ClientPem:            *clientPem,
		ServerKey:            *serverKey,
		ServerPem:            *serverPem,
	}

	switch os.Args[1] {
	case "get":
		return memguarded.GetSecret(config)
	case "set":
		return memguarded.SetSecret(config)
	case "server":
		return memguarded.StartServer(config)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
	return nil
}
