package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimuls/news-storage/storage/nats"
	"github.com/dimuls/news-storage/storage/postgres"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.WithField("subsystem", "main")

	ps, err := postgres.NewStorage(os.Getenv("POSTGRES_URI"))
	if err != nil {
		log.WithError(err).Fatal("failed to create postgres storage")
	}

	defer func() {
		err = ps.Close()
		if err != nil {
			log.WithError(err).Error("failed to close storage")
		}
	}()

	err = ps.Migrate()
	if err != nil {
		log.WithError(err).Fatal("failed to migrate postgres storage")
	}

	ns := nats.NewServer(ps, os.Getenv("NATS_URL"),
		os.Getenv("SUBSCRIBE_SUBJECT"))

	err = ns.Start()
	if err != nil {
		logrus.WithError(err).Error("failed to start nats server")
	}

	log.Info("nats server started")

	ss := make(chan os.Signal)

	signal.Notify(ss, syscall.SIGTERM)

	s := <-ss

	log.Infof("captured %v signal, stopping", s)

	st := time.Now()
	ns.Stop()
	et := time.Now()

	log.Infof("stopped in %g seconds, exiting",
		et.Sub(st).Seconds())
}
