package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strings"

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
	verbose   bool // Global flag to control verbose output
	create    bool // Global flag to control pull request creation
	titleOnly bool // Global flag to control title-only output
	bodyOnly  bool // Global flag to control body-only output
	japanise  bool // Global flag to control Japanese output
)

func printHelp() {
	helpMessage := `
This program generates a pull request title and description based on the git diff with the default branch.

USAGE
  gh aipr [flags]

FLAGS
  --help       Show help for command
  --verbose    Enable verbose output
  --create     Create a pull request
  --title      Output only the title
  --body       Output only the body
  --japanise   Output in Japanese

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

func getDefaultBranch() (string, error) {
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
	return repoInfo.DefaultBranch, nil
}

func getGitDiff() (string, error) {
	defaultBranch, err := getDefaultBranch()
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

func getCurrentBranch() (string, error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var branchOut bytes.Buffer
	branchCmd.Stdout = &branchOut
	err := branchCmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branchOut.String()), nil
}

func createPullRequest(title, body string, defaultBranch string) (int, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return 0, err
	}
	repo, err := repository.Current()
	if err != nil {
		return 0, err
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return 0, err
	}

	prData := map[string]interface{}{
		"title": title,
		"body":  body,
		"head":  currentBranch,
		"base":  defaultBranch, // Use the default branch
	}
	//fmt.Printf("prData: %+v", prData)

	payloadBytes, err := json.Marshal(prData)
	if err != nil {
		return 0, err
	}
	bodyReader := bytes.NewReader(payloadBytes)

	var prResponse struct {
		Number int `json:"number"`
	}
	err = client.Post(fmt.Sprintf("repos/%s/%s/pulls", repo.Owner, repo.Name), bodyReader, &prResponse)
	if err != nil {
		return 0, err
	}
	return prResponse.Number, nil
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		fmt.Printf("Failed to process environment variables: %s\n", err)
		return
	}

	var showHelp bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&create, "create", false, "Create a pull request")
	flag.BoolVar(&showHelp, "help", false, "Show help for command")
	flag.BoolVar(&titleOnly, "title", false, "Output only the title")
	flag.BoolVar(&bodyOnly, "body", false, "Output only the body")
	flag.BoolVar(&japanise, "japanise", false, "Output in Japanese")
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	defaultBranch, err := getDefaultBranch()
	if err != nil {
		fmt.Println("Error getting default branch:", err)
		return
	}

	// Start the spinner for git diff
	diffSpinner := NewSpinner("Getting git diff")
	diffSpinner.Start()
	
	diffOutput, err := getGitDiff()
	
	// Stop the spinner
	diffSpinner.Stop()
	
	if err != nil {
		fmt.Println("Error getting git diff:", err)
		return
	}

	var title, body string

	// Create a spinner for the prompt creation step
	promptSpinner := NewSpinner("Creating prompts")
	promptSpinner.Start()
	promptSpinner.Stop()

	if titleOnly {
		titlePrompt := CreateOpenAIQuestion(PrTitle, diffOutput, japanise)
		title, err = AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, titlePrompt, verbose)
		if err != nil {
			fmt.Println("Error asking OpenAI for title:", err)
			return
		}
		fmt.Println("Generated Pull Request Title:")
		fmt.Println(title)
		return
	}

	if bodyOnly {
		bodyPrompt := CreateOpenAIQuestion(PrBody, diffOutput, japanise)
		body, err = AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, bodyPrompt, verbose)
		if err != nil {
			fmt.Println("Error asking OpenAI for body:", err)
			return
		}
		fmt.Println("Generated Pull Request Description:")
		fmt.Println(body)
		return
	}

	titlePrompt := CreateOpenAIQuestion(PrTitle, diffOutput, japanise)
	bodyPrompt := CreateOpenAIQuestion(PrBody, diffOutput, japanise)
	title, err = AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, titlePrompt, verbose)
	if err != nil {
		fmt.Println("Error asking OpenAI for title:", err)
		return
	}
	body, err = AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, bodyPrompt, verbose)
	if err != nil {
		fmt.Println("Error asking OpenAI for body:", err)
		return
	}

	if create {
		// Add spinner for PR creation
		prSpinner := NewSpinner("Creating pull request")
		prSpinner.Start()
		
		prNumber, err := createPullRequest(title, body, defaultBranch)
		
		prSpinner.Stop()
		
		if err != nil {
			fmt.Println("Error creating pull request:", err)
		} else {
			fmt.Println(prNumber)
		}
	} else {
		fmt.Println("Generated Pull Request Title:")
		fmt.Println(title)
		fmt.Println("")
		fmt.Println("Generated Pull Request Description:")
		fmt.Println(body)
	}
}
