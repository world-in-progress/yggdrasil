package logger

import "github.com/sirupsen/logrus"

var log = logrus.New()

func Info(format string, args ...any) {
	log.Infof(format, args...)
}

func Error(format string, args ...any) {
	log.Errorf(format, args...)
}

func Fatal(format string, args ...any) {
	log.Fatalf(format, args...)
}

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
}
