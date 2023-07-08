package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = newLogger()

func newLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	return log
}
