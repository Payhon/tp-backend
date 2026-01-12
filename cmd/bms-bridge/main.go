package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"project/internal/app"
	"project/internal/bmsbridge"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径 (YAML)")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "missing -config")
		os.Exit(2)
	}

	application, err := app.NewApplication(
		app.WithConfigFile(*configPath),
		app.WithLogger(),
		app.WithDatabase(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init failed: %v\n", err)
		os.Exit(1)
	}
	log := logrus.StandardLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := bmsbridge.LoadConfigFromViper(application.Config)
	if err != nil {
		log.WithError(err).Error("load bms bridge config failed")
		os.Exit(1)
	}
	if !cfg.Enabled {
		log.Info("bms bridge disabled, exiting")
		return
	}

	br := bmsbridge.New(cfg, application.DB, log)
	if err := br.Start(ctx); err != nil {
		log.WithError(err).Error("bms bridge start failed")
		os.Exit(1)
	}

	// wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
	br.Stop()
	br.Wait()
}
