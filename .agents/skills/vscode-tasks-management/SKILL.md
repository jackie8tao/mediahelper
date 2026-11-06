---
name: vscode-tasks-management
description: >-
  Use this skill when configuring, optimizing, or updating VSCode tasks,
  interactive prompts/inputs, or managing .vscode configuration tracking.
---

# VSCode Tasks Management Skill

This skill outlines guidelines for managing `.vscode/tasks.json` in the `mediahelper` repository, ensuring smooth interactive local workflows while safeguarding secrets and team portability.

## Guidelines & Principles

1. **Version Control Inclusion**:
   - Project-level shared build, test, and tooling tasks in `.vscode/tasks.json` are tracked in Git.
   - User-specific private files (like `.vscode/settings.json` or `.vscode/*.local.json`) should be excluded in `.gitignore`.
2. **Interactive Inputs Over Hardcoded Secrets**:
   - Never commit sensitive tokens, API keys, or passwords directly into task command lines.
   - Use VSCode input variables (`inputs` array) with `promptString` or `pickString` to prompt the user dynamically at runtime.

## Input Configuration Pattern

Example of an interactive selector for API keys or options:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Test: Run DeepSeek Recognition",
      "type": "shell",
      "command": "go run ./cmd/mediahelper parse --recognizer deepseek --api-key ${input:deepseekApiKey}",
      "problemMatcher": []
    }
  ],
  "inputs": [
    {
      "id": "deepseekApiKey",
      "type": "pickString",
      "description": "Select or enter your DeepSeek API Key",
      "options": [
        "${env:DEEPSEEK_API_KEY}",
        "custom"
      ],
      "default": "${env:DEEPSEEK_API_KEY}"
    }
  ]
}
```

## Checklist for Modifying Tasks

- [ ] Task labels must be descriptive and grouped intuitively (`Build: ...`, `Test: ...`, `Tool: ...`).
- [ ] Working directory defaults correctly to `${workspaceFolder}`.
- [ ] No hardcoded personal absolute paths.
- [ ] Verification command executed and tested in terminal.

