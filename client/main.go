package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimuls/news-storage/client/web"
	"github.com/dimuls/news-storage/storage/nats"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.WithField("subsystem", "main")

	nc, err := nats.NewClient(os.Getenv("NATS_URL"),
		os.Getenv("SUBSCRIBE_SUBJECT"))
	if err != nil {
		log.WithError(err).Error("failed to create nats client")
	}

	ws := web.NewServer(nc, os.Getenv("BIND_ADDR"))
	ws.Start()

	log.Info("web server started")

	ss := make(chan os.Signal)

	signal.Notify(ss, syscall.SIGTERM)

	s := <-ss

	log.Infof("captured %v signal, stopping", s)

	st := time.Now()
	ws.Stop()
	nc.Close()
	et := time.Now()

	log.Infof("stopped in %g seconds, exiting",
		et.Sub(st).Seconds())
}
