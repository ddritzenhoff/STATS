package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ddritzenhoff/stats/http"
	"github.com/ddritzenhoff/stats/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Execute program.
	if err := Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}

// Run initializes the member and Slack services and starts the HTTP server.
func Run(ctx context.Context) error {
	var configPath string
	flag.StringVar(&configPath, "config", "", "config file (extension: .json)")
	flag.Parse()

	configPath, err := expand(configPath)
	if err != nil {
		return fmt.Errorf("could not expand config path: %w", err)
	}

	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("Error decoding JSON: %w", err)
	}

	DSN, err := expandDSN(config.DSN)
	if err != nil {
		return fmt.Errorf("Run expandDSN: %w", err)
	}

	db := sqlite.NewDB(DSN)
	err = db.Open()
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}

	memberService := sqlite.NewMemberService(db)
	leaderboardService := sqlite.NewLeaderboardService(db)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slackService, err := http.NewSlackService(logger, memberService, leaderboardService, config.Slack.SigningSecret, config.Slack.BotSigningKey, config.Slack.ChannelID)
	if err != nil {
		return fmt.Errorf("Run NewSlackService: %w", err)
	}
	httpServer := http.NewServer(logger, config.ListenAddress, slackService)
	if err := httpServer.Open(); err != nil {
		return fmt.Errorf("Run: %w", err)
	}
	return nil
}

// expand returns path using tilde expansion. This means that a file path that
// begins with the "~" will be expanded to prefix the user's home directory.
func expand(path string) (string, error) {
	// Ignore if path has no leading tilde.
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	// Fetch the current user to determine the home path.
	u, err := user.Current()
	if err != nil {
		return path, fmt.Errorf("expand user.Current: %w", err)
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("expand u.HomeDir: home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}
	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil
}

// expandDSN expands a datasource name. Ignores in-memory databases.
func expandDSN(dsn string) (string, error) {
	if dsn == ":memory:" {
		return dsn, nil
	}
	return expand(dsn)
}

type Config struct {
	ListenAddress string `json:"listen_address"`
	DSN           string `json:"dsn"`
	Slack         struct {
		SigningSecret string `json:"signing_secret"`
		BotSigningKey string `json:"bot_signing_key"`
		ChannelID     string `json:"channel_id"`
	} `json:"slack"`
}
