package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	dbmigrations "bank-darah-backend/database"
	"bank-darah-backend/internal/app"

	"github.com/jackc/pgx/v5/pgxpool"
)

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Hanya setenv jika belum ada di env var sistem/proses
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadEnv()
	reencryptQRTokens := flag.Bool("reencrypt-qr-tokens", false, "reencrypt legacy donor QR tokens and exit")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg := app.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := dbmigrations.RunMigrations(ctx, pool); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}

	qrCipher, err := app.NewQRCipher(cfg.QREncryptionKey)
	if err != nil {
		logger.Error("initialize qr cipher", "error", err)
		os.Exit(1)
	}

	store := app.NewStore(pool, cfg, qrCipher)
	if *reencryptQRTokens {
		reencryptCtx, reencryptCancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer reencryptCancel()

		updated, err := store.ReencryptQRTokens(reencryptCtx, logger)
		if err != nil {
			logger.Error("reencrypt qr tokens", "error", err)
			os.Exit(1)
		}
		logger.Info("qr token reencrypt complete", "updated", updated)
		return
	}

	fcmClient, err := loadFCMClient(cfg, logger)
	if err != nil {
		logger.Error("initialize fcm", "error", err)
		os.Exit(1)
	}

	scheduler := app.NewScheduler(store, logger)
	scheduler.Start()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.New(cfg, store, logger, fcmClient).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("bank darah api started", "addr", server.Addr)

	serverErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			scheduler.Stop()
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
		scheduler.Stop()
		return
	}

	scheduler.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown server", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("close server", "error", closeErr)
		}
	}

	if err := <-serverErr; err != nil {
		logger.Error("server stopped", "error", err)
	}
}

func loadFCMClient(cfg app.Config, logger *slog.Logger) (*app.FCMClient, error) {
	if cfg.FCMProjectID == "" && cfg.FCMServiceAccountPath == "" {
		logger.Info("fcm disabled", "reason", "FCM_PROJECT_ID and FCM_SERVICE_ACCOUNT_PATH are not set")
		return nil, nil
	}
	if cfg.FCMProjectID == "" || cfg.FCMServiceAccountPath == "" {
		return nil, fmt.Errorf("FCM_PROJECT_ID and FCM_SERVICE_ACCOUNT_PATH must both be set")
	}

	serviceAccountJSON, err := os.ReadFile(cfg.FCMServiceAccountPath)
	if err != nil {
		return nil, fmt.Errorf("read fcm service account file: %w", err)
	}

	client, err := app.NewFCMClient(serviceAccountJSON, cfg.FCMProjectID)
	if err != nil {
		return nil, err
	}
	logger.Info("fcm enabled", "project_id", cfg.FCMProjectID)
	return client, nil
}
