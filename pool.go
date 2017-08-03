package logrustash

import (
	"net"
	"time"

	"github.com/bitly/go-hostpool"
	"gopkg.in/fatih/pool.v2"
)

// maxRetries is temporary until it is made configurable
const maxRetries = 3
const connectTimeOut = 3 // seconds

type logstashPool struct {
	net.Conn
	hosts   hostpool.HostPool
	conns   pool.Pool
	timeout time.Time
}

func newPool(hosts []string, initialCap, maxCap int) (*logstashPool, error) {
	hpool := hostpool.New(hosts)
	factory := makeFactory(hpool, len(hosts))
	conns, err := pool.NewChannelPool(initialCap, maxCap, factory)
	if err != nil {
		return nil, err
	}
	return &logstashPool{
		hosts: hpool,
		conns: conns,
	}, nil
}

func makeFactory(hosts hostpool.HostPool, totalHosts int) pool.Factory {
	return func() (net.Conn, error) {
		var conn net.Conn
		var err error
		attempts := 0
		for conn == nil && attempts < totalHosts {
			attempts++
			hostresp := hosts.Get()
			conn, err = net.DialTimeout("tcp", hostresp.Host(), time.Duration(connectTimeOut)*time.Second)
			if err != nil {
				hostresp.Mark(err)
			}
		}
		return conn, err
	}
}

func (p *logstashPool) SetWriteDeadline(t time.Time) error {
	p.timeout = t
	return nil
}

func (p *logstashPool) Close() error {
	p.conns.Close()
	p.hosts.Close()
	return nil
}

func (p *logstashPool) Write(data []byte) (n int, err error) {
	return p.retry(func() (int, error) {
		return p.write(data)
	}, maxRetries)
}

func (p *logstashPool) write(data []byte) (n int, err error) {
	conn, err := p.conns.Get()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if p.timeout.After(time.Now()) {
		_ = conn.SetWriteDeadline(p.timeout)
	}

	n, err = conn.Write(data)
	if netErr, ok := err.(net.Error); ok {
		if netErr.Temporary() || netErr.Timeout() {
			if pcon, pok := conn.(*pool.PoolConn); pok {
				pcon.MarkUnusable()
			}
		}
	}

	return n, err
}

func (p *logstashPool) retry(action func() (int, error), retriesLeft int) (n int, err error) {
	n, err = action()
	if err != nil && retriesLeft > 0 {
		return p.retry(action, retriesLeft-1)
	}

	return n, err
}
