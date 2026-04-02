package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v2"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/user/protocol_registry/internal/app"
	"github.com/user/protocol_registry/internal/config"
	"github.com/user/protocol_registry/internal/migrations"
)

func main() {
	cliApp := &cli.App{
		Name:  "protocol-registry",
		Usage: "Protocol Registry server",
		Action: func(c *cli.Context) error {
			return runServe()
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the gRPC server",
				Action: func(c *cli.Context) error {
					return runServe()
				},
			},
			{
				Name:  "migrate",
				Usage: "Database migration commands",
				Subcommands: []*cli.Command{
					{
						Name:  "up",
						Usage: "Apply all pending migrations",
						Action: func(c *cli.Context) error {
							return runMigrateUp()
						},
					},
				},
			},
			{
				Name:  "auth",
				Usage: "Authentication management",
				Subcommands: []*cli.Command{
					{
						Name:  "create-key",
						Usage: "Generate a new API key and store its hash in the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "namespace",
								Value: "default",
								Usage: "Namespace to scope the key to",
							},
							&cli.StringFlag{
								Name:  "description",
								Value: "",
								Usage: "Human-readable description for the key",
							},
						},
						Action: func(c *cli.Context) error {
							return runCreateAPIKey(c.String("namespace"), c.String("description"))
						},
					},
				},
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	application := app.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		application.Shutdown()
		cancel()
	}()

	return application.Run(ctx)
}

func runMigrateUp() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)

	if err := goose.Up(db, "sql"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("migrations applied successfully")
	return nil
}

func runCreateAPIKey(namespace, description string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("failed to generate random key: %w", err)
	}
	rawKey := hex.EncodeToString(raw)

	sum := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(sum[:])

	var id string
	err = db.QueryRowContext(context.Background(),
		`INSERT INTO api_keys (namespace, description, key_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		namespace, description, keyHash,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to store api key: %w", err)
	}

	fmt.Printf("id:        %s\n", id)
	fmt.Printf("namespace: %s\n", namespace)
	fmt.Printf("key:       %s\n", rawKey)
	fmt.Println()
	fmt.Println("Store the key securely — it will not be shown again.")
	return nil
}
