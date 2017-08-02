package logrustash

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Hook represents a logrus hook for Logstash.
// To initialize it use the `New` function.
type Hook struct {
	writer    io.Writer
	formatter logrus.Formatter
	levels    []logrus.Level
}

// New returns a new logrus.Hook for Logstash.
//
// To create a new hook that sends logs to `tcp://logstash.corp.io:9999`:
//
// conn, _ := net.Dial("tcp", "logstash.corp.io:9999")
// hook := logrustash.New(conn, logrustash.DefaultFormatter())
func New(w io.Writer, f logrus.Formatter) *Hook {
	return &Hook{
		writer:    w,
		formatter: f,
		levels:    logrus.AllLevels,
	}
}

// Fire takes, formats and sends the entry to Logstash.
// Hook's formatter is used to format the entry into Logstash format
// and Hook's writer is used to write the formatted entry to the Logstash instance.
func (h *Hook) Fire(entry *logrus.Entry) error {
	return h.fire(entry)
}

// Levels returns all logrus levels.
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// SetLevels sets logging level to fire this hook.
func (h *Hook) SetLevels(levels []logrus.Level) {
	h.levels = levels
}

func (h *Hook) fire(entry *logrus.Entry) error {
	dataBytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.writer.Write(dataBytes)
	return err
}
