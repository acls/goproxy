package proxy

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type frontend struct {
	Name string
	*zap.Logger
	TLSConfig *tls.Config
	Listener  net.Listener
	Strategy  BackendStrategy
}

func (f *frontend) Run() {
	f.Info("Handling connections",
		zap.String("frontend", f.Name),
	)
	for {
		// accept next connection to this frontend
		conn, err := f.Listener.Accept()
		if err != nil {
			f.Error("Failed to accept new connection",
				zap.String("from", conn.RemoteAddr().String()),
				zap.Error(err),
			)
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					continue
				}
			}
			return
		}
		f.Debug("Accepted new connection",
			zap.String("name", f.Name),
			zap.String("from", conn.RemoteAddr().String()),
		)

		// proxy the connection to an backend
		go f.proxyConnection(conn)
	}
}

func (f *frontend) proxyConnection(c net.Conn) (err error) {
	// unwrap if tls cert/key was specified
	if f.TLSConfig != nil {
		c = tls.Server(c, f.TLSConfig)
	}

	// pick the backend
	backend := f.Strategy.NextBackend()

	// dial the backend
	upConn, err := net.DialTimeout("tcp", backend.Addr, time.Duration(backend.ConnectTimeout)*time.Millisecond)
	if err != nil {
		f.Error("Failed to dial backend connection",
			zap.String("backend", backend.Addr),
			zap.Error(err),
		)
		c.Close()
		return
	}
	f.Debug("Initiated new connection to backend",
		zap.String("from", upConn.LocalAddr().String()),
		zap.String("to", upConn.RemoteAddr().String()),
	)

	// join the connections
	f.joinConnections(c, upConn)
	return
}

func (f *frontend) joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	halfJoin := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		n, err := io.Copy(dst, src)
		f.Debug("Copy failed after N bytes",
			zap.String("from", src.RemoteAddr().String()),
			zap.String("to", dst.RemoteAddr().String()),
			zap.Int64("bytes", n),
			zap.Error(err),
		)
	}

	f.Info("Joining connections",
		zap.String("from", c1.RemoteAddr().String()),
		zap.String("to", c2.RemoteAddr().String()),
	)
	wg.Add(2)
	go halfJoin(c1, c2)
	go halfJoin(c2, c1)
	wg.Wait()
}
