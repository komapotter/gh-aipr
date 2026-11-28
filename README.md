# gh-aipr

A GitHub CLI extension that automatically generates pull request titles and descriptions using AI.

## Overview

`gh-aipr` is a GitHub CLI extension that uses OpenAI to analyze your git diff and generate meaningful pull request titles and descriptions. This tool helps you create more descriptive and standardized PR content with minimal effort.

## Features

- Generate PR titles and descriptions based on your code changes
- OpenAI Chat Completions (default model: `gpt-6-luna`)
- Structured JSON output assembled into Markdown (`## Description` / `### Changes` / `### Testing`)
- Sanitizes placeholders and validates the result, retrying once on failure
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
gh aipr auth register              # paste the OpenAI API key (no echo)
gh aipr auth register -p openai    # provider flag is optional (openai only)
gh aipr auth status                # shows whether an OpenAI key is registered (never prints the key)
gh aipr auth remove
```

`login` / `logout` are not used; the commands are **register** / **remove** / **status**.

API keys are stored in the OS keyring, never in a config file.

### Credential resolution order

When generating a pull request title or description, credentials are resolved in this order:

1. Environment variables win if set (`OPENAI_API_KEY`)
2. Otherwise the OS keyring
3. Otherwise an error suggesting `gh aipr auth register`

Existing env-only workflows keep working without running `auth register`.

## Configuration

Configure the tool using environment variables. These remain optional when the matching key is registered in the OS keyring.

### OpenAI Configuration

- `OPENAI_API_KEY`: Your OpenAI API key (required unless registered via `gh aipr auth register`)
- `OPENAI_MODEL`: The OpenAI model to use (default: `gpt-6-luna`)
- `OPENAI_TEMPERATURE`: Temperature setting (default: `0.1`). Sent only for non-reasoning models, or when `OPENAI_REASONING_EFFORT` is `none`.
- `OPENAI_MAX_TOKENS`: Maximum completion tokens in the response, including reasoning tokens (default: `2000`). Sent as `max_completion_tokens` on Chat Completions.
- `OPENAI_REASONING_EFFORT`: Reasoning effort for reasoning models (`gpt-6*`, `gpt-5*`, o-series). Default: `low`. Allowed: `none`, `low`, `medium`, `high`, `xhigh`, `max`. Invalid values are rejected. Omitted for non-reasoning models such as `gpt-4o`.

## Output format

The model is asked for a JSON object (`title`, `description`, `changes`, `testing`) via OpenAI `response_format` `json_schema` (strict). If that request is rejected, the client falls back to `json_object`. Markdown is assembled locally with these headings:

```markdown
## Description

<summary>

### Changes

1. <item>

### Testing

<how to verify>
```

Headings stay in English even with `--japanise`; the title and section contents follow the requested language.

Before printing or `--create`, output is sanitized (code fences, leading preamble, title quotes/newlines, template remnants such as `---`, `* desc`, `1. change:`, `...`, empty sections) and validated (title empty/multi-line/too long; body missing headings or leftover remnants). Invalid output is retried once with a short fix instruction. There is no infinite retry loop.

`--title` and `--body` still print only the requested part; title and body are generated together in one structured call.

## License

[MIT License](LICENSE)
