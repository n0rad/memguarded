package memguarded

import (
	"github.com/n0rad/go-erlog/errs"
	"github.com/oklog/run"
	"os"
)

type CliConfig struct {
	SocketPath     string
	CertPassphrase *Service
	Secret         *Service
	ClientKey      string
	ClientPem      string
	ServerKey      string
	ServerPem      string
	CaPem          string

	// server only
	StopOnAnyClientError bool
}

func StartServer(config CliConfig) error {
	var g run.Group

	// sigterm
	sigterm := SigtermService{}
	sigterm.Init()
	g.Add(sigterm.Start, sigterm.Stop)

	// Cert passphrase
	//config.CertPassphrase.Init()
	//g.Add(config.CertPassphrase.Start, config.CertPassphrase.Stop)

	// secret
	config.Secret.Init()
	g.Add(config.Secret.Start, config.Secret.Stop)

	// socket
	socketServer := Server{
		CertKey:              config.ServerKey,
		CertPem:              config.ServerPem,
		CAPem: 				  config.CaPem,
		SocketPath:           config.SocketPath,
		StopOnAnyClientError: config.StopOnAnyClientError,
	}

	if err := socketServer.Init(config.Secret); err != nil {
		return err
	}
	g.Add(socketServer.Start, socketServer.Stop)

	// start services
	if err := g.Run(); err != nil {
		return err
	}

	return nil
}

func GetSecret(config CliConfig) error {
	//cert passphrase
	config.CertPassphrase.Init()
	go config.CertPassphrase.Start()
	defer config.CertPassphrase.Stop(nil)
	if err := config.CertPassphrase.AskSecret(false, "Cert passphrase"); err != nil {
		return errs.WithE(err, "Failed to ask passphrase")
	}

	client := Client{
		CertPem:        config.ClientPem,
		CertKey:        config.ClientKey,
		SocketPath:     config.SocketPath,
		CertPassphrase: config.CertPassphrase,
	}
	if err := client.Connect(); err != nil {
		return err
	}

	if err := client.GetSecret(config.Secret); err != nil {
		return err
	}

	if err := config.Secret.Write(os.Stdout); err != nil {
		return errs.WithE(err, "Failed to write password to stdin")
	}

	return nil
}

func SetSecret(config CliConfig) error {
	//cert passphrase
	config.CertPassphrase.Init()
	go config.CertPassphrase.Start()
	defer config.CertPassphrase.Stop(nil)
	if err := config.CertPassphrase.AskSecret(false, "Cert passphrase"); err != nil {
		return errs.WithE(err, "Failed to ask passphrase")
	}

	// secret
	config.Secret.Init()
	go config.Secret.Start()
	defer config.Secret.Stop(nil)
	if err := config.Secret.AskSecret(false, "Secret"); err != nil {
		return errs.WithE(err, "Failed to ask secret")
	}

	client := Client{
		CertPem:        config.ClientPem,
		CertKey:        config.ClientKey,
		SocketPath:     config.SocketPath,
		CertPassphrase: config.CertPassphrase,
	}
	if err := client.Connect(); err != nil {
		return err
	}

	return client.SetSecret(config.Secret)
}
