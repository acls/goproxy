package proxy

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"testing"

	"github.com/acls/goproxy/conf"
	"go.uber.org/zap"
)

var snakeoilCert = `-----BEGIN CERTIFICATE-----
MIICGTCCAYICCQCww5WxTI3a5jANBgkqhkiG9w0BAQUFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTEzMTIxOTExMDMzNloXDTQxMDUwNjExMDMzNlowXTELMAkG
A1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDEWMBQGA1UEAwwNKi5leGFtcGxlLmNvbTCBnzANBgkq
hkiG9w0BAQEFAAOBjQAwgYkCgYEArmBi147MNv5v+97eznwD2OTyCOToKV/IIOBM
qrSNu3iKASb817CoiPV9x9NmxdoLeVvVWHgGC9cBDo+j5fTPEdxQCE4Xm6KOUy0S
4/rJzxNniWFWusVgT4VbwWeNdEg22PM8uGKM9nrQ42UXdNsrXRWQdAxR966ZBCoG
xcwx4ZcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQBd4bS8qYe7vld2rgIOsNM5sqBk
mMcVCZPqUDX9axYQGGHkxF1qXv2ohnNvdmlVQtreuKF82HNL0P5uuU5jIms8fXPv
20TxAD7CbdR4dFn38mRHovprt9No3vtL8PmxhDOs7EOKtNyXplbVtmjf1N27UbQ3
K+MApaOowXqkoBSx9Q==
-----END CERTIFICATE-----`

var snakeoilKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCuYGLXjsw2/m/73t7OfAPY5PII5OgpX8gg4EyqtI27eIoBJvzX
sKiI9X3H02bF2gt5W9VYeAYL1wEOj6Pl9M8R3FAITheboo5TLRLj+snPE2eJYVa6
xWBPhVvBZ410SDbY8zy4Yoz2etDjZRd02ytdFZB0DFH3rpkEKgbFzDHhlwIDAQAB
AoGAWw7sLqJcE8+0TLOqZ+ss2yNbHLfkYE6rJDfc8TuN07rzXfytBjkzGSoQ/7tu
LJ1bZolFFIjAp4gj/iWWMewwAMfkoG3nT25z3Q8v+EPwO97kT5rgMW/sI9yamRhb
LQpENsaxF1UFW4ADxl32go2sPbYv/5hnMLB7bfR0vgZaFHkCQQDaAUgmKogKj0qb
BeuIftzLJWJ+uYYtUGpICF53LAbd/lUygnUx4fapcVQDTyHcpb1lRRRXuGfZn1x2
jn9KRC87AkEAzMSIpdZXXCigvEMWYi0laNV/AJjKKafBcq/l8VQcAq0FUhgeRCoB
FjSVJrngMwzu1cQC1Xwtp6Dh6+V4T51pVQJBALPQatpQKnXLSxYjA+tJ+IP3Cg7M
p8eolIFlpcVWIzPoHA3VXSUP5IxOVaWFF8EPU/C70dOo3r+5mmKPlp6DLxECQAxM
QWi0VsrSJdUosk9zJqwFJnuCsaGO0a9xoP29b3E5svgbOrYdT7NltQ9+Wli2jiGI
hCMOMi+/GdJxFaiya4ECQCabLUAE0YEZL0M4mrcALa4T0C2sKCW8Xo2wvbwDGc1Y
+GQErfiGNv0xDOWLYrqe40x71R8z4kZv4EKLH/7zjTE=
-----END RSA PRIVATE KEY-----`

const bindAddr = "127.0.0.1:55111"

func init() {
	loadTLSConfig = func(crtPath, keyPath string) (*tls.Config, error) {
		cert, err := tls.X509KeyPair([]byte(snakeoilCert), []byte(snakeoilKey))
		if err != nil {
			return nil, err
		}

		return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
	}
}

func init() {
	log, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(log)
}

func backendOrFail(t *testing.T) (net.Listener, string) {
	cfg, err := loadTLSConfig("", "")
	if err != nil {
		t.Fatalf("Failed to make snakeoil certificate: %v", err)
	}

	l, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	return l, fmt.Sprintf("127.0.0.1:%d", l.Addr().(*net.TCPAddr).Port)
}

func mkServer(t *testing.T, binding *conf.Binding) *Server {
	srv := &Server{
		Binding:   binding,
		Logger:    zap.L(),
		frontends: make(map[string]*frontend),
	}
	srv.Init()
	return srv
}

func TestSimple(t *testing.T) {
	l, addr := backendOrFail(t)

	f1 := &conf.Frontend{
		BoundAddr: bindAddr,
		Name:      "wrong",
		Backends: []conf.Backend{
			conf.Backend{
				Addr: addr,
			},
		},
	}

	s := mkServer(t, &conf.Binding{
		Secure:   true,
		BindAddr: bindAddr,
		Frontends: map[string]*conf.Frontend{
			f1.Name: f1,
		},
	})

	go s.Run()
	// wait for the listener to bind
	<-s.Ready()

	// test that replacing works
	s.RemoveFrontend(f1.Name)
	f1.Name = "test.example.com"
	s.ReplaceFrontend(f1)

	defer s.mux.Close()

	expected := []byte("Hello World")
	go func() {

		out, err := tls.Dial("tcp", bindAddr, &tls.Config{ServerName: "test.example.com", InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		out.Write(expected)
		out.Close()
	}()

	in, err := l.Accept()
	if err != nil {
		t.Fatalf("Failed to accept new connection: %v", err)
	}

	got, err := ioutil.ReadAll(in)
	if err != nil {
		t.Fatalf("Error reading data from connection: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Wrong data read from connection. Got %v, expected %v", got, expected)
	}
}

func TestMany(t *testing.T) {
	l1, addr1 := backendOrFail(t)
	l2, addr2 := backendOrFail(t)

	s := mkServer(t, &conf.Binding{
		Secure:   true,
		BindAddr: bindAddr,
		Frontends: map[string]*conf.Frontend{
			"test1.example.com": &conf.Frontend{
				BoundAddr: bindAddr,
				Name:      "test1.example.com",
				Backends: []conf.Backend{
					conf.Backend{
						Addr: addr1,
					},
				},
			},
			"test2.example.com": &conf.Frontend{
				BoundAddr: bindAddr,
				Name:      "test2.example.com",
				Backends: []conf.Backend{
					conf.Backend{
						Addr: addr2,
					},
				},
			},
		},
	})

	go s.Run()
	// wait for the listener to bind
	<-s.Ready()
	defer s.mux.Close()

	sendData := func(payload, name string) {
		out, err := tls.Dial("tcp", bindAddr, &tls.Config{ServerName: name, InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}

		out.Write([]byte(payload))
		out.Close()
	}

	check := func(l net.Listener, expected string) {
		in, err := l.Accept()
		if err != nil {
			t.Errorf("Failed to accept new connection: %v", err)
			return
		}

		got, err := ioutil.ReadAll(in)
		if err != nil {
			t.Errorf("Error reading data from connection: %v", err)
			return
		}

		if !reflect.DeepEqual(got, []byte(expected)) {
			t.Errorf("Wrong data read from connection. Got %v, expected %v", got, []byte(expected))
		}
	}

	go sendData("Hello 1", "test1.example.com")
	zap.L().Info("asfd")
	check(l1, "Hello 1")

	go sendData("Hello 2", "test2.example.com")
	check(l2, "Hello 2")
}

func TestHostNotFound(t *testing.T) {
	_, addr := backendOrFail(t)

	s := mkServer(t, &conf.Binding{
		Secure:   true,
		BindAddr: bindAddr,
		Frontends: map[string]*conf.Frontend{
			"test.example.com": &conf.Frontend{
				BoundAddr: bindAddr,
				Name:      "test.example.com",
				Backends: []conf.Backend{
					conf.Backend{
						Addr: addr,
					},
				},
			},
		},
	})

	go s.Run()
	<-s.Ready()
	defer s.mux.Close()

	_, err := tls.Dial("tcp", bindAddr, &tls.Config{ServerName: "foo.example.com", InsecureSkipVerify: true})
	if err == nil {
		t.Fatalf("Expected error when dialing wrong name, got nil")
	}
}

func TestRoundRobin(t *testing.T) {
	l1, addr1 := backendOrFail(t)
	l2, addr2 := backendOrFail(t)

	s := mkServer(t, &conf.Binding{
		Secure:   true,
		BindAddr: bindAddr,
		Frontends: map[string]*conf.Frontend{
			"test.example.com": &conf.Frontend{
				BoundAddr: bindAddr,
				Name:      "test.example.com",
				Backends: []conf.Backend{
					conf.Backend{
						Addr: addr1,
					},
					conf.Backend{
						Addr: addr2,
					},
				},
			},
		},
	})

	go s.Run()
	// wait for the listener to bind
	<-s.Ready()
	defer s.mux.Close()

	payload := "Hello world!"
	go func() {
		for i := 0; i < 20; i++ {
			out, err := tls.Dial("tcp", bindAddr, &tls.Config{ServerName: "test.example.com", InsecureSkipVerify: true})
			if err != nil {
				t.Fatalf("Failed to dial: %v", err)
			}

			out.Write([]byte(payload))
			out.Close()
		}
	}()

	var count1, count2 int

	var l net.Listener
	l = l1
	for i := 0; i < 20; i++ {
		// conections should switch off between backends
		if l == l1 {
			l = l2
		} else {
			l = l1
		}

		in, err := l.Accept()
		if err != nil {
			t.Errorf("Failed to accept new connection: %v", err)
			return
		}

		got, err := ioutil.ReadAll(in)
		if err != nil {
			t.Errorf("Error reading data from connection: %v", err)
			return
		}

		if !reflect.DeepEqual(got, []byte(payload)) {
			t.Errorf("Wrong data read from connection. Got %v, expected %v", got, []byte(payload))
			return
		}

		if l == l1 {
			count1++
		} else {
			count2++
		}
	}

	if count1 != 10 || count2 != 10 {
		t.Fatalf("Expected 10 connections to each backend, got: %v %v", count1, count2)
	}
}

// Check that we un
func TestTerminateTLS(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", l.Addr().(*net.TCPAddr).Port)

	s := mkServer(t, &conf.Binding{
		Secure:   true,
		BindAddr: bindAddr,
		Frontends: map[string]*conf.Frontend{
			"test.example.com": &conf.Frontend{
				TLSCrt:    "/snakeoil.crt",
				TLSKey:    "/snakeoil.key",
				BoundAddr: bindAddr,
				Name:      "test.example.com",
				Backends: []conf.Backend{
					conf.Backend{
						Addr: addr,
					},
				},
			},
		},
	})

	go s.Run()
	// wait for the listener to bind
	<-s.Ready()
	defer s.mux.Close()

	expected := []byte("Hello World")
	go func() {
		out, err := tls.Dial("tcp", bindAddr, &tls.Config{ServerName: "test.example.com", InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		out.Write(expected)
		out.Close()
	}()

	in, err := l.Accept()
	if err != nil {
		t.Fatalf("Failed to accept new connection: %v", err)
	}

	got, err := ioutil.ReadAll(in)
	if err != nil {
		t.Fatalf("Error reading data from connection: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Wrong data read from connection. Got %v, expected %v", got, expected)
	}
}
