package loggers

import (
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug})),
	}
}

type DataLoggerHandler struct {
	Msg         string
	Status      int
	Method      string
	Path        string
	UserUUID    string
	Errors      string
	DataRequest map[string]any
}

func (l *Logger) LoggerHandler(data *DataLoggerHandler) {
	if data.Status >= 200 && data.Status < 400 {
		l.Logger.Info(data.Msg,
			slog.Int("status", data.Status),
			slog.String("method", data.Method),
			slog.String("path", data.Path),
			slog.String("user_uuid", data.UserUUID))
	} else if data.Status >= 400 && data.Status < 500 {
		l.Logger.Info(data.Msg,
			slog.Int("status", data.Status),
			slog.String("method", data.Method),
			slog.String("path", data.Path),
			slog.String("user_uuid", data.UserUUID),
			slog.Any("errors", data.Errors),
			slog.Any("data", data.DataRequest))
	} else {
		l.Logger.Error(data.Msg,
			slog.Int("status", data.Status),
			slog.String("method", data.Method),
			slog.String("path", data.Path),
			slog.String("user_uuid", data.UserUUID),
			slog.Any("errors", data.Errors),
			slog.Any("data", data.DataRequest))
	}
}
