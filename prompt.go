package main

import "fmt"

// CreateOpenAIQuestion formats a question for the OpenAI API based on the git diff output.
func CreateOpenAIQuestion(diffOutput string) string {
	prompt := `
Please generate appropriate pull request message based on the context.

Here is a sample of pull-request format.

---
## Pull Request Title


## Description

* desc 

### Changes
1. change:
   - desc

2. change:
   - desc

3. change:
   - desc

...
...
...

### Testing
- desc

---

%s`
	return fmt.Sprintf(prompt, diffOutput)
}
