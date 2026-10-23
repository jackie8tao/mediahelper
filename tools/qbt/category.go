package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var categoriesJSON bool

func init() {
	categoriesCmd.Flags().BoolVar(&categoriesJSON, "json", false, "Output raw JSON response")
}

var categoriesCmd = &cobra.Command{
	Use:     "categories",
	Aliases: []string{"cats"},
	Short:   "List all categories and save paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		categories, err := cli.GetCategoriesCtx(ctx)
		if err != nil {
			return fmt.Errorf("failed to get categories: %w", err)
		}

		if categoriesJSON {
			data, err := json.MarshalIndent(categories, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Configured categories (%d):\n\n", len(categories))
		for name, cat := range categories {
			fmt.Printf("  • Category:  %s\n", name)
			fmt.Printf("    Save Path: %s\n", cat.SavePath)
		}
		return nil
	},
}
