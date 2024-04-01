package logger

import "github.com/sirupsen/logrus"

var logger *logrus.Logger = NewLogger()

func GetLogger() *logrus.Logger {
	return logger
}

func SetLogger(newLogger *logrus.Logger) {
	logger = newLogger
}

func NewLogger() *logrus.Logger {
	logging := logrus.New()
	logging.SetFormatter(&logrus.JSONFormatter{})
	logging.SetLevel(logrus.InfoLevel)
	return logging
}
