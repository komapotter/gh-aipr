package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/kelseyhightower/envconfig"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

type Config struct {
	OpenAIKey         string  `envconfig:"OPENAI_API_KEY" required:"true"`
	OpenAIModel       string  `envconfig:"OPENAI_MODEL" default:"gpt-4o"`
	OpenAITemperature float64 `envconfig:"OPENAI_TEMPERATURE" default:"0.1"`
	OpenAIMaxTokens   int     `envconfig:"OPENAI_MAX_TOKENS" default:"450"`
}

var verbose bool // Global flag to control verbose output

func getGitDiff() (string, error) {
	// Determine the default branch using go-gh
	client, err := api.DefaultRESTClient()
	if err != nil {
		return "", err
	}
	repo, err := repository.Current()
	if err != nil {
		return "", err
	}
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	err = client.Get(fmt.Sprintf("repos/%s/%s", repo.Owner, repo.Name), &repoInfo)
	if err != nil {
		return "", err
	}
	defaultBranch := repoInfo.DefaultBranch
	if err != nil {
		return "", err
	}

	// Get the git diff with the default branch
	diffCmd := exec.Command("git", "diff", "origin/"+defaultBranch)
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	err = diffCmd.Run()
	if err != nil {
		return "", err
	}
	return diffOut.String(), nil
}

func printHelp() {
	helpMessage := `
This program generates a pull request title and description based on the git diff with the default branch.

USAGE
  gh aipr [flags]

FLAGS
  --help     Show help for command
  --verbose  Enable verbose output

EXAMPLES
  $ gh aipr --help
  $ gh aipr --verbose

ENVIRONMENT VARIABLES
  OPENAI_API_KEY         Your OpenAI API key (required)
  OPENAI_MODEL           The OpenAI model to use (default: gpt-4o)
  OPENAI_TEMPERATURE     The temperature to use for the OpenAI model (default: 0.1)
  OPENAI_MAX_TOKENS      The maximum number of tokens to use for the OpenAI model (default: 450)
`
	fmt.Println(helpMessage)
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		fmt.Printf("Failed to process environment variables: %s\n", err)
		return
	}

	verbose = false // Default verbose to false
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			printHelp()
			return
		} else if os.Args[1] == "-v" || os.Args[1] == "--verbose" {
			verbose = true
		}
	}

	diffOutput, err := getGitDiff()
	if err != nil {
		fmt.Println("Error getting git diff:", err)
		return
	}

	question := CreateOpenAIQuestion(diffOutput)
	title, body, err := AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, question, verbose)
	if err != nil {
		fmt.Println("Error asking OpenAI:", err)
		return
	}

	fmt.Println("Generated Pull Request Title:")
	fmt.Println(title)
	fmt.Println("")
	fmt.Println("Generated Pull Request Description:")
	fmt.Println(body)
}
