package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New(level, format string) *Logger {
	log := logrus.New()

	// Set log level
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	// Set log format
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	log.SetOutput(os.Stdout)

	return &Logger{log}
}

// WithFields creates an entry with multiple fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithField creates an entry with a single field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithTenant adds tenant context to log entry
func (l *Logger) WithTenant(tenantID string) *logrus.Entry {
	return l.WithField("tenant_id", tenantID)
}

// WithUser adds user context to log entry
func (l *Logger) WithUser(userID string) *logrus.Entry {
	return l.WithField("user_id", userID)
}

// WithRequest adds request context to log entry
func (l *Logger) WithRequest(method, path, userAgent string) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"method":     method,
		"path":       path,
		"user_agent": userAgent,
	})
}

// WithError adds error context to log entry
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}