package logrustash

import (
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

const defaultBufSize uint = 8192

// Hook represents a logrus hook for Logstash.
// To initialize it use the `New` function.
type Hook struct {
	writer    io.Writer
	formatter logrus.Formatter
	levels    []logrus.Level
	async     bool
	buf       chan *logrus.Entry
	wg        sync.WaitGroup
	mu        sync.RWMutex
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
		async:     false,
	}
}

// Fire takes, formats and sends the entry to Logstash.
// Hook's formatter is used to format the entry into Logstash format
// and Hook's writer is used to write the formatted entry to the Logstash instance.
func (h *Hook) Fire(entry *logrus.Entry) error {
	h.mu.RLock() // Claim the mutex as a RLock - allowing multiple go routines to log simultaneously
	defer h.mu.RUnlock()

	if !h.async {
		return h.fire(entry)
	}

	// send log asynchroniously and return no error.

	// if a buffering is enabled push the entry to the buffer
	// and process using a background process
	if h.buf != nil {
		h.wg.Add(1)
		h.buf <- entry
	} else {
		// otherwise no buffer so just process the request in a background process
		go h.fire(entry)
	}
	return nil
}

// Levels returns all logrus levels.
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// SetLevels sets logging level to fire this hook.
func (h *Hook) SetLevels(levels []logrus.Level) {
	h.levels = levels
}

// Async sets async flag and send log asynchroniously.
// If use this option, Fire() does not return error.
func (h *Hook) Async() {
	h.async = true
}

// AsyncBuffer creates a buffer for log entries and starts a
// background process to handle processing the buffer entries.
func (h *Hook) AsyncBuffer(bufsize uint) {
	bsize := bufsize
	if bsize <= 0 {
		bsize = defaultBufSize
	}

	h.Async()
	h.buf = make(chan *logrus.Entry, bsize)
	go h.processBuffer() // Log in background
}

// Flush waits for the log queue to be empty.
func (h *Hook) Flush() {
	if !h.async || h.buf == nil {
		return
	}
	h.mu.Lock() // claim the mutex as a Lock - we want exclusive access to it
	defer h.mu.Unlock()

	h.wg.Wait()
}

func (h *Hook) processBuffer() {
	for {
		entry := <-h.buf // receive new entry on channel
		if err := h.fire(entry); err != nil {
			fmt.Printf("Error during sending message to logstash: %v\n", err)
		}
		h.wg.Done()
	}
}

func (h *Hook) fire(entry *logrus.Entry) error {
	dataBytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.writer.Write(dataBytes)
	return err
}
