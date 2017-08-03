package logrustash

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type simpleFmter struct{}

func (f simpleFmter) Format(e *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("msg: %#v", e.Message)), nil
}

func TestFire(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := New(buffer, simpleFmter{})

	entry := &logrus.Entry{
		Message: "my message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	h.Flush() // does nothing; should result in immediate return

	expected := "msg: \"my message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

type failFmt struct{}

func (f failFmt) Format(e *logrus.Entry) ([]byte, error) {
	return nil, errors.New("")
}

func TestFireFormatError(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := New(buffer, failFmt{})

	defer h.Flush() // does nothing; should result in immediate return

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}

type failWrite struct{}

func (w failWrite) Write(d []byte) (int, error) {
	return 0, errors.New("failed to write")
}

func TestFireWriteError(t *testing.T) {
	h := New(failWrite{}, &logrus.JSONFormatter{})

	defer h.Flush() // does nothing; should result in immediate return

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}

func TestFireWriteErrorBufferAsync(t *testing.T) {
	h := New(failWrite{}, &logrus.JSONFormatter{})
	h.AsyncBuffer(10)

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err != nil {
		t.Error("unexpected error when in async mode")
	}

	h.Flush()
	// Output:
	// Error during sending message to logstash: failed to write

}

func TestFireAsync(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := New(buffer, simpleFmter{})
	h.Async()

	defer h.Flush() // does nothing; should result in immediate return

	entry := &logrus.Entry{
		Message: "my async message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	time.Sleep(100 * time.Millisecond)

	expected := "msg: \"my async message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

func TestFireAsyncBuffer(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := New(buffer, simpleFmter{})
	h.AsyncBuffer(0)

	entry := &logrus.Entry{
		Message: "my async message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	// wait for the buffer to be processed
	h.Flush()

	expected := "msg: \"my async message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

func TestUsePool(t *testing.T) {
	h := New(nil, simpleFmter{})
	hosts := []string{address}
	err := h.UsePool(hosts, initCap, maxCap)
	if err != nil {
		t.Error("expected UsePool to not return error")
	}
}

func TestUsePoolError(t *testing.T) {
	h := New(nil, simpleFmter{})
	hosts := []string{"127.0.0.1:7778"}
	err := h.UsePool(hosts, initCap, maxCap)
	if err == nil {
		t.Error("expected UsePool to return error")
	}
	expected := "factory is not able to fill the pool: dial tcp 127.0.0.1:7778: getsockopt: connection refused"
	if expected != err.Error() {
		t.Errorf("expected to see '%s' in '%s'", expected, err.Error())
	}
}

func TestSetTimeout_Ignored(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := New(buffer, simpleFmter{})
	h.SetTimeout(time.Duration(1) * time.Second)

	entry := &logrus.Entry{
		Message: "my message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	h.Flush() // does nothing; should result in immediate return

	expected := "msg: \"my message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

func TestSetTimeout_Used(t *testing.T) {
	conn, err := net.Dial(network, address)
	if err != nil {
		t.Error("Expected successful connection to simpleTCPServer")
	}
	h := New(conn, simpleFmter{})
	h.SetTimeout(time.Duration(1) * time.Second)

	entry := &logrus.Entry{
		Message: "my message with timeout",
		Data:    logrus.Fields{},
	}

	err = h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	time.Sleep(100 * time.Millisecond)

	expected := "msg: \"my message with timeout\""
	got := srvBuffer.String()
	if got != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, got)
	}
}

func TestSetTimeout_ConnectionPool(t *testing.T) {
	h := New(nil, simpleFmter{})
	hosts := []string{address}
	err := h.UsePool(hosts, initCap, maxCap)
	if err != nil {
		t.Error("expected UsePool to not return error")
	}
	h.SetTimeout(time.Duration(1) * time.Second)

	entry := &logrus.Entry{
		Message: "my message with timeout",
		Data:    logrus.Fields{},
	}

	err = h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	time.Sleep(100 * time.Millisecond)

	expected := "msg: \"my message with timeout\""
	got := srvBuffer.String()
	if got != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, got)
	}
}
