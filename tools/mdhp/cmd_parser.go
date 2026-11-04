package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/parser/deepseek"
	"github.com/spf13/cobra"
)

var (
	parserType    string
	parserAPIKey  string
	parserBaseURL string
	parserModel   string
)

var parserCmd = &cobra.Command{
	Use:   "parser",
	Short: "Media filename and torrent title parser operations",
	Long:  `Test and debug the parser module with different providers and model configurations.`,
}

var parseInput string

var parseCmd = &cobra.Command{
	Use:   "parse [flags] [raw_title]",
	Short: "Parse a raw media filename or torrent title into structured metadata",
	Example: `  mdhp parser parse -k sk-xxx "[SubsPlease] Sousou no Frieren - 01 (1080p).mkv"
  mdhp parser parse -k sk-xxx -i "Inception.2010.2160p.UHD.BluRay.x265-GROUP"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw := strings.TrimSpace(parseInput)
		if raw == "" && len(args) > 0 {
			raw = strings.TrimSpace(args[0])
		}
		if raw == "" {
			return fmt.Errorf("raw input title is required (provide as argument or via -i/--input)")
		}

		detailBytes, err := json.Marshal(deepseek.Config{
			APIKey:  parserAPIKey,
			BaseURL: parserBaseURL,
			Model:   parserModel,
		})
		if err != nil {
			return fmt.Errorf("marshal parser config failed: %w", err)
		}

		p, err := parser.CreateParser(config.ParserComponentConfig{
			Type:   parserType,
			Detail: detailBytes,
		})
		if err != nil {
			return fmt.Errorf("create parser failed: %w", err)
		}

		info, err := p.Parse(context.Background(), raw)
		if err != nil {
			return fmt.Errorf("parse failed: %w", err)
		}

		out, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("format result failed: %w", err)
		}
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	parserCmd.PersistentFlags().StringVarP(&parserType, "type", "t", "deepseek", "Parser provider type (e.g. deepseek)")
	parserCmd.PersistentFlags().StringVarP(&parserAPIKey, "apikey", "k", "", "LLM API key")
	parserCmd.PersistentFlags().StringVarP(&parserBaseURL, "baseurl", "u", "", "LLM API base URL (optional)")
	parserCmd.PersistentFlags().StringVarP(&parserModel, "model", "m", "", "LLM model name (optional)")

	parseCmd.Flags().StringVarP(&parseInput, "input", "i", "", "Raw filename or torrent title to parse")

	parserCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(parserCmd)
}
