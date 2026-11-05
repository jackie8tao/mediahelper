package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

var (
	parseInput string
	parsePath  string
)

var parseCmd = &cobra.Command{
	Use:   "parse [flags] [raw_title]",
	Short: "Parse a raw media filename, torrent title, or local directory into structured metadata",
	Example: `  mdhp parser parse -k sk-xxx "[SubsPlease] Sousou no Frieren - 01 (1080p).mkv"
  mdhp parser parse -k sk-xxx -i "Inception.2010.2160p.UHD.BluRay.x265-GROUP"
  mdhp parser parse -k sk-xxx -p ./tests/src/Frieren.S01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw := strings.TrimSpace(parseInput)
		targetPath := strings.TrimSpace(parsePath)
		if raw == "" && targetPath == "" && len(args) > 0 {
			raw = strings.TrimSpace(args[0])
		}

		var (
			relFiles  []string
			isDirPath bool
		)
		if targetPath != "" {
			if raw == "" {
				raw = filepath.Base(targetPath)
			}
			if fi, err := os.Stat(targetPath); err == nil && fi.IsDir() {
				isDirPath = true
				relFiles, _ = parser.ScanRelativeVideoFiles(targetPath)
			}
		}

		if raw == "" {
			return fmt.Errorf("raw input title or path is required (provide as argument, -i/--input, or -p/--path)")
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

		var info *parser.MediaInfo
		if len(relFiles) > 0 {
			info, err = p.ParseFiles(context.Background(), raw, relFiles)
			if err == nil && isDirPath {
				for i := range info.Files {
					if info.Files[i].FilePath != "" && !filepath.IsAbs(info.Files[i].FilePath) {
						info.Files[i].FilePath = filepath.Join(targetPath, filepath.FromSlash(info.Files[i].FilePath))
					}
				}
			}
		} else {
			info, err = p.Parse(context.Background(), raw)
			if err == nil && targetPath != "" && !isDirPath {
				info.Files = []parser.FileItem{
					{
						FilePath: targetPath,
						Season:   info.Season,
						Episode:  info.Episode,
					},
				}
			}
		}
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
	parseCmd.Flags().StringVarP(&parsePath, "path", "p", "", "Local directory or file path to scan and parse")

	parserCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(parserCmd)
}
