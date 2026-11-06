---
name: media-mock-data-generator
description: >-
  Use this skill when generating mock media files, test directories, or test cases
  for media parsing, organizing, and pipeline workflows without consuming physical disk space.
---

# Media Mock Data Generator Skill

This skill defines standard procedures and best practices for creating lightweight test media files (sparse/zero-byte files) to validate media naming, directory structure organization, and end-to-end mediahelper workflows.

## Principles

1. **Zero Real Disk Consumption**: Never copy or allocate large multi-gigabyte media files for testing. Always use sparse files or zero-byte files that preserve correct file extensions (`.mkv`, `.mp4`, `.flac`, `.srt`, `.nfo`).
2. **Deterministic Layout**: Generate mock data in dedicated isolated locations (e.g., `testdata/mock_media/` or a temporary directory) to avoid polluting local media libraries.
3. **High Scenario Coverage**: Test data must encompass diverse real-world media naming variants, messy release group tags, multi-part episodes, audio specs, and resolution indicators.

## Tooling Usage (`tools/mdgen`)

Use the workspace CLI tool `mdgen` to automate test data generation:

```bash
# Build mdgen binary if needed
go build -o bin/mdgen ./tools/mdgen

# Generate test datasets with custom targets
./bin/mdgen generate --dir ./testdata/mock_media --preset all
```

## Supported Media Categories & Naming Patterns

### 1. Feature Films (Movies)
- Standard: `Inception (2010)/Inception.2010.1080p.BluRay.x264.DTS-HD.MA.5.1-GROUP.mkv`
- Multi-edition: `Blade Runner (1982)/Blade.Runner.1982.Final.Cut.2160p.UHD.HDR.TrueHD.7.1.Atmos-GROUP.mkv`
- Accompanying assets: `.nfo`, `.chs.srt`, `poster.jpg`

### 2. Episodic TV Series
- Multi-season:
  ```text
  Breaking Bad/
  ├── Season 01/
  │   ├── Breaking.Bad.S01E01.1080p.BluRay.x264-GROUP.mkv
  │   └── Breaking.Bad.S01E02.1080p.BluRay.x264-GROUP.mkv
  └── Season 02/
      └── Breaking.Bad.S02E01.1080p.BluRay.x264-GROUP.mkv
  ```
- Anime / Absolute numbering: `[ReleaseGroup] Frieren - Beyond Journey's End - 01 [1080p HEVC AAC].mkv`
- Multi-episode bundles: `Friends.S01E01-E02.1080p.mkv`
- Specials: `Breaking Bad/Season 00/Breaking.Bad.S00E01.mkv`

### 3. Audio & Music
- Lossless albums:
  ```text
  Artist Name - Album Title (2020)/
  ├── 01. Track One.flac
  ├── 02. Track Two.flac
  └── cover.jpg
  ```

## Verification Checklist

- [ ] Path names contain no illegal shell characters.
- [ ] Directory structure isolates movies, tv shows, and music into respective folders.
- [ ] Test runs can parse titles, release years, seasons, and episodes correctly against metadata matchers.
- [ ] Cleanup command or helper removes test data when no longer needed.

