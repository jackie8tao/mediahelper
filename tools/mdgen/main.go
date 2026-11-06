package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mdgen",
	Short: "MediaHelper test data generator",
	Long: `mdgen is a CLI tool for automatically generating mock media files and directories.
It allows testing Parser, Organizer, and Scraper modules without consuming physical disk space.`,
}
