package main

import (
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/n0rad/memguarded"
	"github.com/oklog/run"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"
)

var Version = ""
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

	socket := flag.String("socket", "/tmp/"+app+".sock", "socket path")
	confirm := flag.Bool("confirm", false, "confirm password")
	flag.Parse()

	switch os.Args[1] {
	case "version":
		fmt.Println(app)
		fmt.Println("version : ", Version)
		return nil
	case "get":
		return getPassword(*socket)
	case "set":
		return setPassword(*socket, *confirm)
	case "server":
		return startServer(*socket)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
	return nil
}

func startServer(socketPath string) error {
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
		SocketPath: socketPath,
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


func getPassword(socketPath string) error {
	passService := memguarded.Service{}
	passService.Init()
	go passService.Start()
	defer passService.Stop(nil)

	// connect
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return errs.WithEF(err, data.WithField("socketPath", socketPath), "Failed to connect to socketPath")
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return errs.WithE(err, "Failed to set deadline")
	}

	if err := memguarded.WriteBytes(conn, []byte("get_password\n")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := passService.FromConnection(conn); err != nil {
		return errs.WithE(err, "Failed to get password")
	}

	if err := passService.Write(os.Stdout); err != nil {
		return errs.WithE(err, "Failed to write password to stdin")
	}

	return nil
}

func setPassword(socketPath string, confirm bool) error {
	passService := memguarded.Service{}
	passService.Init()
	go passService.Start()
	defer passService.Stop(nil)

	if err := passService.FromStdin(confirm); err != nil {
		return errs.WithE(err, "Failed to ask password")
	}

	// connect
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return errs.WithEF(err, data.WithField("socketPath", socketPath), "Failed to connect to socketPath")
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return errs.WithE(err, "Failed to set deadline")
	}

	if err := memguarded.WriteBytes(conn, []byte("set_password ")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := passService.Write(conn); err != nil {
		return errs.WithE(err, "Failed to write key")
	}

	if err := memguarded.WriteBytes(conn, []byte{'\n'}); err != nil {
		return err
	}

	return nil
}

func homeDotConfigPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errs.WithE(err, "Failed to find user home folder")
	}
	return home + "/.config", nil
}

