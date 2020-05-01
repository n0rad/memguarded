package main

import (
	"github.com/n0rad/go-erlog/logs"
	"os"
	"os/signal"
	"syscall"
)

type SigtermService struct {
	stop chan struct{}
}

func (s *SigtermService) Init() {
	s.stop = make(chan struct{})
}

func (s SigtermService) Start() error {
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logs.Debug("Received SIGTERM, exiting gracefully...")
	case <-s.stop:
		break
	}
	return nil
}

func (s SigtermService) Stop(e error) {
	close(s.stop)
}
