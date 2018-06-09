package conf

import (
	"reflect"
	"testing"
)

func TestParseConfigYaml(t *testing.T) {
	// must use spaces for yaml indentation
	input := `bob:
  secure: true
  bind_addr: 127.0.0.1:55111
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
	got, err := ParseConfigYaml([]byte(input))
	if err != nil {
		t.Errorf("Error parsing yaml config: %v", err)
		return
	}

	expected := Configuration{
		"bob": &Binding{
			Secure:   true,
			BindAddr: "127.0.0.1:55111",
			Frontends: map[string]*Frontend{
				"test1.example.com": &Frontend{
					TLSCrt: "/test1.crt",
					TLSKey: "/test1.key",
					Backends: []Backend{
						Backend{
							Addr: ":443",
						},
					},
				},
				"test2.example.com": &Frontend{
					Backends: []Backend{
						Backend{
							Addr: ":80",
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Got %v, expected %v", got, expected)
	}
}
