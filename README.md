# gh-aipr

A GitHub CLI extension that automatically generates pull request titles and descriptions using AI.

## Overview

`gh-aipr` is a GitHub CLI extension that uses AI (OpenAI or Anthropic) to analyze your git diff and generate meaningful pull request titles and descriptions. This tool helps you create more descriptive and standardized PR content with minimal effort.

## Features

- Generate PR titles and descriptions based on your code changes
- Support for both OpenAI (default) and Anthropic APIs
- Option to create the PR directly 
- Japanese language support
- Configurable AI parameters via environment variables
- Optional OS keyring storage for API keys (`gh aipr auth register`)

## Installation

```bash
# Install the gh CLI if you haven't already
# https://cli.github.com/

# Install the extension
gh extension install komapotter/gh-aipr
```

## Usage

```bash
gh aipr [flags]
gh aipr auth <command>
```

### Flags

- `--help`: Show help for command
- `--version`: Print version and exit
- `--verbose`: Enable verbose output
- `--create`: Create a pull request
- `--title`: Output only the title
- `--body`: Output only the body
- `--japanise`: Output in Japanese

### Examples

```bash
# Generate both title and description
gh aipr

# Print the installed version
gh aipr --version

# Generate only the title
gh aipr --title

# Generate only the description
gh aipr --body

# Generate content in Japanese
gh aipr --japanise

# Generate and create the PR in one command
gh aipr --create

# Show detailed information
gh aipr --verbose
```

### Authentication

API keys can live in environment variables (CI and one-off overrides) or in the OS secure store:

- macOS: Keychain
- Linux: Secret Service (when available)
- Windows: Credential Manager (Wincred)

On macOS, the first Keychain access may show a permission dialog.

```bash
gh aipr auth register              # choose openai or anthropic, then paste the key (no echo)
gh aipr auth register -p openai    # or pass the provider as a flag / argument
gh aipr auth status                # shows registered providers and the active one (never prints the key)
gh aipr auth switch anthropic      # openai or anthropic only
gh aipr auth remove -p openai
```

`login` / `logout` are not used; the commands are **register** / **remove** / **status** / **switch**.

Non-secret settings (the active provider) are stored in `$XDG_CONFIG_HOME/gh-aipr/config.yml`, defaulting to `~/.config/gh-aipr/config.yml` (on Windows, `%AppData%\gh-aipr\config.yml`). API keys are never written to this file.

### Credential resolution order

When generating a pull request title or description, credentials are resolved in this order:

1. Environment variables win if set (`OPENAI_API_KEY` / `ANTHROPIC_API_KEY` / `AI_PROVIDER`)
2. Otherwise the OS keyring plus local config
3. Otherwise an error suggesting `gh aipr auth register`

Existing env-only workflows keep working without running `auth register`.

## Configuration

Configure the tool using environment variables. These remain optional when the matching key is registered in the OS keyring.

### OpenAI Configuration

- `OPENAI_API_KEY`: Your OpenAI API key (required when using OpenAI unless registered via `gh aipr auth register`)
- `OPENAI_MODEL`: The OpenAI model to use (default: `gpt-4o`)
- `OPENAI_TEMPERATURE`: Temperature setting (default: `0.1`)
- `OPENAI_MAX_TOKENS`: Maximum tokens in response (default: `450`)

### Anthropic Configuration

- `ANTHROPIC_API_KEY`: Your Anthropic API key (required when using Anthropic unless registered via `gh aipr auth register`)
- `ANTHROPIC_MODEL`: The Anthropic model to use (default: `claude-3-haiku-20240307`)
- `ANTHROPIC_TEMPERATURE`: Temperature setting (default: `0.1`)
- `ANTHROPIC_MAX_TOKENS`: Maximum tokens in response (default: `450`)

### General Configuration

- `AI_PROVIDER`: The AI provider to use (`openai` or `anthropic`, default: `openai`, or the value from local config)

## License

[MIT License](LICENSE)
