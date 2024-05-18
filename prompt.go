package main

import "fmt"

// CreateOpenAIQuestion formats a question for the OpenAI API based on the git diff output.
func CreateOpenAIQuestion(diffOutput string) string {
	prompt := `
Please generate appropriate pull request message based on the context.

Here is a sample of pull-request format.

---
## Pull Request Title
Add environment variables section to README.md

## Description

* This pull request adds a section to the README.md file regarding the setting of environment variables. 
* This change ensures that users can correctly configure the necessary environment variables to use the tool effectively.

### Changes
1. **Addition of environment variables explanation**:
   - OPENAI_API_KEY: Your OpenAI API key
   - NUM_CANDIDATES: The number of commit message candidates to generate (default: 3)
   - OPENAI_MODEL: The OpenAI model to use (default: gpt-4o)
   - OPENAI_TEMPERATURE: The OpenAI temperature parameter (default: 0.1)
   - OPENAI_MAX_TOKENS: The maximum number of tokens for OpenAI (default: 450)

2. **Example of setting environment variables**:
   Added a shell script example to demonstrate how to set the environment variables.

   export OPENAI_API_KEY="your_openai_api_key"
   export NUM_CANDIDATES=3
   export OPENAI_MODEL="gpt-4o"
   export OPENAI_TEMPERATURE=0.1
   export OPENAI_MAX_TOKENS=450

### Testing
This change was tested by following the instructions in the README.md to set the environment variables and ensuring the tool works correctly.

---

%s`
	return fmt.Sprintf(prompt, diffOutput)
}
