package main

import (
	"os"

	_ "github.com/jackie8tao/mediahelper/internal/parser/matcher/tmdb"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mdhp",
	Short: "MediaHelper CLI management and debugging tool",
	Long:  `mdhp is a CLI utility for testing and debugging MediaHelper modules.`,
}
