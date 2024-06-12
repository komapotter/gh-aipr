package main

import "fmt"

type PromptType int

const (
	PrTitle PromptType = iota
	PrBody
)

func CreateOpenAIQuestion(promptType PromptType, diffOutput string, japanise bool) string {
	if japanise {
		if promptType == PrTitle {
			return fmt.Sprintf(`
コンテキストに基づいて適切なプルリクエストのタイトルを生成してください。
（タイトルのみを一行で出力してください。）
（git diffの結果を出力しないでください。）

%s`, diffOutput)
		} else if promptType == PrBody {
			return fmt.Sprintf(`
プルリクエストの説明を生成してください。
（git diffの結果を出力しないでください。）

以下はプルリクエスト説明フォーマットのサンプルです。

---
## 説明

* 説明

### 変更点
1. 変更点:
   - 説明

2. 変更点:
   - 説明

3. 変更点:
   - 説明

...
...
...

### テスト
- 説明

---
%s`, diffOutput)
		}
	} else {
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
	}
	return ""
}
