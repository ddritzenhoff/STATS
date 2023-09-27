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

// main is the entry point to the application binary.
func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	// Execute program.
	if err := m.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// clean up program
	m.Close()
}

// Main represents the program.
type Main struct {
	// Configuration path and parsed config data.
	Config     Config
	ConfigPath string

	// SQLite database used by SQLite service implementations.
	DB *sqlite.DB

	// HTTP server for handling HTTP communication.
	// SQLite services are attached to it before running.
	HTTPServer *http.Server
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Config:     DefaultConfig(),
		ConfigPath: DefaultConfigPath,

		DB: sqlite.NewDB(""),
	}
}

// Run initializes the member and Slack services and starts the HTTP server.
func (m *Main) Run(ctx context.Context) error {
	var configPath string
	flag.StringVar(&configPath, "config", "", "config file (extension: .json)")
	flag.Parse()

	cfg, err := ReadConfigFile(configPath)
	if err != nil {
		return err
	}

	DSN, err := expandDSN(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("Run expandDSN: %w", err)
	}

	m.DB = sqlite.NewDB(DSN)
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("db open: %w", err)
	}

	memberService := sqlite.NewMemberService(m.DB)
	leaderboardService := sqlite.NewLeaderboardService(m.DB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slackService, err := http.NewSlackService(logger, memberService, leaderboardService, cfg.Slack.SigningSecret, cfg.Slack.BotSigningKey, cfg.Slack.ChannelID)
	if err != nil {
		return fmt.Errorf("Run NewSlackService: %w", err)
	}

	m.HTTPServer = http.NewServer(logger, cfg.HTTP.Addr, slackService)
	if err := m.HTTPServer.Open(); err != nil {
		return fmt.Errorf("Run: %w", err)
	}

	return nil
}

func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
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

const (
	// DefaultConfigPath is the default path to the application configuration.
	DefaultConfigPath = "~/statsd.conf"

	// DefaultDSN is the default datasource name.
	DefaultDSN = "~/.statsd/db"
)

// Config represents the CLI configuration file.
type Config struct {
	DB struct {
		DSN string `json:"dsn"`
	} `json:"db"`

	HTTP struct {
		Addr string `json:"addr"`
	}

	Slack struct {
		SigningSecret string `json:"signing_secret"`
		BotSigningKey string `json:"bot_signing_key"`
		ChannelID     string `json:"channel_id"`
	} `json:"slack"`
}

// DefaultConfig returns a new instance of Config with defaults set.
func DefaultConfig() Config {
	var config Config
	config.DB.DSN = DefaultDSN
	return config
}

// ReadConfigFile decodes the config from the file path.
func ReadConfigFile(filepath string) (*Config, error) {
	configPath, err := expand(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not expand config path: %w", err)
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("could not decode JSON: %w", err)
	}

	return &config, nil
}
