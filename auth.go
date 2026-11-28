package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/komapotter/gh-aipr/internal/auth"
	"golang.org/x/term"
)

var errAuthUsage = errors.New("auth usage")

type authCLI struct {
	store      auth.Store
	getenv     auth.EnvLookup
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	readSecret func(provider string) (string, error)
}

func newAuthCLI() (*authCLI, error) {
	return &authCLI{
		store:  auth.KeyringStore{},
		getenv: os.LookupEnv,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}, nil
}

func runAuth(args []string) error {
	cli, err := newAuthCLI()
	if err != nil {
		return err
	}
	return cli.run(args)
}

func (c *authCLI) run(args []string) error {
	if len(args) == 0 {
		printAuthHelp(c.stdout)
		return errAuthUsage
	}
	switch args[0] {
	case "register":
		return c.register(args[1:])
	case "remove":
		return c.remove(args[1:])
	case "status":
		return c.status(args[1:])
	case "-h", "-help", "--help":
		printAuthHelp(c.stdout)
		return nil
	default:
		fmt.Fprintf(c.stderr, "unknown auth command %q\n\n", args[0])
		printAuthHelp(c.stderr)
		return errAuthUsage
	}
}

func printAuthHelp(w io.Writer) {
	fmt.Fprint(w, `
Usage: gh aipr auth <command>

Store an OpenAI API key in the OS keyring (macOS Keychain,
Linux Secret Service, or Windows Credential Manager) instead of a
plaintext env file.

Commands:
  register   Store an OpenAI API key in the OS keyring
  remove     Remove the stored OpenAI API key from the OS keyring
  status     Show whether an OpenAI API key is registered

Run "gh aipr auth <command> -h" for command help.

API keys are never printed. On macOS, the first Keychain access may
show a permission dialog.
`)
}

func (c *authCLI) newFlagSet(name string, usage func(io.Writer)) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	fs.Usage = func() { usage(fs.Output()) }
	return fs
}

func parseProviderFlags(fs *flag.FlagSet, args []string) (flagProvider string, positionals []string, err error) {
	var provider string
	fs.StringVar(&provider, "p", "", "Provider: openai")
	fs.StringVar(&provider, "provider", "", "Provider: openai")
	if err := fs.Parse(args); err != nil {
		return "", nil, err
	}
	return provider, fs.Args(), nil
}

func (c *authCLI) register(args []string) error {
	fs := c.newFlagSet("gh aipr auth register", func(w io.Writer) {
		fmt.Fprint(w, `Usage: gh aipr auth register [options] [provider]

Store an OpenAI API key in the OS keyring. The key is prompted with echoing
disabled and is never printed.

Options:
  -p, -provider string   Provider to register (openai only; default: openai)
`)
	})
	flagProvider, rest, err := parseProviderFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	provider, err := resolveProviderInput(flagProvider, rest)
	if err != nil {
		return err
	}

	secret, err := c.promptSecret(provider)
	if err != nil {
		return err
	}
	if err := c.store.Set(provider, secret); err != nil {
		return err
	}

	fmt.Fprintf(c.stdout, "Registered %s API key.\n", provider)
	return nil
}

func (c *authCLI) remove(args []string) error {
	fs := c.newFlagSet("gh aipr auth remove", func(w io.Writer) {
		fmt.Fprint(w, `Usage: gh aipr auth remove [options] [provider]

Remove the stored OpenAI API key from the OS keyring.

Options:
  -p, -provider string   Provider to remove (openai only; default: openai)
`)
	})
	flagProvider, rest, err := parseProviderFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	provider, err := resolveProviderInput(flagProvider, rest)
	if err != nil {
		return err
	}
	if err := c.store.Delete(provider); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return fmt.Errorf("no API key registered for %s", provider)
		}
		return err
	}
	fmt.Fprintf(c.stdout, "Removed %s API key from the keyring.\n", provider)
	return nil
}

func (c *authCLI) status(args []string) error {
	fs := c.newFlagSet("gh aipr auth status", func(w io.Writer) {
		fmt.Fprint(w, `Usage: gh aipr auth status

Show whether an OpenAI API key is registered. The raw key is never
printed.

Environment variables override the keyring when set.
`)
	})
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if rest := fs.Args(); len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}

	st := auth.LookupStatus(c.getenv, c.store)
	for _, p := range auth.Providers {
		fmt.Fprintf(c.stdout, "  %-10s %s\n", p+":", formatKeyStatus(st.Keys[p]))
	}
	return nil
}

func formatKeyStatus(ks auth.KeyStatus) string {
	if ks.Err != nil {
		return "not registered (keyring unavailable)"
	}
	if !ks.Registered {
		return "not registered"
	}
	switch ks.Source {
	case auth.SourceEnvironment:
		return "available (environment)"
	case auth.SourceKeyring:
		return "registered (keyring)"
	default:
		return "registered (" + ks.Source + ")"
	}
}

func resolveProviderInput(flagProvider string, rest []string) (string, error) {
	if flagProvider != "" && len(rest) > 0 {
		return "", fmt.Errorf("provide a provider via -provider or as an argument, not both")
	}
	raw := flagProvider
	if raw == "" && len(rest) > 0 {
		raw = rest[0]
		rest = rest[1:]
	}
	if len(rest) > 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}
	if raw == "" {
		return auth.ProviderOpenAI, nil
	}
	return auth.NormalizeProvider(raw)
}

func (c *authCLI) promptSecret(provider string) (string, error) {
	var secret string
	if c.readSecret != nil {
		s, err := c.readSecret(provider)
		if err != nil {
			return "", err
		}
		secret = s
	} else {
		prompt := fmt.Sprintf("Enter API key for %s (input hidden): ", provider)
		fmt.Fprint(c.stderr, prompt)
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			b, err := term.ReadPassword(fd)
			fmt.Fprintln(c.stderr)
			if err != nil {
				return "", err
			}
			secret = string(b)
		} else {
			line, err := bufio.NewReader(c.stdin).ReadString('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				return "", err
			}
			secret = line
		}
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}
	return secret, nil
}
