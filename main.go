package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

type Config struct {
	// OpenAI configuration
	OpenAIKey         string  `envconfig:"OPENAI_API_KEY"`
	OpenAIModel       string  `envconfig:"OPENAI_MODEL" default:"gpt-4o"`
	OpenAITemperature float64 `envconfig:"OPENAI_TEMPERATURE" default:"0.1"`
	OpenAIMaxTokens   int     `envconfig:"OPENAI_MAX_TOKENS" default:"450"`
	
	// Anthropic configuration
	AnthropicKey         string  `envconfig:"ANTHROPIC_API_KEY"`
	AnthropicModel       string  `envconfig:"ANTHROPIC_MODEL" default:"claude-3-haiku-20240307"`
	AnthropicTemperature float64 `envconfig:"ANTHROPIC_TEMPERATURE" default:"0.1"`
	AnthropicMaxTokens   int     `envconfig:"ANTHROPIC_MAX_TOKENS" default:"450"`
	
	// API provider to use
	Provider string `envconfig:"AI_PROVIDER" default:"openai"`
}

var (
	verbose   bool // Global flag to control verbose output
	create    bool // Global flag to control pull request creation
	titleOnly bool // Global flag to control title-only output
	bodyOnly  bool // Global flag to control body-only output
	japanise  bool // Global flag to control Japanese output
	issueNo   int  // Global flag for the issue number
)

func printHelp() {
	helpMessage := `
This program generates a pull request title and description based on the git diff with the default branch.

USAGE
  gh aipr [flags]

FLAGS
  --help         Show help for command
  --version      Print version and exit
  --verbose      Enable verbose output
  --create       Create a pull request
  --title        Output only the title
  --body         Output only the body
  --japanise     Output in Japanese
  --issue-no     Associate an issue number with the pull request

EXAMPLES
  $ gh aipr --help
  $ gh aipr --version
  $ gh aipr --verbose

ENVIRONMENT VARIABLES
  # OpenAI configuration
  OPENAI_API_KEY         Your OpenAI API key (required when using OpenAI)
  OPENAI_MODEL           The OpenAI model to use (default: gpt-4o)
  OPENAI_TEMPERATURE     The temperature to use for the OpenAI model (default: 0.1)
  OPENAI_MAX_TOKENS      The maximum number of tokens to use for the OpenAI model (default: 450)
  
  # Anthropic configuration
  ANTHROPIC_API_KEY      Your Anthropic API key (required when using Anthropic)
  ANTHROPIC_MODEL        The Anthropic model to use (default: claude-3-haiku-20240307)
  ANTHROPIC_TEMPERATURE  The temperature to use for the Anthropic model (default: 0.1)
  ANTHROPIC_MAX_TOKENS   The maximum number of tokens to use for the Anthropic model (default: 450)
  
  # General configuration
  AI_PROVIDER            The AI provider to use (openai or anthropic, default: openai)
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

func getIssue(issueNo int) (string, error) {
	issueCmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", issueNo))
	var issueOut bytes.Buffer
	issueCmd.Stdout = &issueOut
	err := issueCmd.Run()
	if err != nil {
		return "", err
	}
	return issueOut.String(), nil
}

func confirm(prompt string) bool {
	var response string
	fmt.Print(prompt)
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false
	}
	return strings.ToLower(response) == "y"
}

func registerAppFlags(fs *flag.FlagSet, verbose, create, showHelp, titleOnly, bodyOnly, japanise, showVersion *bool, issueNo *int) {
	fs.BoolVar(verbose, "verbose", false, "Enable verbose output")
	fs.BoolVar(create, "create", false, "Create a pull request")
	fs.BoolVar(showHelp, "help", false, "Show help for command")
	fs.BoolVar(showVersion, "version", false, "Print version and exit")
	fs.BoolVar(titleOnly, "title", false, "Output only the title")
	fs.BoolVar(bodyOnly, "body", false, "Output only the body")
	fs.BoolVar(japanise, "japanise", false, "Output in Japanese")
	fs.IntVar(issueNo, "issue-no", 0, "Issue number to associate with the pull request")
}

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		fmt.Printf("Failed to process environment variables: %s\n", err)
		return
	}

	var showHelp, showVersion bool
	registerAppFlags(flag.CommandLine, &verbose, &create, &showHelp, &titleOnly, &bodyOnly, &japanise, &showVersion, &issueNo)
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}
	if showVersion {
		fmt.Println(versionString())
		return
	}

	if issueNo != 0 {
		issue, err := getIssue(issueNo)
		if err != nil {
			fmt.Println("Error getting issue:", err)
			return
		}
		fmt.Println(issue)
		if !confirm("Create pull request with this issue? (y/N): ") {
			return
		}
	}

	defaultBranch, err := getDefaultBranch()
	if err != nil {
		fmt.Println("Error getting default branch:", err)
		return
	}

	// Start the spinner for git diff
	diffSpinner := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Getting git diff")
	diffSpinner.start()

	diffOutput, err := getGitDiff()

	// Stop the spinner
	diffSpinner.stop()
	
	if err != nil {
		fmt.Println("Error getting git diff:", err)
		return
	}

	var title, body string

	// Create a spinner for the prompt creation step
	promptSpinner := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Creating prompts")
	promptSpinner.start()
	promptSpinner.stop()


	if titleOnly {
		titlePrompt := CreateOpenAIQuestion(PrTitle, diffOutput, japanise)
		title, err = AskAI(config, titlePrompt, verbose)
		if err != nil {
			fmt.Printf("Error asking AI for title: %s\n", err)
			return
		}
		fmt.Println("Generated Pull Request Title:")
		fmt.Println(title)
		return
	}

	if bodyOnly {
		bodyPrompt := CreateOpenAIQuestion(PrBody, diffOutput, japanise)
		body, err = AskAI(config, bodyPrompt, verbose)
		if err != nil {
			fmt.Printf("Error asking AI for body: %s\n", err)
			return
		}
		fmt.Println("Generated Pull Request Description:")
		fmt.Println(body)
		return
	}

	titlePrompt := CreateOpenAIQuestion(PrTitle, diffOutput, japanise)
	bodyPrompt := CreateOpenAIQuestion(PrBody, diffOutput, japanise)
	title, err = AskAI(config, titlePrompt, verbose)
	if err != nil {
		fmt.Printf("Error asking AI for title: %s\n", err)
		return
	}
	body, err = AskAI(config, bodyPrompt, verbose)
	if err != nil {
		fmt.Printf("Error asking AI for body: %s\n", err)
		return
	}

	if issueNo != 0 {
		body = fmt.Sprintf("#%d\n%s", issueNo, body)
	}

	if create {
		// Add spinner for PR creation
		prSpinner := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Creating pull request")
		prSpinner.start()

		prNumber, err := createPullRequest(title, body, defaultBranch)

		prSpinner.stop()
		
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
