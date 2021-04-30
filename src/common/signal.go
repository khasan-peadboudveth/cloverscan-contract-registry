package common

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func WaitForSignal() {
	var stop = make(chan struct{})
	go func() {
		var sig = make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sig)
		<-sig
		log.Printf("got interrupt, shutting down...")
		stop <- struct{}{}
	}()
	<-stop
}
