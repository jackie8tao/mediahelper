---
name: mediahelper-quality-and-commit
description: >-
  Use this skill when preparing to commit code changes, running full pre-commit verification,
  or generating conventional commit messages in this repository.
---

# Mediahelper Quality and Commit Skill

This skill enforces strict code quality checks, language conventions, and git commit guidelines for `mediahelper`.

## Pre-Commit Verification Triad

Before staging or committing any code changes, always run the complete verification suite from workspace root:

```bash
# 1. Run all unit tests
go test ./...

# 2. Run static analysis
go vet ./...

# 3. Ensure full repository compiles cleanly
go build ./...
```

If any step fails, diagnose and resolve the issue before proceeding to commit.

## Language and Formatting Conventions

- **User Communication**: Always communicate with the user in Chinese (unless explicitly requested otherwise).
- **Code Comments**: All source code comments, documentation blocks, and inline explanations MUST be in English.
- **Git Commit Messages**: Write commit messages in English using the Conventional Commits specification.

## Conventional Commit Format

Structure:
```text
<type>(<scope>): <concise description in lowercase imperative mood>
```

### Common Types:
- `feat`: A new feature or capability.
- `fix`: A bug fix.
- `test`: Adding missing tests or correcting existing tests.
- `refactor`: Code change that neither fixes a bug nor adds a feature.
- `chore`: Maintenance tasks, config tweaks, dependency updates.
- `docs`: Documentation updates.

### Common Scopes:
- `database`: GORM models, migrations, and SQLite configurations.
- `pipeline`: Engine, step execution, triggers, and state recovery.
- `parser`: Filename title extraction, matchers (TMDB), recognizers (DeepSeek).
- `scraper`: Metadata fetchers (TMDB, Douban).
- `organizer`: File movement, symlinks, directory structuring.
- `downloader`: QBittorrent client integration.
- `tools`: Utilities in `tools/` (`mdgen`, `mdhp`, `qbt`).
- `config`: Configuration loading and system settings.

## Example Workflow

```bash
# Stage specific changes
git add internal/database/

# Check status
git status

# Commit with standard message
git commit -m "feat(database): complete missing gorm tags and enable automatic timestamps"
```

