goproxy is a simple reverse-proxy with SNI multiplexing (TLS virtual hosts).

That means you can send TLS/SSL connections for multiple different applications to the same port and forward
them all to the appropriate backend hosts depending on the intended destination.

# Features

### SNI Multiplexing
goproxy multiplexes connections to a single TLS port by inspecting the name in the SNI extension field of each connection.

### Simple YAML Configuration
You configure goproxy with a simple YAML configuration file:

```yaml
":443":
  secure: true
  frontends:
    v1.example.com:
      backends:
      - addr: :4443

    v2.example.com:
      backends:
      - addr: 192.168.0.2:443
      - addr: 192.168.0.1:443

":80":
  secure: false
  watch: true

":1234":
  secure: false
  frontends:
    test.example.com:1234:
      backends:
      - addr: 192.168.1.1:443
```

If `watch` is true a folder with the name is watched for frontend configs:

```yaml
# ./:80/test1.example.com.yml
backends:
- addr: 192.168.1.1:80
```
```yaml
# ./:80/test2.example.com.yml
backends:
- addr: 192.168.1.2:80
```

NOTE: When using non-standard ports the frontend domain needs to include the port. eg: test.example.com:1234


### Optional TLS Termination
Sometimes, you don't actually want to terminate the TLS traffic, you just want to forward it elsewhere. goproxy only
terminates the TLS traffic if you specify a private key and certificate file like so:

```yaml
":443":
  frontends:
    v1.example.com:
      tls_key: /path/to/v1.example.com.key
      tls_crt: /path/to/v1.example.com.crt
```


### Round robin load balancing among arbitrary backends
goproxy performs simple round-robin load balancing when more than one backend is available (other strategies will be available in the future):

```yaml
":443":
  frontends:
    v1.example.com:
      backends:
      - addr: :8080
      - addr: :8081
```


# Running it
Running goproxy is also simple. It takes a single argument, the path to the configuration file:

    ./goproxy /path/to/config.yml


# Building it
Just cd into the directory and "go build". It requires Go 1.1+.

# Testing it
Just cd into the directory and "go test".

# As a Systemd Service

## Copy service file
NOTE: change ExecStart paths to match your paths, since the paths must be absolute. My $GOPATH is my home directory.

```bash
cp $GOPATH/src/github.com/acls/goproxy/goproxy.sample.service /etc/systemd/system/goproxy.service
vim /etc/systemd/system/goproxy.service
```

## Start service
```bash
systemctl start goproxy.service
```

## View logs
```bash
journalctl -u goproxy.service     # all logs
journalctl -u goproxy.service -f  # follow logs
```

## Set to automatically run on boot
```bash
systemctl enable goproxy.service
```

## Reload service without restarting after making changes to config
```bash
systemctl reload goproxy.service
```

# License
Apache
