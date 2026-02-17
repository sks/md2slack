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
blocks := md2slack.ConvertToBlocks("# Welcome\n\nHello **world**.\n\n---\n\n![banner](https://example.com/banner.png)")
```

```json
[
  {
    "type": "header",
    "text": {
      "type": "plain_text",
      "text": "Welcome"
    }
  },
  {
    "type": "section",
    "text": {
      "type": "mrkdwn",
      "text": "Hello *world*."
    }
  },
  {
    "type": "divider"
  },
  {
    "type": "image",
    "image_url": "https://example.com/banner.png",
    "alt_text": "banner"
  }
]
```

`ConvertToBlocks` scans the input line-by-line and produces semantically appropriate block types:

| Markdown construct | Block Kit block type |
| :--- | :--- |
| `# Heading` through `###### Heading` | `header` (plain_text) |
| `---`, `***`, `___` | `divider` |
| `![alt](url)` (standalone line) | `image` |
| Fenced code blocks (`` ``` `` / `~~~`) | `section` (mrkdwn with `` ``` `` delimiters) |
| Everything else | `section` (mrkdwn), split at blank lines |

Inline images within text remain as mrkdwn links (`<url|alt>`) inside section blocks.

## Conversion reference

| Markdown | Slack mrkdwn | Notes |
| :--- | :--- | :--- |
| `**bold**` / `__bold__` | `*bold*` | |
| `~~strikethrough~~` | `~strikethrough~` | |
| `[text](url)` | `<url\|text>` | Pipes escaped as `%7C`, nested parens supported |
| `![alt](url)` | `<url\|alt>` | Standalone lines become image blocks in `ConvertToBlocks` |
| `# Heading` | `*Heading*` | All levels h1–h6; header blocks in `ConvertToBlocks` |
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
