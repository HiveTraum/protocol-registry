package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
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
		log.Println("Shutting down...")
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

	log.Println("Migrations applied successfully")
	return nil
}
