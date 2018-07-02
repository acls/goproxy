package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Frontend_ParseYaml(t *testing.T) {
	// must use spaces for yaml indentation
	input := `
tls_key: /test1.key
tls_crt: /test1.crt
backends:
- addr: :80
- addr: :8080
`
	got := NewFrontend("127.0.0.1:55111", "test1.example.com", nil)
	err := got.ParseYaml([]byte(input))
	if err != nil {
		t.Errorf("Error parsing yaml config: %v", err)
		return
	}

	expected := &Frontend{
		BoundAddr: "127.0.0.1:55111",   // from parent
		Name:      "test1.example.com", // from parent map key or filename
		TLSCrt:    "/test1.crt",
		TLSKey:    "/test1.key",
		Backends: []Backend{
			Backend{
				Addr:           ":80",
				ConnectTimeout: defaultConnectTimeout,
			},
			Backend{
				Addr:           ":8080",
				ConnectTimeout: defaultConnectTimeout,
			},
		},
	}

	assert.EqualValues(t, expected, got)
}
