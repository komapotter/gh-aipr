package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"github.com/cli/go-gh/v2/pkg/api"
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
	// Determine the default branch
	defaultBranchCmd := exec.Command("sh", "-c", "git remote show origin | awk '/HEAD branch/ {print $NF}'")
	var defaultBranchOut bytes.Buffer
	defaultBranchCmd.Stdout = &defaultBranchOut
	err := defaultBranchCmd.Run()
	if err != nil {
		return "", err
	}
	defaultBranch := strings.TrimSpace(defaultBranchOut.String())

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

func main() {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		fmt.Printf("Failed to process environment variables: %s\n", err)
		return
	}

	verbose = false // Default verbose to false
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		verbose = true
	}

	fmt.Println("hi world, this is the gh-aipr extension!")
	client, err := api.DefaultRESTClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	response := struct{ Login string }{}
	err = client.Get("user", &response)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("running as %s\n", response.Login)

	diffOutput, err := getGitDiff()
	if err != nil {
		fmt.Println("Error getting git diff:", err)
		return
	}

	question := CreateOpenAIQuestion(diffOutput)
	responseText, err := AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, question, verbose)
	if err != nil {
		fmt.Println("Error asking OpenAI:", err)
		return
	}

	fmt.Println("Generated Pull Request Title and Description:")
	fmt.Println(responseText)
}
