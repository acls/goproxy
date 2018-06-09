package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"slt/conf"
	"slt/proxy"

	"go.uber.org/zap"
)

type Options struct {
	configPath string
}

func parseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <config file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s is a simple TLS reverse proxy that can multiplex TLS connections\n"+
			"by inspecting the SNI extension on each incoming connection. This\n"+
			"allows you to accept connections to many different backend TLS\n"+
			"applications on a single port.\n\n"+
			"%s takes a single argument: the path to a YAML configuration file.\n\n", os.Args[0], os.Args[0])
	}
	flag.Parse()

	if len(flag.Args()) != 1 {
		return nil, fmt.Errorf("You must specify a single argument, the path to the configuration file.")
	}

	return &Options{
		configPath: flag.Arg(0),
	}, nil

}

// func loadTLSConfig(crtPath, keyPath string) (*tls.Config, error) {
// 	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &tls.Config{
// 		Certificates: []tls.Certificate{cert},
// 	}, nil
// }

func main() {
	// log := zap.L()
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	zap.ReplaceGlobals(log)

	// parse command line options
	opts, err := parseArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// read configuration file
	configBuf, err := ioutil.ReadFile(opts.configPath)
	if err != nil {
		log.Fatal("Failed to read configuration file",
			zap.String("path", opts.configPath),
			zap.Error(err),
		)
		os.Exit(1)
	}

	// parse configuration file
	config, err := conf.ParseConfigYaml(configBuf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for key, binding := range config {
		// run server
		s := &proxy.Server{
			Name:    key,
			Binding: binding,
			Logger:  log,
		}

		go func(s *proxy.Server) {
			if err := s.Run(); err != nil {
				log.Fatal("Failed to start slt",
					zap.Error(err),
				)
				os.Exit(1)
			}
		}(s)
	}

	// block forever
	<-make(chan bool)
}
