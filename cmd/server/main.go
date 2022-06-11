package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/constants"
	"github.com/mikarios/imageresizer/internal/routes"
	"github.com/mikarios/imageresizer/internal/services/cdnservice"
	"github.com/mikarios/imageresizer/internal/services/config"
	"github.com/mikarios/imageresizer/internal/services/imageservice"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
	"github.com/mikarios/imageresizer/pkg/queueservice"
)

func main() {
	bgCTX := context.Background()
	cfg := config.Init("", constants.ServerTypes.ImageResizer)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	if err := setupLogger(cfg.LOG.Level, cfg.LOG.Format, cfg.LOG.Trace); err != nil {
		logger.Panic(bgCTX, err)
	}

	createServicesNeeded(cfg)

	httpServer := startHTTPServer(bgCTX)

	go startListenerForJobs()

	event := <-quit

	logger.Warning(bgCTX, fmt.Sprintf("RECEIVED SIGNAL: %v", event))
	gracefullyShutdown(httpServer)
	destroyServices()
}

func setupLogger(level, formatter string, trace bool) error {
	if err := logger.SetFormatter(formatter); err != nil {
		return err
	}

	logger.SetLogTrace(trace)

	return logger.SetLogLevel(level)
}

func startListenerForJobs() {
	q := queueservice.GetInstance()

	incoming, err := q.ImageConsume()
	if err != nil {
		panic(err)
	}

	for job := range incoming {
		imageJob := &imagedto.ImageProcessJob{
			QueueJob: job,
			Data:     &imagedto.ImageProcessJobData{},
		}
		if err = json.Unmarshal(job.Body, &imageJob.Data); err != nil {
			logger.Error(context.Background(), err, "could not unmarshal imageJob", job.Body)
			_ = job.Nack(false, false)

			continue
		}

		imageservice.AddImageJob(imageJob)
	}

	logger.Warning(context.Background(), "Closing listener for jobs")
}

func createServicesNeeded(cfg *config.Config) {
	cdnservice.Init(cfg.CDN.Bucket, cfg.CDN.Key, cfg.CDN.Secret, cfg.CDN.Endpoint, cfg.CDN.Region)
	imageservice.Init()
	queueservice.Init(true, true, false, false)
}

func destroyServices() {
	queueservice.Destroy()
	imageservice.Destroy()
}

func startHTTPServer(bgCTX context.Context) *http.Server {
	logger.Info(bgCTX, "starting http server")

	cfg := config.GetInstance()

	router := routes.SetupRoutes()
	listenAddress := cfg.HTTP.IP + ":" + cfg.HTTP.Port

	if cfg.HTTP.PortTLS != "" {
		listenAddress = cfg.HTTP.IP + ":" + cfg.HTTP.PortTLS
	}

	return createServer(bgCTX, router, listenAddress, cfg)
}

func createServer(ctx context.Context, handler http.Handler, listenAddress string, cfg *config.Config) *http.Server {
	server := &http.Server{
		Addr:    listenAddress,
		Handler: handler,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	if cfg.HTTP.PortTLS != "" {
		// Redirect http requests to https.
		go startRedirect(ctx, cfg)
		go startTLS(ctx, server, cfg)
		logger.Info(ctx, "server started at:", cfg.HTTP.IP+":"+cfg.HTTP.Port, server.Addr)

		return server
	}

	// Start http server if no secure is configured.
	go startInsecure(ctx, server, cfg)
	logger.Info(ctx, "server started at", server.Addr)

	return server
}

func startRedirect(ctx context.Context, cfg *config.Config) {
	if err := http.ListenAndServe(cfg.HTTP.IP+":"+cfg.HTTP.Port, redirectHTTP(cfg.HTTP.PortTLS)); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Panic(ctx, err, "could not initiate http server", cfg.HTTP.Port)
		}
	}
}

func redirectHTTP(portTLS string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + strings.Split(r.Host, ":")[0] + ":" + portTLS + r.URL.String()

		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}
}

func startTLS(ctx context.Context, server *http.Server, cfg *config.Config) {
	if err := server.ListenAndServeTLS(cfg.HTTP.CertFile, cfg.HTTP.KeyFile); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Panic(ctx, err, "could not initiate https server", cfg.HTTP.PortTLS)
		}
	}
}

func startInsecure(ctx context.Context, server *http.Server, cfg *config.Config) {
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Panic(ctx, err, "could not initiate https server", cfg.HTTP.Port)
		}
	}
}

// gracefullyShutdown handles the server shutdown process after a signal received.
func gracefullyShutdown(server *http.Server) {
	bgCTX := context.Background()

	logger.Info(bgCTX, "Termination signal received. Shutting down everything gracefully.")

	ctx, cancel := context.WithTimeout(bgCTX, 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)

	if err := server.Shutdown(ctx); err != nil {
		logger.Error(bgCTX, err, "could not gracefully shutdown the http server")
	} else {
		logger.Info(bgCTX, "http server stopped")
	}

	logger.Info(bgCTX, "Shutdown process completed.")
}
