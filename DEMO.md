# Demo: posting to Slack

This guide shows how to write a small Go program that uses md2slack to convert Markdown and post the result to a Slack channel via the [chat.postMessage](https://api.slack.com/methods/chat.postMessage) API.

The example uses [Claude Code](https://claude.ai/code) to generate real Markdown content, then pipes it through md2slack into Slack -- a realistic workflow for AI-powered notifications and reports.

## Prerequisites

1. **Slack Bot Token** -- Create a Slack app at <https://api.slack.com/apps>, add the `chat:write` bot scope, install it to your workspace, and copy the `xoxb-...` token.
2. **Channel ID** -- Right-click a channel in Slack, select *View channel details*, and copy the ID at the bottom (e.g. `C0123456789`). The bot must be a member of the channel.
3. **Claude Code** -- Install via `npm install -g @anthropic-ai/claude-code` (optional, only needed for the Claude-sourced example).

## Generate Markdown with Claude Code

Use Claude Code to produce Markdown that exercises headings, bold, code blocks, lists, and links:

```bash
claude -p \
  "Explain the architecture of this project briefly. \
   Use markdown headings, bold, code examples with language tags, \
   and link references." \
  --output-format stream-json 2>/dev/null \
  | jq -r 'select(.type == "result") | .result' \
  > /tmp/claude_output.md
```

This saves the final Markdown to `/tmp/claude_output.md`. You can inspect it:

```bash
cat /tmp/claude_output.md
```

## Script

Create a file (e.g. `cmd/demo/main.go`) outside the library. It reads Markdown from a file (or falls back to a built-in sample) and posts it to Slack using both output modes:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/navidemad/md2slack"
)

func main() {
	token := os.Getenv("SLACK_TOKEN")     // xoxb-...
	channel := os.Getenv("SLACK_CHANNEL") // C0123456789

	if token == "" || channel == "" {
		fmt.Fprintln(os.Stderr, "Set SLACK_TOKEN and SLACK_CHANNEL env vars")
		os.Exit(1)
	}

	// Read Markdown from file argument, or use a built-in sample.
	md := sampleMarkdown
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v\n", os.Args[1], err)
			os.Exit(1)
		}
		md = string(data)
	}

	// --- Post as mrkdwn text ---
	mrkdwn := md2slack.Convert(md)
	postText(token, channel, mrkdwn)

	// --- Post as Block Kit blocks ---
	blocks := md2slack.ConvertToBlocks(md)
	postBlocks(token, channel, blocks)
}

const sampleMarkdown = `# Hello from md2slack

This is **bold**, _italic_, and ~~strikethrough~~.

- bullet one
- bullet two

1. first
2. second

> A blockquote with a [link](https://example.com)

![gopher](https://go.dev/images/gopher/header.jpg)
`

func postText(token, channel, text string) {
	payload, _ := json.Marshal(map[string]string{
		"channel": channel,
		"text":    text,
	})
	resp := slackPost(token, payload)
	fmt.Println("text response:", resp)
}

func postBlocks(token, channel string, blocks []md2slack.Block) {
	blocksJSON, _ := json.Marshal(blocks)
	payload, _ := json.Marshal(map[string]json.RawMessage{
		"channel": json.RawMessage(`"` + channel + `"`),
		"text":    json.RawMessage(`"fallback text"`),
		"blocks":  blocksJSON,
	})
	resp := slackPost(token, payload)
	fmt.Println("blocks response:", resp)
}

func slackPost(token string, payload []byte) string {
	req, _ := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
```

## Run

With the built-in sample:

```bash
export SLACK_TOKEN="xoxb-your-token"
export SLACK_CHANNEL="C0123456789"
go run ./cmd/demo/
```

With Claude-generated Markdown:

```bash
go run ./cmd/demo/ /tmp/claude_output.md
```

This posts two messages to the channel: one using `Convert` (plain mrkdwn) and one using `ConvertToBlocks` (Block Kit blocks). Compare how Slack renders each format.

## One-liner: Claude to Slack

Combine generation and posting in a single pipeline:

```bash
claude -p "Summarize the latest changes in this repo." \
  --output-format stream-json 2>/dev/null \
  | jq -r 'select(.type == "result") | .result' \
  | go run ./cmd/demo/ /dev/stdin
```

## Threading

To post both messages in the same thread, capture the `"ts"` field from the first response and include it as `"thread_ts"` in the second payload:

```go
var result struct{ TS string `json:"ts"` }
json.Unmarshal([]byte(resp), &result)
// add "thread_ts": result.TS to the second payload
```
