package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available mock media presets",
	Long:  `List all built-in mock test cases and presets categorized by movie, tv, anime, and chinese.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available Mock Media Presets:")
		fmt.Println("---------------------------------------------------------------------------------------------------------")
		fmt.Printf("%-10s %-30s %-32s %-8s\n", "CATEGORY", "PRESET ID", "NAME", "FILES")
		fmt.Println("---------------------------------------------------------------------------------------------------------")

		for _, p := range DefaultPresets {
			fileCount := fmt.Sprintf("%d files", len(p.Files))
			if !p.IsDir {
				fileCount = "single"
			}
			fmt.Printf("%-10s %-30s %-32s %-8s\n",
				strings.ToUpper(p.Category),
				p.ID,
				truncateString(p.Name, 30),
				fileCount,
			)
			fmt.Printf("           └─ %s\n", p.Description)
		}
		fmt.Println("---------------------------------------------------------------------------------------------------------")
		fmt.Println("Usage example:")
		fmt.Println("  Generate specific presets: mdgen generate --ids movie-inception-single,anime-frieren-batch")
		fmt.Println("  Generate by category:     mdgen generate -c anime -o ./downloads")
	},
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(listCmd)
}
