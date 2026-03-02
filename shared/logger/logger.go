package logger

import (
	"context"
	"encoding/json"
	"log"
	"order-billing-system/shared/requestid"
	"os"
	"time"
)

type LogLevel string

const (
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
)

type LogEntry struct {
	Level     LogLevel `json:"level"`
	Time      string   `json:"time"`
	Service   string   `json:"service"`
	Message   string   `json:"message"`
	Error     string   `json:"error,omitempty"`
	RequestID string   `json:"request_id,omitempty"`
}

var serviceName string

func Init(service string) {
	serviceName = service
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

func InfoCtx(ctx context.Context, msg string) {
	write(LogEntry{
		Level:     INFO,
		Time:      time.Now().UTC().Format(time.RFC3339),
		Service:   serviceName,
		Message:   msg,
		RequestID: requestid.Get(ctx),
	})
}

func ErrorCtx(ctx context.Context, msg string, err error) {
	write(LogEntry{
		Level:     ERROR,
		Time:      time.Now().UTC().Format(time.RFC3339),
		Service:   serviceName,
		Message:   msg,
		Error:     err.Error(),
		RequestID: requestid.Get(ctx),
	})
}

func write(entry LogEntry) {
	b, _ := json.Marshal(entry)
	log.Println(string(b))
}
