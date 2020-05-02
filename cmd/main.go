package main

import (
	"flag"
	"fmt"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/n0rad/memguarded"
	"github.com/oklog/run"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

var Version = ""
var SocketPath = "/tmp/" + app + ".sock"
var SocketPassword = ""

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

	flagger := flag.NewFlagSet("command", flag.ExitOnError)
	socketPath := flagger.String("socket", "", "socket path")
	socketPassword := flagger.String("password", "", "socket password")
	confirm := flagger.Bool("confirm", false, "confirm password")
	debug := flagger.Bool("debug", false, "debug")
	continueOnError := flagger.Bool("continue-on-error", false, "Do not stop the server on any error")

	if err := flagger.Parse(os.Args[2:]); err != nil {
		return err
	}

	if *socketPassword == "" {
		*socketPassword = SocketPassword
	}

	if *socketPath == "" {
		*socketPath = SocketPath
	}

	if *debug {
		logs.SetLevel(logs.TRACE)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(app)
		fmt.Println("version : ", Version)
		return nil
	case "get":
		return getSecret(*socketPath, *socketPassword)
	case "set":
		return setSecret(*socketPath, *socketPassword, *confirm)
	case "server":
		return startServer(*socketPath, *socketPassword, *continueOnError)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
	return nil
}

func startServer(socketPath string, socketPassword string, continueOnError bool) error {
	var g run.Group

	// sigterm
	sigterm := SigtermService{}
	sigterm.Init()
	g.Add(sigterm.Start, sigterm.Stop)

	// password
	passService := memguarded.Service{}
	passService.Init()
	g.Add(passService.Start, passService.Stop)

	// socket
	socketServer := memguarded.Server{
		SocketPath:                   socketPath,
		AnyClientErrorCloseTheServer: !continueOnError,
	}
	if err := socketServer.Init(&passService); err != nil {
		return err
	}
	g.Add(socketServer.Start, socketServer.Stop)

	// start services
	if err := g.Run(); err != nil {
		return err
	}

	logs.Trace("Bye !")

	return nil
}

func getSecret(socketPath string, socketPassword string) error {
	secretService := memguarded.Service{}
	secretService.Init()
	go secretService.Start()
	defer secretService.Stop(nil)

	client := memguarded.Client{
		SocketPath:     socketPath,
		SocketPassword: socketPassword,
	}
	if err := client.Connect(); err != nil {
		return err
	}

	if err := client.GetSecret(&secretService); err != nil {
		return err
	}

	if err := secretService.Write(os.Stdout); err != nil {
		return errs.WithE(err, "Failed to write password to stdin")
	}

	return nil
}

func setSecret(socketPath string, socketPassword string, confirm bool) error {
	secretService := memguarded.Service{}
	secretService.Init()
	go secretService.Start()
	defer secretService.Stop(nil)

	if err := secretService.FromStdin(confirm); err != nil {
		return errs.WithE(err, "Failed to ask password")
	}

	client := memguarded.Client{
		SocketPath:     socketPath,
		SocketPassword: socketPassword,
	}
	if err := client.Connect(); err != nil {
		return err
	}

	return client.SetSecret(&secretService)
}
