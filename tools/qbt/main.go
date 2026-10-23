package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	qbt "github.com/autobrr/go-qbittorrent"
	"github.com/spf13/cobra"
)

// Global persistent flags
var (
	host     string
	username string
	password string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "qbt",
	Short: "qBittorrent CLI management and debugging tool",
	Long: `qbt is a CLI utility for interacting with qBittorrent in MediaHelper,
supporting torrent inspection, file exploration, category and tag management.`,
}

func init() {
	// Global persistent flags (inherited by all subcommands)
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "qBittorrent WebUI host URL (e.g. 127.0.0.1:8080)")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "qBittorrent username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "qBittorrent password")

	// Register subcommands
	rootCmd.AddCommand(categoriesCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(torrentCmd)
}

// newCtxClient initializes and authenticates the qBittorrent Client
func newCtxClient(ctx context.Context) (*qbt.Client, error) {
	targetHost := host
	if targetHost != "" && !strings.HasPrefix(targetHost, "http://") && !strings.HasPrefix(targetHost, "https://") {
		targetHost = "http://" + targetHost
	}

	cli := qbt.NewClient(qbt.Config{
		Host:     targetHost,
		Username: username,
		Password: password,
	})

	if err := cli.LoginCtx(ctx); err != nil {
		return nil, err
	}
	return cli, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func defaultStr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
