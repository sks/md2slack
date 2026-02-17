# md2slack

A Go library that converts standard Markdown into Slack-compatible formats — both [mrkdwn](https://api.slack.com/reference/surfaces/formatting) text and [Block Kit](https://api.slack.com/block-kit) blocks.

## Installation

```bash
go get github.com/navidemad/md2slack
```

Requires Go 1.22+.

## Usage

### Mrkdwn text

```go
import "github.com/navidemad/md2slack"

slack := md2slack.Convert("## Hello\n\nThis is **bold** and a [link](https://example.com).")
// *Hello*
//
// This is *bold* and a <https://example.com|link>.
```

### Block Kit blocks

```go
blocks := md2slack.ConvertToBlocks("Hello **world**")
// []Block{{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Hello *world*"}}}
```

`ConvertToBlocks` currently wraps the mrkdwn output in a single section block. Future versions will produce richer block structures (headers, code blocks, lists as separate blocks).

## Supported Conversions

| Markdown | Slack mrkdwn | Notes |
|---|---|---|
| `**bold**` / `__bold__` | `*bold*` | |
| `~~strikethrough~~` | `~strikethrough~` | |
| `[text](url)` | `<url\|text>` | Pipes in URLs escaped as `%7C`, nested parens supported |
| `# Heading` | `*Heading*` | All heading levels (h1–h6) |
| `1. item` / `1) item` | `- item` | Numbered → bullet lists |
| `` `code` `` | `` `code` `` | Inline code content left untouched |
| ` ``` ` code blocks | ` ``` ` code blocks | Language hint stripped, content preserved |
| `> quote` | `> quote` | Leading `>` preserved, inner `>` escaped |
| `&`, `<`, `>` | `&amp;`, `&lt;`, `&gt;` | Escaped outside code blocks/quotes |

## Testing

```bash
go test ./... -v
```

## Contributing

Fork the repository and open a pull request. Contributions are welcome!

## License

MIT License - see [LICENSE](LICENSE) file for details.
