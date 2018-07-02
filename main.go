package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"slt/conf"
	"slt/proxy"
	"sync"

	"go.uber.org/zap"
)

func main() {
	{
		log, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		defer log.Sync()
		zap.ReplaceGlobals(log)
	}

	// parse command line options
	opts, err := parseArgs()
	if err != nil {
		zap.L().Fatal("Parse args", zap.Error(err))
		os.Exit(1)
	}

	config := conf.NewConfiguration()
	// parse configuration file
	if err := config.ParseFile(opts.ConfigPath); err != nil {
		zap.L().Fatal("Parse configuration", zap.Error(err))
		os.Exit(1)
	}

	baseDir := path.Dir(opts.ConfigPath)
	var cw *conf.ConfigWatcher

	var wg sync.WaitGroup
	for key, binding := range config {
		wg.Add(1)
		// run server
		s := &proxy.Server{
			Name:    key,
			Binding: binding,
			Logger:  zap.L(),
		}
		s.Init()
		if binding.Watch {
			// lazy init config watcher
			if cw == nil {
				cw, err = conf.NewConfigWatcher()
				if err != nil {
					zap.L().Fatal("New configuration watcher", zap.Error(err))
					os.Exit(1)
				}
			}

			dir := path.Join(baseDir, s.Name)
			_ = os.MkdirAll(dir, os.ModeDir|os.ModePerm)

			if err := cw.Add(dir, binding.BindAddr, s); err != nil {
				zap.L().Fatal("Failed to add watch directory",
					zap.Error(err),
					zap.String("dir", dir),
				)
				os.Exit(1)
			}
		}

		go func(s *proxy.Server) {
			go func() {
				<-s.Ready()
				wg.Done()
			}()
			if err := s.Run(); err != nil {
				zap.L().Fatal("Failed to start slt", zap.Error(err))
				os.Exit(1)
			}
		}(s)
	}

	wg.Wait()
	zap.L().Info("Done waiting")
	if cw != nil {
		zap.L().Info("Start watching")
		cw.Start()
	}

	// block forever
	<-make(chan bool)
}

type Options struct {
	ConfigPath string
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
		return nil, errors.New("You must specify a single argument, the path to the configuration file")
	}

	return &Options{
		ConfigPath: flag.Arg(0),
	}, nil
}
