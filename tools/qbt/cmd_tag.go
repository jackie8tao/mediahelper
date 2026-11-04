package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	tagAdd    bool
	tagRemove bool
	tagHash   string
	tagName   string
)

func init() {
	tagsCmd.Flags().BoolVar(&tagAdd, "add", false, "Add tag(s) to torrent")
	tagsCmd.Flags().BoolVar(&tagRemove, "remove", false, "Remove tag(s) from torrent")
	tagsCmd.Flags().StringVar(&tagHash, "hash", "", "Torrent hash (required when adding or removing tags)")
	tagsCmd.Flags().StringVarP(&tagName, "tag", "t", "", "Tag name(s) (comma-separated)")
}

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags, or add/remove tags for a torrent",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		// 1. Add tags
		if tagAdd {
			if tagHash == "" || tagName == "" {
				return fmt.Errorf("--hash and --tag flags are both required when adding tags")
			}
			if err := cli.AddTagsCtx(ctx, []string{tagHash}, tagName); err != nil {
				return fmt.Errorf("failed to add tags: %w", err)
			}
			fmt.Printf("Successfully added tag(s) [%s] to torrent [%s]\n", tagName, tagHash)
			return nil
		}

		// 2. Remove tags
		if tagRemove {
			if tagHash == "" || tagName == "" {
				return fmt.Errorf("--hash and --tag flags are both required when removing tags")
			}
			if err := cli.RemoveTagsCtx(ctx, []string{tagHash}, tagName); err != nil {
				return fmt.Errorf("failed to remove tags: %w", err)
			}
			fmt.Printf("Successfully removed tag(s) [%s] from torrent [%s]\n", tagName, tagHash)
			return nil
		}

		// 3. Default: list all tags
		tags, err := cli.GetTagsCtx(ctx)
		if err != nil {
			return fmt.Errorf("failed to get tags: %w", err)
		}

		fmt.Printf("Tags (%d):\n", len(tags))
		for _, t := range tags {
			fmt.Printf("  • %s\n", t)
		}
		return nil
	},
}
