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
```

### Flags

- `--help`: Show help for command
- `--verbose`: Enable verbose output
- `--create`: Create a pull request
- `--title`: Output only the title
- `--body`: Output only the body
- `--japanise`: Output in Japanese

### Examples

```bash
# Generate both title and description
gh aipr

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

## Configuration

Configure the tool using environment variables:

### OpenAI Configuration

- `OPENAI_API_KEY`: Your OpenAI API key (required when using OpenAI)
- `OPENAI_MODEL`: The OpenAI model to use (default: `gpt-4o`)
- `OPENAI_TEMPERATURE`: Temperature setting (default: `0.1`)
- `OPENAI_MAX_TOKENS`: Maximum tokens in response (default: `450`)

### Anthropic Configuration

- `ANTHROPIC_API_KEY`: Your Anthropic API key (required when using Anthropic)
- `ANTHROPIC_MODEL`: The Anthropic model to use (default: `claude-3-haiku-20240307`)
- `ANTHROPIC_TEMPERATURE`: Temperature setting (default: `0.1`)
- `ANTHROPIC_MAX_TOKENS`: Maximum tokens in response (default: `450`)

### General Configuration

- `AI_PROVIDER`: The AI provider to use (`openai` or `anthropic`, default: `openai`)

## License

[MIT License](LICENSE)