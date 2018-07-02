package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Configuration_ParseYaml(t *testing.T) {
	// must use spaces for yaml indentation
	input := `
"127.0.0.1:55111":
  secure: true
  frontends:
    test1.example.com:
      backends:
      - addr: :443
      tls_crt: /test1.crt
      tls_key: /test1.key
    test2.example.com:
      backends:
      - addr: :80
`
	got := NewConfiguration()
	err := got.ParseYaml([]byte(input))
	if err != nil {
		t.Errorf("Error parsing yaml config: %v", err)
		return
	}

	binding := &Binding{
		Secure:   true,
		BindAddr: "127.0.0.1:55111", // from parent map key
	}
	f1 := &Frontend{
		Name:      "test1.example.com", // from parent map key or filename
		BoundAddr: "127.0.0.1:55111",   // from parent
		TLSCrt:    "/test1.crt",
		TLSKey:    "/test1.key",
		Backends: []Backend{
			Backend{
				Addr:           ":443",
				ConnectTimeout: defaultConnectTimeout,
			},
		},
	}
	f2 := &Frontend{
		Name:      "test2.example.com", // from parent map key or filename
		BoundAddr: "127.0.0.1:55111",   // from parent
		Backends: []Backend{
			Backend{
				Addr:           ":80",
				ConnectTimeout: defaultConnectTimeout,
			},
		},
	}
	binding.Frontends = map[string]*Frontend{
		f1.Name: f1,
		f2.Name: f2,
	}

	expected := Configuration{
		binding.BindAddr: binding,
	}

	assert.EqualValues(t, expected, got)
}
