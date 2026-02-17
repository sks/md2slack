# Demo: posting to Slack

This guide walks you through building a small Go program that converts Markdown to Slack format using md2slack, then posts the result to a Slack channel.

## Prerequisites

| What | How to get it |
|------|---------------|
| **Go 1.22+** | Install from <https://go.dev/dl/> — run `go version` to verify |
| **Slack Bot Token** | Create an app at <https://api.slack.com/apps>, add the `chat:write` bot scope, install to your workspace, copy the `xoxb-…` token |
| **Channel ID** | In Slack, right-click a channel → *View channel details* → copy the ID at the bottom (e.g. `C0123456789`). Invite the bot to the channel |
| **Claude Code** *(optional)* | Install from <https://docs.claude.com/en/docs/setup> — only needed for the Claude-generated example |

## Step 1 — Set up a new Go project

Create a directory for your demo program and initialize a Go module:

```bash
mkdir slack-demo && cd slack-demo
go mod init slack-demo
```

Then add md2slack as a dependency:

```bash
go get github.com/navidemad/md2slack
```

This creates `go.mod` and `go.sum` files that Go uses to track dependencies.

## Step 2 — Write the demo program

Create `main.go` with the following content. It reads Markdown from a file (or uses a built-in sample) and posts it to Slack in two formats:

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

	// Read Markdown from a file argument, or fall back to the built-in sample.
	md := sampleMarkdown
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v\n", os.Args[1], err)
			os.Exit(1)
		}
		md = string(data)
	}

	// Post as mrkdwn text (simple formatting).
	mrkdwn := md2slack.Convert(md)
	postText(token, channel, mrkdwn)

	// Post as Block Kit blocks (rich layout).
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

![gopher](https://go.dev/blog/gopher/header.jpg)
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
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "error: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
```

## Step 3 — Run the demo

Set your Slack credentials and run the program:

```bash
export SLACK_TOKEN="xoxb-your-token"
export SLACK_CHANNEL="C0123456789"
go run main.go
```

This posts two messages to your channel — one using `Convert` (plain mrkdwn) and one using `ConvertToBlocks` (Block Kit blocks). Compare how Slack renders each format.

To use a Markdown file instead of the built-in sample:

```bash
go run main.go /path/to/your/file.md
```

## Optional: generate Markdown with Claude Code

Use Claude Code to produce Markdown that exercises all the formatting features md2slack supports — headings, bold, italic, strikethrough, inline code, code blocks, links, images, ordered and unordered lists, blockquotes, tables, horizontal rules, and reference-style links:

```bash
claude -p \
  'Write a short project overview in Markdown. Include:
   an h1 title with bold text, h2 and h3 subheadings,
   **bold**, _italic_, and ~~strikethrough~~ inline formatting,
   `inline code` spans, a fenced code block with a language tag,
   a [named link](url) and a standalone link on its own line,
   an ![image](url) on its own line,
   a bulleted list and a numbered list,
   a > blockquote,
   a Markdown table with a header row,
   a --- horizontal rule between sections,
   and reference-style links like [text][ref] with [ref]: url definitions at the bottom.' \
  --output-format stream-json 2>/dev/null \
  | jq -r 'select(.type == "result") | .result' \
  > /tmp/claude_output.md
```

Then post it:

```bash
go run main.go /tmp/claude_output.md
```

Or combine generation and posting in one pipeline:

```bash
claude -p "Summarize the latest changes in this repo." \
  --output-format stream-json 2>/dev/null \
  | jq -r 'select(.type == "result") | .result' \
  | go run main.go /dev/stdin
```

## Tip: threading replies

By default the demo posts two separate messages. To make the Block Kit message appear as a reply to the mrkdwn message, you need to:

1. Parse the `"ts"` timestamp from the first Slack response.
2. Include that value as `"thread_ts"` in the second request.

Replace the two posting calls in `main()` with:

```go
	// Post as mrkdwn text (simple formatting).
	mrkdwn := md2slack.Convert(md)
	textResp := postText(token, channel, mrkdwn)

	// Extract the message timestamp from Slack's response.
	var slackResp struct {
		OK bool   `json:"ok"`
		TS string `json:"ts"`
	}
	json.Unmarshal([]byte(textResp), &slackResp)

	// Post Block Kit blocks as a threaded reply.
	blocks := md2slack.ConvertToBlocks(md)
	postBlocksThreaded(token, channel, slackResp.TS, blocks)
```

And add this helper alongside the existing `postBlocks` function:

```go
func postBlocksThreaded(token, channel, threadTS string, blocks []md2slack.Block) {
	blocksJSON, _ := json.Marshal(blocks)
	payload, _ := json.Marshal(map[string]json.RawMessage{
		"channel":   json.RawMessage(`"` + channel + `"`),
		"text":      json.RawMessage(`"fallback text"`),
		"thread_ts": json.RawMessage(`"` + threadTS + `"`),
		"blocks":    blocksJSON,
	})
	resp := slackPost(token, payload)
	fmt.Println("threaded blocks response:", resp)
}
```

You will also need to change `postText` to return the response string (`func postText(...) string`) so the timestamp can be captured.
