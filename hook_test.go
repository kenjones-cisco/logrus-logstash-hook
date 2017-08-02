package logrustash

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

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

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}

type failWrite struct{}

func (w failWrite) Write(d []byte) (int, error) {
	return 0, errors.New("")
}

func TestFireWriteError(t *testing.T) {
	h := New(failWrite{}, &logrus.JSONFormatter{})

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}
