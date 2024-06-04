package main

import "fmt"

type PromptType int

const (
	PrTitle PromptType = iota
	PrBody
)

func CreateOpenAIQuestion(promptType PromptType, diffOutput string) string {
	if promptType == PrTitle {
		return fmt.Sprintf(`
Please generate an appropriate pull request title based on the context.
(Output only the title in one line.)
(Do not output the result of git diff)

%s`, diffOutput)
	} else if promptType == PrBody {
		return fmt.Sprintf(`
Please generate an appropriate pull request description based on the context.
(Do not output the result of git diff)

Here is a sample of pull-request description format.

---
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
%s`, diffOutput)
	}
	return ""
}
