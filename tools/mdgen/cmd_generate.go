package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	outputDir      string
	categoryFilter string
	sparseMode     bool
	cleanBeforeGen bool
	selectedIDs    string
)

var genCmd = &cobra.Command{
	Use:     "generate [flags]",
	Aliases: []string{"gen"},
	Short:   "Generate mock media test files and directories",
	Long: `Generate realistic mock media files and directories covering movies, TV series,
anime, and Chinese dramas. By default, files are 0-byte or minimal dummy files that take no disk space.
With --sparse, it creates sparse files showing simulated multi-gigabyte sizes without consuming disk blocks.`,
	Example: `  mdgen generate -o ./test_downloads
  mdgen generate -o ./test_downloads -c anime
  mdgen generate -o ./test_downloads -s --clean
  mdgen generate --ids movie-inception-single,tv-breaking-bad-s01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := strings.TrimSpace(outputDir)
		if out == "" {
			out = "./test_downloads"
		}

		absOut, err := filepath.Abs(out)
		if err != nil {
			return fmt.Errorf("resolve output directory path error: %w", err)
		}

		if cleanBeforeGen {
			fmt.Printf("Cleaning output directory: %s\n", absOut)
			_ = os.RemoveAll(absOut)
		}

		if err := os.MkdirAll(absOut, 0755); err != nil {
			return fmt.Errorf("create output directory error: %w", err)
		}

		idFilterMap := make(map[string]bool)
		if selectedIDs != "" {
			for _, id := range strings.Split(selectedIDs, ",") {
				idFilterMap[strings.TrimSpace(strings.ToLower(id))] = true
			}
		}

		cat := strings.ToLower(strings.TrimSpace(categoryFilter))
		if cat == "" {
			cat = "all"
		}

		var (
			totalFiles int
			totalDirs  int
		)

		fmt.Printf("Generating mock media test data into: %s\n", absOut)
		fmt.Printf("Options: Category=%s, SparseMode=%v\n\n", cat, sparseMode)

		for _, preset := range DefaultPresets {
			if len(idFilterMap) > 0 {
				if !idFilterMap[strings.ToLower(preset.ID)] {
					continue
				}
			} else if cat != "all" && !strings.EqualFold(preset.Category, cat) {
				continue
			}

			fmt.Printf("▶ [%s] %s (%s)\n", strings.ToUpper(preset.Category), preset.Name, preset.Description)

			var baseDir string
			if preset.IsDir {
				baseDir = filepath.Join(absOut, preset.DirName)
				if err := os.MkdirAll(baseDir, 0755); err != nil {
					return fmt.Errorf("create directory %s error: %w", baseDir, err)
				}
				totalDirs++
				fmt.Printf("  📁 %s/\n", preset.DirName)
			} else {
				baseDir = absOut
			}

			for _, f := range preset.Files {
				filePath := filepath.Join(baseDir, f.RelPath)
				if dir := filepath.Dir(filePath); dir != baseDir {
					_ = os.MkdirAll(dir, 0755)
				}

				if err := createMockFile(filePath, f.SimulatedSize, sparseMode); err != nil {
					return fmt.Errorf("create mock file %s error: %w", filePath, err)
				}
				totalFiles++

				sizeInfo := "0 B"
				if sparseMode && f.SimulatedSize > 0 {
					sizeInfo = formatSize(f.SimulatedSize) + " (sparse)"
				}
				fmt.Printf("    📄 %s [%s]\n", f.RelPath, sizeInfo)
			}
			fmt.Println()
		}

		fmt.Println("-------------------------------------------------------------------")
		fmt.Printf("Generation completed! Total directories: %d, Total files: %d\n", totalDirs, totalFiles)
		fmt.Printf("Output directory: %s\n\n", absOut)
		fmt.Println("Next steps to verify with MediaHelper:")
		fmt.Printf("  1. Test parser on a file:\n")
		fmt.Printf("     go run ./tools/mdhp parser parse -i \"Inception.2010.2160p.UHD.BluRay.x265.TrueHD.7.1.Atmos-GROUP.mkv\"\n\n")
		fmt.Printf("  2. Test parser on a directory:\n")
		fmt.Printf("     go run ./tools/mdhp parser parse -p \"%s/Breaking.Bad.S01.1080p.BluRay.x265-FLUX\"\n\n", absOut)
		fmt.Printf("  3. Test organizer on a file or folder:\n")
		fmt.Printf("     go run ./tools/mdhp organizer organize -d \"/path/to/media/library\" ...\n")
		return nil
	},
}

func createMockFile(path string, simulatedSize int64, sparse bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if sparse && simulatedSize > 0 {
		return f.Truncate(simulatedSize)
	}

	_, err = f.WriteString("MediaHelper Mock Data File\n")
	return err
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	if bytes >= gb {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	}
	if bytes >= mb {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	}
	if bytes >= kb {
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	}
	return fmt.Sprintf("%d B", bytes)
}

func init() {
	genCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "./test_downloads", "Directory where mock test files will be generated")
	genCmd.Flags().StringVarP(&categoryFilter, "category", "c", "all", "Category filter (all, movie, tv, anime, chinese)")
	genCmd.Flags().BoolVarP(&sparseMode, "sparse", "s", false, "Create sparse files showing simulated gigabyte sizes without consuming disk blocks")
	genCmd.Flags().BoolVar(&cleanBeforeGen, "clean", false, "Clean output directory before generating")
	genCmd.Flags().StringVar(&selectedIDs, "ids", "", "Comma-separated preset IDs to generate specifically")

	rootCmd.AddCommand(genCmd)
}
