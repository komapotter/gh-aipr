package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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

var (
	verbose bool // Global flag to control verbose output
	create  bool // Global flag to control pull request creation
)

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

	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&create, "create", false, "Create a pull request")
	flag.Parse()

	if flag.NArg() > 0 && (flag.Arg(0) == "-h" || flag.Arg(0) == "--help") {
		printHelp()
		return
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

	if create {
		err = createPullRequest(title, body)
		if err != nil {
			fmt.Println("Error creating pull request:", err)
		}
	} else {
		fmt.Println("Generated Pull Request Title:")
		fmt.Println(title)
		fmt.Println("")
		fmt.Println("Generated Pull Request Description:")
		fmt.Println(body)
	}
}
func getCurrentBranch() (string, error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var branchOut bytes.Buffer
	branchCmd.Stdout = &branchOut
	err := branchCmd.Run()
	if err != nil {
		return "", err
	}
	return branchOut.String(), nil
}

func createPullRequest(title, body string) error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}
	repo, err := repository.Current()
	if err != nil {
		return err
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	prData := map[string]interface{}{
		"title": title,
		"body":  body,
		"head":  currentBranch,
		"base":  "main", // Replace with the actual base branch name
	}

	payloadBytes, err := json.Marshal(prData)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(payloadBytes)

	return client.Post(fmt.Sprintf("repos/%s/%s/pulls", repo.Owner, repo.Name), bodyReader, nil)
}
