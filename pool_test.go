package logrustash

import (
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

var (
	initCap   = 3
	maxCap    = 5
	network   = "tcp"
	address   = "127.0.0.1:7777"
	srvBuffer *serverBuffer
)

func init() {
	srvBuffer = &serverBuffer{}

	// used for factory function
	go simpleTCPServer()
	time.Sleep(time.Millisecond * 300) // wait until tcp server has been settled

	rand.Seed(time.Now().UTC().UnixNano())
}

type serverBuffer struct {
	buf []byte
	err error
}

func (s *serverBuffer) GetData() []byte {
	return s.buf
}

func (s *serverBuffer) String() string {
	return string(s.buf)
}

func (s *serverBuffer) Error() string {
	return s.err.Error()
}

func simpleTCPServer() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			buf := make([]byte, 256)
			n, err := conn.Read(buf)
			if err != nil {
				srvBuffer.err = err
				srvBuffer.buf = []byte{}
			} else {
				srvBuffer.buf = buf[:n]
				srvBuffer.err = nil
			}
		}()
	}
}

func TestNewPool(t *testing.T) {
	hosts := []string{address}
	pool, err := newPool(hosts, initCap, maxCap)
	if err != nil {
		t.Errorf("newPool error: %s", err)
	}
	defer pool.Close()
}

func TestNewPoolError_BadRange(t *testing.T) {
	hosts := []string{address}
	_, err := newPool(hosts, initCap, 2)
	if err == nil {
		t.Errorf("newPool expected an error")
	}
	expected := "invalid capacity settings"
	if expected != err.Error() {
		t.Errorf("expected to see '%s' in '%s'", expected, err.Error())
	}
}

func TestNewPoolError_BadAddress(t *testing.T) {
	hosts := []string{"127.0.0.1:7778"}
	_, err := newPool(hosts, initCap, maxCap)
	if err == nil {
		t.Errorf("newPool expected an error")
	}
	expected := "factory is not able to fill the pool: dial tcp 127.0.0.1:7778: getsockopt: connection refused"
	if expected != err.Error() {
		t.Errorf("expected to see '%s' in '%s'", expected, err.Error())
	}
}

func TestWrite(t *testing.T) {
	hosts := []string{address}
	pool, err := newPool(hosts, initCap, maxCap)
	if err != nil {
		t.Errorf("newPool error: %s", err)
	}
	defer pool.Close()

	data := []byte("sample data")
	n, werr := pool.Write(data)
	if werr != nil {
		t.Errorf("Write error: %s", werr)
	}
	if n != len(data) {
		t.Errorf("expected to see '%d' in '%d'", len(data), n)
	}

}

func TestWriteError(t *testing.T) {
	hosts := []string{address}
	pool, err := newPool(hosts, initCap, maxCap)
	if err != nil {
		t.Errorf("newPool error: %s", err)
	}
	defer pool.Close()

	pool.conns.Close()

	data := []byte("sample data")
	_, werr := pool.Write(data)
	if werr == nil {
		t.Errorf("Write expected error")
	}
	expected := "pool is closed"
	if expected != werr.Error() {
		t.Errorf("expected to see '%s' in '%s'", expected, werr.Error())
	}

}

func TestWriteTimeout(t *testing.T) {
	hosts := []string{address}
	pool, err := newPool(hosts, initCap, maxCap)
	if err != nil {
		t.Errorf("newPool error: %s", err)
	}
	defer pool.Close()

	ms := time.Duration(500) * time.Millisecond
	_ = pool.SetWriteDeadline(time.Now().Add(ms))

	data := []byte("sample data")
	n, werr := pool.Write(data)
	if werr != nil {
		t.Errorf("Write error: %s", werr)
	}
	if n != len(data) {
		t.Errorf("expected to see '%d' in '%d'", len(data), n)
	}
}
