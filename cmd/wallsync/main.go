package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eleanor/wallsync/internal/admin"
	"github.com/eleanor/wallsync/internal/config"
	"github.com/eleanor/wallsync/internal/feeds"
	"github.com/eleanor/wallsync/internal/hub"
	"github.com/eleanor/wallsync/internal/media"
	"github.com/eleanor/wallsync/internal/scene"
	"github.com/eleanor/wallsync/internal/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	logger.Info("startup", "addr", cfg.Addr, "tick_hz", cfg.TickHz, "initial_scene", cfg.InitialScene,
		"wall_w", cfg.WallWidth, "wall_h", cfg.WallHeight, "per_node_w", cfg.PerNodeWidth, "per_node_h", cfg.PerNodeHeight)

	h := hub.New(logger, cfg.NodeCount)
	director := scene.NewDirector(cfg, h, logger)

	mediaStore, err := media.New(cfg.MediaDir)
	if err != nil {
		logger.Error("media store init failed", "err", err, "dir", cfg.MediaDir)
		os.Exit(1)
	}
	logger.Info("media store ready", "dir", mediaStore.Root(), "max_upload_bytes", cfg.MaxUploadBytes)

	ticker := &feeds.Ticker{
		Hub:      h,
		Director: director,
		Hz:       cfg.TickHz,
		Logger:   logger,
		FFTBins:  128,
	}
	adminSrv := &admin.Admin{
		Hub:            h,
		Director:       director,
		Logger:         logger,
		Media:          mediaStore,
		MaxUploadBytes: cfg.MaxUploadBytes,
		BasePath:       cfg.BasePath,
	}

	connHandler := &hub.ConnHandler{
		Hub:          h,
		Welcomes:     director,
		PingInterval: cfg.PingInterval,
		Logger:       logger,
		NodeCount:    cfg.NodeCount,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /wall", web.WallHandler(cfg.BasePath))
	mux.Handle("GET /", web.StaticHandler())
	mux.Handle("GET /ws", connHandler)
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(mediaStore.Root()))))
	mux.HandleFunc("GET /admin", adminSrv.ServeAdmin)
	mux.HandleFunc("GET /admin/status", adminSrv.ServeStatus)
	mux.HandleFunc("POST /admin/scene", adminSrv.ServeSceneSwitch)
	mux.HandleFunc("GET /admin/ws", adminSrv.ServeAdminWS)
	mux.HandleFunc("GET /admin/videos", adminSrv.ServeListVideos)
	mux.HandleFunc("POST /admin/videos", adminSrv.ServeUploadVideo)
	mux.HandleFunc("DELETE /admin/videos/{id}", adminSrv.ServeDeleteVideo)
	mux.HandleFunc("POST /admin/play", adminSrv.ServePlayVideo)
	mux.HandleFunc("POST /admin/stop", adminSrv.ServeStopVideo)
	mux.HandleFunc("GET /admin/images", adminSrv.ServeListImages)
	mux.HandleFunc("POST /admin/images", adminSrv.ServeUploadImage)
	mux.HandleFunc("DELETE /admin/images/{id}", adminSrv.ServeDeleteImage)
	mux.HandleFunc("POST /admin/play-image", adminSrv.ServePlayImage)
	mux.HandleFunc("POST /admin/stop-image", adminSrv.ServeStopImage)
	mux.HandleFunc("POST /admin/autorotate", adminSrv.ServeToggleAutorotate)

	var handler http.Handler = mux
	if cfg.BasePath != "" {
		handler = http.StripPrefix(cfg.BasePath, mux)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go ticker.Run(rootCtx)
	go scene.Autorotate(rootCtx, director, cfg.AutoRotate)

	go func() {
		logger.Info("http listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "err", err)
			cancel()
		}
	}()

	<-rootCtx.Done()
	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
