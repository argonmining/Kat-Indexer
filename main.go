// //////////////////////////////
package main

import (
	"context"
	"fmt"
	"kasplex-executor/api"
	"kasplex-executor/config"
	"kasplex-executor/explorer"
	"kasplex-executor/storage"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// //////////////////////////////
func main() {
	fmt.Println("KASPlex Executor v" + config.Version)

	// Set the correct working directory.
	arg0 := os.Args[0]
	if !strings.Contains(arg0, "go-build") {
		dir, err := filepath.Abs(filepath.Dir(arg0))
		if err != nil {
			log.Fatalln("main fatal:", err.Error())
		}
		os.Chdir(dir)
	}

	// Use the file lock for startup.
	fLock := "./.lockExecutor"
	lock, err := os.Create(fLock)
	if err != nil {
		log.Fatalln("main fatal:", err.Error())
	}
	defer os.Remove(fLock)
	defer lock.Close()
	err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		log.Fatalln("main fatal:", err.Error())
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	// Load config.
	var cfg config.Config
	config.Load(&cfg)

	// Set the log level.
	logOpt := &slog.HandlerOptions{Level: slog.LevelError}
	if cfg.Debug == 3 {
		logOpt = &slog.HandlerOptions{Level: slog.LevelDebug}
	} else if cfg.Debug == 2 {
		logOpt = &slog.HandlerOptions{Level: slog.LevelInfo}
	} else if cfg.Debug == 1 {
		logOpt = &slog.HandlerOptions{Level: slog.LevelWarn}
	}
	logHandler := slog.NewTextHandler(os.Stdout, logOpt)
	slog.SetDefault(slog.New(logHandler))

	// Set exit signal.
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	down := false
	go func() {
		<-c
		slog.Info("main stopping ..")
		cancel()
		down = true
		wg.Done()
	}()

	// Init storage driver.
	storage.Init(cfg.Cassandra, cfg.Rocksdb)

	// Init explorer if api server up.
	if !down {
		explorer.Init(ctx, wg, cfg.Startup, cfg.Testnet)
		go explorer.Run()
	}

	// Create a channel for shutdown signals
	stop := make(chan struct{})
	serverError := make(chan error, 1)

	if cfg.Api.Enabled {
		slog.Info("Starting API server", "port", cfg.Api.Port)
		apiServer := api.NewServer(
			cfg.Api.Port,
			cfg.Api.AllowedOrigins,
		)

		// Start server in goroutine
		go func() {
			if err := apiServer.Start(stop); err != nil && err != http.ErrServerClosed {
				slog.Error("API server error", "error", err)
				serverError <- err
			}
		}()
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		slog.Info("Shutdown signal received")
	case err := <-serverError:
		slog.Error("Server error", "error", err)
	}

	// Initiate graceful shutdown
	close(stop)

	// Give processes a moment to cleanup
	time.Sleep(time.Second)

	slog.Info("Shutdown complete")
}
