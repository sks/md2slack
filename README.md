# md2slack

[![Go Reference](https://pkg.go.dev/badge/github.com/navidemad/md2slack.svg)](https://pkg.go.dev/github.com/navidemad/md2slack)
[![CI](https://github.com/navidemad/md2slack/actions/workflows/ci.yml/badge.svg)](https://github.com/navidemad/md2slack/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/navidemad/md2slack)](https://goreportcard.com/report/github.com/navidemad/md2slack)

> Convert standard Markdown into Slack-compatible formats — [mrkdwn] text and [Block Kit] blocks.

- **Zero dependencies** — stdlib only
- **Idempotent** — already-converted mrkdwn passes through unchanged
- **Two output modes** — plain mrkdwn text or structured Block Kit blocks
- **Safe** — escapes `&`, `<`, `>` and protects code spans from transformation

## Install

```bash
go get github.com/navidemad/md2slack
```

Requires **Go 1.22+**.

## Quick start

### Mrkdwn text

```go
import "github.com/navidemad/md2slack"

out := md2slack.Convert("## Hello\n\nThis is **bold** and a [link](https://example.com).")
```

Output:

```
*Hello*

This is *bold* and a <https://example.com|link>.
```

### Block Kit blocks

```go
blocks := md2slack.ConvertToBlocks("Hello **world**")
```

```json
[
  {
    "type": "section",
    "text": {
      "type": "mrkdwn",
      "text": "Hello *world*"
    }
  }
]
```

`ConvertToBlocks` currently wraps the mrkdwn output in a single section block. Future versions will produce richer block structures (headers, code blocks, lists as separate blocks).

## Conversion reference

| Markdown | Slack mrkdwn | Notes |
| :--- | :--- | :--- |
| `**bold**` / `__bold__` | `*bold*` | |
| `~~strikethrough~~` | `~strikethrough~` | |
| `[text](url)` | `<url\|text>` | Pipes escaped as `%7C`, nested parens supported |
| `# Heading` | `*Heading*` | All levels h1–h6 |
| `1. item` / `1) item` | `- item` | Numbered lists become bullet lists |
| `` `code` `` | `` `code` `` | Content left untouched |
| ` ``` ` code blocks | ` ``` ` code blocks | Language hint stripped, content preserved |
| `> quote` | `> quote` | Leading `>` preserved, inner `>` escaped |
| `&`, `<`, `>` | `&amp;`, `&lt;`, `&gt;` | Escaped outside code blocks and quotes |

## Contributing

Fork the repository and open a pull request. Contributions are welcome!

## License

[MIT](LICENSE)

[mrkdwn]: https://api.slack.com/reference/surfaces/formatting
[Block Kit]: https://api.slack.com/block-kit
