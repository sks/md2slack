# Demo: posting to Slack

This guide walks you through building a small Go program that converts Markdown to Slack Block Kit blocks using md2slack, then posts the result to a Slack channel.

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

Then add md2slack and slack-go as dependencies:

```bash
go get github.com/navidemad/md2slack
go get github.com/slack-go/slack
```

This creates `go.mod` and `go.sum` files that Go uses to track dependencies.

## Step 2 — Write the demo program

Create `main.go` with the following content. It reads Markdown from a file (or uses a built-in sample), converts it to Slack Block Kit blocks, and posts them to a channel:

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
	"github.com/slack-go/slack"
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

	// Convert Markdown to Slack Block Kit blocks.
	blocks, err := md2slack.Convert(md)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert error: %v\n", err)
		os.Exit(1)
	}

	// Split into chunks of 50 blocks (Slack's per-message limit).
	chunks := md2slack.ChunkBlocks(blocks, 50)

	for i, chunk := range chunks {
		if len(chunks) > 1 {
			fmt.Fprintf(os.Stderr, "Posting message %d/%d...\n", i+1, len(chunks))
		}
		postBlocks(token, channel, chunk)
	}
}

const sampleMarkdown = `# Hello from md2slack

This is **bold**, _italic_, and ~~strikethrough~~.

- bullet one
- bullet two

1. first
2. second

> A blockquote with a [link](https://example.com)

![gopher](https://go.dev/doc/gopher/frontpage.png)
`

func postBlocks(token, channel string, blocks []slack.Block) {
	blocksJSON, _ := json.Marshal(blocks)
	payload, _ := json.Marshal(map[string]json.RawMessage{
		"channel": json.RawMessage(`"` + channel + `"`),
		"text":    json.RawMessage(`"Markdown message"`),
		"blocks":  blocksJSON,
	})
	resp := slackPost(token, payload)
	fmt.Println("response:", resp)
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

This converts the sample Markdown to Block Kit blocks and posts them to your channel. If the output exceeds 50 blocks, it automatically splits across multiple messages using `ChunkBlocks`.

To use a Markdown file instead of the built-in sample:

```bash
go run main.go /path/to/your/file.md
```

## Optional: generate Markdown with Claude Code

Use Claude Code to produce Markdown that exercises all the formatting features md2slack supports — headings, bold, italic, strikethrough, inline code, code blocks, links, images, ordered and unordered lists, blockquotes, tables, horizontal rules, and reference-style links:

```bash
claude -p \
  'Print a detailed Markdown document (at least 80 lines) about a fictional
   Go CLI tool called "taskrunner" that executes build pipelines. Use every
   one of these Markdown features with real content:

   - An h1 title, an h2 section heading, and an h3 sub-heading
   - **bold**, _italic_, and ~~strikethrough~~ text inline within sentences
   - `inline code` for function names, commands, and variables
   - A fenced code block with a Go language tag containing a real code sample
   - A fenced code block with a bash language tag showing CLI usage
   - An inline [named link](https://example.com) within a paragraph
   - A standalone link on its own line: [Link text](url)
   - An image on its own line: ![alt text](https://via.placeholder.com/600x200)
   - A bulleted list with at least 3 items
   - A numbered list with at least 3 items
   - A blockquote (> ) with at least two lines
   - A Markdown table with a header row, separator, and at least 3 data rows
   - A horizontal rule (---) separating two sections
   - At least two reference-style links using [text][ref] syntax with
     [ref]: url definitions at the bottom of the document

   Output only raw Markdown.' \
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

To post multiple chunk messages as a thread (so only the first appears in the channel), capture the `"ts"` timestamp from the first response and pass it as `"thread_ts"` in subsequent requests.

Replace the posting loop in `main()` with:

```go
	var threadTS string
	for i, chunk := range chunks {
		if len(chunks) > 1 {
			fmt.Fprintf(os.Stderr, "Posting message %d/%d...\n", i+1, len(chunks))
		}
		resp := postBlocks(token, channel, threadTS, chunk)

		// After the first message, thread the rest as replies.
		if threadTS == "" {
			var slackResp struct {
				OK bool   `json:"ok"`
				TS string `json:"ts"`
			}
			json.Unmarshal([]byte(resp), &slackResp)
			if slackResp.OK {
				threadTS = slackResp.TS
			}
		}
	}
```

And update `postBlocks` to accept and use the thread timestamp:

```go
func postBlocks(token, channel, threadTS string, blocks []slack.Block) string {
	blocksJSON, _ := json.Marshal(blocks)
	payload := map[string]json.RawMessage{
		"channel": json.RawMessage(`"` + channel + `"`),
		"text":    json.RawMessage(`"Markdown message"`),
		"blocks":  blocksJSON,
	}
	if threadTS != "" {
		payload["thread_ts"] = json.RawMessage(`"` + threadTS + `"`)
	}
	payloadJSON, _ := json.Marshal(payload)
	resp := slackPost(token, payloadJSON)
	fmt.Println("response:", resp)
	return resp
}
```
