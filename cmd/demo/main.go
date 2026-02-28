package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/presmihaylov/md2slack"
	"github.com/slack-go/slack"
)

const sampleMarkdown = `# Hello from md2slack

This is **bold**, _italic_, and ~~strikethrough~~.

Here is some ` + "`inline code`" + ` and a [link](https://github.com/presmihaylov/md2slack).

` + "```go" + `
func main() {
    fmt.Println("Hello, Slack!")
}
` + "```" + `

- bullet one
- bullet two
- bullet three

1. first
2. second
3. third

> A blockquote with a [link](https://example.com)
> spanning multiple lines.

| Feature | Status |
|---------|--------|
| Headings | ✅ |
| Bold/Italic | ✅ |
| Code blocks | ✅ |

---

![placeholder](https://placehold.co/300x200.png)
`

func main() {
	fileInput := flag.String("file-input", "", "read Markdown from a file")
	defaultInput := flag.Bool("default-input", false, "use the built-in sample Markdown")
	threadTS := flag.String("thread-ts", "", "reply into a thread (timestamp or Slack message URL)")
	tsAlias := flag.String("ts", "", "alias for --thread-ts")
	flag.Parse()

	// Resolve --ts alias.
	if *threadTS == "" && *tsAlias != "" {
		*threadTS = *tsAlias
	}

	// Accept a full Slack message URL and extract the timestamp.
	if *threadTS != "" {
		*threadTS = parseThreadTS(*threadTS)
	}

	token := os.Getenv("SLACK_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL")
	if token == "" || channel == "" {
		fmt.Fprintln(os.Stderr, "Set SLACK_TOKEN and SLACK_CHANNEL env vars")
		os.Exit(1)
	}

	md, err := readInput(*fileInput, *defaultInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	blocks, err := md2slack.Convert(md)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert error: %v\n", err)
		os.Exit(1)
	}

	chunks := md2slack.ChunkBlocks(blocks, 50)

	var parentTS string
	if *threadTS != "" {
		parentTS = *threadTS
	}

	for i, chunk := range chunks {
		if len(chunks) > 1 {
			fmt.Fprintf(os.Stderr, "Posting message %d/%d...\n", i+1, len(chunks))
		}
		resp := postBlocks(token, channel, parentTS, chunk)
		fmt.Println("response:", resp)

		// After the first message, thread the rest as replies.
		if parentTS == "" {
			var slackResp struct {
				OK bool   `json:"ok"`
				TS string `json:"ts"`
			}
			if err := json.Unmarshal([]byte(resp), &slackResp); err == nil && slackResp.OK {
				parentTS = slackResp.TS
			}
		}
	}
}

// parseThreadTS accepts either a raw timestamp ("1234567890.123456") or a full
// Slack message URL ("https://…/archives/C…/p1234567890123456") and returns the
// dot-formatted timestamp.
func parseThreadTS(s string) string {
	if !strings.Contains(s, "/") {
		return s
	}
	// Extract the last path segment (e.g. "p1234567890123456").
	seg := path.Base(s)
	// Strip query/fragment if present.
	if i := strings.IndexAny(seg, "?#"); i != -1 {
		seg = seg[:i]
	}
	if len(seg) > 1 && seg[0] == 'p' {
		digits := seg[1:]
		if len(digits) > 6 {
			return digits[:len(digits)-6] + "." + digits[len(digits)-6:]
		}
	}
	return s
}

func readInput(fileInput string, defaultInput bool) (string, error) {
	if fileInput != "" && defaultInput {
		return "", fmt.Errorf("--file-input and --default-input are mutually exclusive")
	}
	if defaultInput {
		return sampleMarkdown, nil
	}
	if fileInput != "" {
		data, err := os.ReadFile(fileInput)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", fileInput, err)
		}
		return string(data), nil
	}
	// Fallback: read from stdin.
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("no input: use --file-input, --default-input, or pipe to stdin")
	}
	return string(data), nil
}

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
	return slackPost(token, payloadJSON)
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
