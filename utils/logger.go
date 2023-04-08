package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/gommon/color"
	"github.com/sirupsen/logrus"
)

// Override for testing
var osHostname = os.Hostname

// ILogger is an interface of genericLogger
type ILoggerUtils interface {
	LogIt(severity, message string, fields map[string]interface{})
	SetModule(name string)
	SetOperation(name string)
	GetHostname() (string, error)
	SetHostname(func() (string, error))
}

var (
	rootLogger *logrus.Logger
)

type PlainFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}

func (f *PlainFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	severity := colorLog(f.LevelDesc[entry.Level], f.LevelDesc[entry.Level])
	date := colorLog(f.LevelDesc[entry.Level], entry.Time.Format(f.TimestampFormat))
	message := colorLog(f.LevelDesc[entry.Level], entry.Message)

	formatter := fmt.Sprintf("%s %s %s\n", severity, date, message)
	return []byte(formatter), nil
}

func initLogger() *logrus.Logger {
	rootLogger = logrus.New()
	color.Enable()

	if os.Getenv("ENVIRONMENT") == "DEV" {
		plainFormatter := new(PlainFormatter)
		plainFormatter.TimestampFormat = "02/01/2006 15:04:05"
		plainFormatter.LevelDesc = []string{"PANIC", "FAIL", "ERROR", "WARN", "INFO", "DEBUG"}
		rootLogger.SetFormatter(plainFormatter)
	} else {
		rootLogger.SetNoLock()
		rootLogger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339}
	}

	rootLogger.SetLevel(getLogLevel("LOGRUS_LOG_LEVEL"))
	return rootLogger
}

// GenericLogger represents log struct
type genericLogger struct {
	rootLogger    *logrus.Logger
	Log           *logrus.Entry
	Hostname      string
	Module        string
	OperationName string
}

// NewGenericLogger create a new genericlogger
func NewGenericLogger() ILoggerUtils {
	g := &genericLogger{}
	hostname := "unknown"
	hostname, _ = g.GetHostname()
	g.Hostname = hostname
	if g.rootLogger == nil {
		g.rootLogger = initLogger()
	}
	g.Log = rootLogger.WithFields(logrus.Fields{})
	return g
}

func (g *genericLogger) SetModule(name string) {
	g.Module = name
}

func (g *genericLogger) SetOperation(name string) {
	g.OperationName = name
}

func (g *genericLogger) SetHostname(hostname func() (string, error)) {
	osHostname = hostname
}

func (g *genericLogger) GetHostname() (string, error) {
	host, err := osHostname()
	if err != nil {
		return "unknown", err
	}
	return host, nil
}

// LogIt log a new message to stdout
func (g *genericLogger) LogIt(severity, message string, fields map[string]interface{}) {
	logger := g.Log
	logger = logger.WithFields(logrus.Fields{})
	if fields != nil {
		logger = logger.WithFields(fields)
	}

	logType := os.Getenv("LOGRUS_LOG_LEVEL")

	switch severity {
	case "DEBUG":
		if logType == "DEBUG" {
			logger.Debug(message)
		}
	case "INFO":
		if logType == "DEBUG" || logType == "INFO" {
			logger.Info(message)
		}
	case "WARN":
		logger.Warn(message)
	case "ERROR":
		logger.Error(message)
	default:
		if logType == "INFO" {
			logger.Info(message)
		}
	}
}

func getLogLevel(envVariable string) logrus.Level {
	switch os.Getenv(envVariable) {
	case "DEBUG":
		return logrus.DebugLevel
	case "INFO":
		return logrus.InfoLevel
	case "WARN":
		return logrus.WarnLevel
	case "ERROR":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func colorLog(severity, message string) string {
	message = strings.TrimSuffix(message, "\n")
	formattedMessage := fmt.Sprintf("[%v]", message)

	if os.Getenv("ENVIRONMENT") != "DEV" {
		return fmt.Sprint(formattedMessage)
	}

	switch severity {
	case "DEBUG":
		return fmt.Sprint(color.Cyan(formattedMessage))
	case "WARN":
		return fmt.Sprint(color.Yellow(formattedMessage))
	case "ERROR":
		return fmt.Sprint(color.Red(formattedMessage))
	default:
		return fmt.Sprint(color.Green(formattedMessage))
	}
}
