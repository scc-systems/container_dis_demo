package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scc-systems/wallsync/internal/admin"
	"github.com/scc-systems/wallsync/internal/config"
	"github.com/scc-systems/wallsync/internal/feeds"
	"github.com/scc-systems/wallsync/internal/hub"
	"github.com/scc-systems/wallsync/internal/media"
	"github.com/scc-systems/wallsync/internal/playback"
	"github.com/scc-systems/wallsync/internal/playlist"
	"github.com/scc-systems/wallsync/internal/protocol"
	"github.com/scc-systems/wallsync/internal/scene"
	"github.com/scc-systems/wallsync/internal/web"
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

	playlistStore, err := playlist.New(cfg.MediaDir)
	if err != nil {
		logger.Error("playlist store init failed", "err", err, "dir", cfg.MediaDir)
		os.Exit(1)
	}
	logger.Info("playlist store ready", "path", playlistStore.Path(), "count", len(playlistStore.List()))

	player := playback.New(director, mediaStore, cfg.BasePath, logger)

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
		Playlists:      playlistStore,
		Playback:       player,
		MaxUploadBytes: cfg.MaxUploadBytes,
		BasePath:       cfg.BasePath,
	}

	connHandler := &hub.ConnHandler{
		Hub:          h,
		Welcomes:     director,
		PingInterval: cfg.PingInterval,
		Logger:       logger,
		NodeCount:    cfg.NodeCount,
		OnClientMessage: func(nodeID int, env protocol.Envelope) {
			if env.Type != protocol.TypeVideoDuration {
				return
			}
			var p protocol.VideoDurationPayload
			if err := protocol.UnmarshalPayload(env, &p); err != nil {
				return
			}
			if p.VideoID == "" || p.DurationS <= 0 {
				return
			}
			ms := int64(p.DurationS * 1000)
			if err := mediaStore.SetDuration(p.VideoID, ms); err != nil {
				logger.Warn("SetDuration failed", "err", err, "video_id", p.VideoID)
			}
			player.ReportVideoDuration(p.VideoID, ms)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /wall", web.WallHandler(cfg.BasePath))
	mux.Handle("GET /", web.StaticHandler())
	mux.Handle("GET /ws", connHandler)
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(mediaStore.Root()))))
	mux.HandleFunc("GET /admin", adminSrv.ServeAdmin)
	mux.HandleFunc("GET /admin/status", adminSrv.ServeStatus)
	mux.HandleFunc("POST /admin/scene", adminSrv.ServeSceneSwitch)
	mux.HandleFunc("POST /admin/restart", adminSrv.ServeRestart)
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

	mux.HandleFunc("GET /admin/playlists", adminSrv.ServeListPlaylists)
	mux.HandleFunc("POST /admin/playlists", adminSrv.ServeCreatePlaylist)
	mux.HandleFunc("PUT /admin/playlists/{id}", adminSrv.ServeUpdatePlaylist)
	mux.HandleFunc("DELETE /admin/playlists/{id}", adminSrv.ServeDeletePlaylist)

	mux.HandleFunc("GET /admin/queue", adminSrv.ServeQueueSnapshot)
	mux.HandleFunc("POST /admin/queue/load", adminSrv.ServeQueueLoad)
	mux.HandleFunc("POST /admin/queue/items", adminSrv.ServeQueueAddItem)
	mux.HandleFunc("DELETE /admin/queue/items/{index}", adminSrv.ServeQueueRemoveItem)
	mux.HandleFunc("POST /admin/queue/reorder", adminSrv.ServeQueueReorder)
	mux.HandleFunc("POST /admin/queue/play", adminSrv.ServeQueuePlay)
	mux.HandleFunc("POST /admin/queue/pause", adminSrv.ServeQueuePause)
	mux.HandleFunc("POST /admin/queue/skip", adminSrv.ServeQueueSkip)
	mux.HandleFunc("POST /admin/queue/stop", adminSrv.ServeQueueStop)
	mux.HandleFunc("POST /admin/queue/save-as", adminSrv.ServeQueueSaveAs)

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
