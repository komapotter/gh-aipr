package main

import "fmt"

func systemPrompt(japanise bool) string {
	if japanise {
		return `git diff から GitHub のプルリクエストタイトルと本文を生成してください。

次のキーを持つ JSON オブジェクトだけを返してください:
- title: 簡潔な1行の PR タイトル（引用符で囲まない）
- description: 変更の要約（プレースホルダ禁止）
- changes: 具体的な変更点の配列（1要素が1項目）
- testing: 検証方法

制約:
- タイトルは空でない1行にする。
- JSON を Markdown のコードフェンスで囲まない。
- Markdown 見出しは出力しない。クライアントが ## Description / ### Changes / ### Testing を付与する。
- "desc" / "change:" / "* desc" / "..." などのプレースホルダや埋め用テンプレを使わない。
- git diff を再掲・引用しない。
- diff にないファイルや挙動を作らない。
- title 以外の内容は日本語で書く。`
	}
	return `You generate a GitHub pull request title and body from a git diff.

Return a single JSON object with keys:
- title: a concise single-line PR title (no surrounding quotes)
- description: a short summary of the change (no placeholders)
- changes: an array of concrete change items (one string per item)
- testing: how a reviewer can verify the change

Constraints:
- Title must be one non-empty line.
- Do not wrap the JSON in markdown code fences.
- Do not include Markdown headings; the client adds ## Description, ### Changes, and ### Testing.
- Do not use placeholders or fill-in templates such as "desc", "change:", "* desc", or "...".
- Do not restate or quote the git diff.
- Do not invent files or behavior that are not in the diff.
- Write in English.`
}

func retryFixInstruction(err error, japanise bool) string {
	if japanise {
		return fmt.Sprintf("前回の出力は検証に失敗しました: %s。title, description, changes, testing を持つ JSON だけを返してください。コードフェンス・プレースホルダ・diff の再掲は禁止です。", err)
	}
	return fmt.Sprintf("Previous output failed validation: %s. Return corrected JSON only (title, description, changes, testing). No markdown fences, no placeholders, no restated diff.", err)
}
