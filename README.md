# md2slack

[![Go Reference](https://pkg.go.dev/badge/github.com/navidemad/md2slack.svg)](https://pkg.go.dev/github.com/navidemad/md2slack)
[![CI](https://github.com/navidemad/md2slack/actions/workflows/ci.yml/badge.svg)](https://github.com/navidemad/md2slack/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/navidemad/md2slack)](https://goreportcard.com/report/github.com/navidemad/md2slack)

> Convert standard Markdown into Slack-compatible formats — [mrkdwn] text and [Block Kit] blocks.

- **Zero dependencies** — stdlib only
- **Idempotent** — already-converted mrkdwn passes through unchanged
- **Two output modes** — plain mrkdwn text or structured Block Kit blocks
- **Rich Block Kit output** — headers, dividers, images, rich text lists, action buttons, and context blocks
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
    "text": { "type": "plain_text", "text": "Welcome" }
  },
  {
    "type": "section",
    "text": { "type": "mrkdwn", "text": "Hello *world*." }
  },
  { "type": "divider" },
  {
    "type": "image",
    "image_url": "https://example.com/banner.png",
    "alt_text": "banner",
    "title": { "type": "plain_text", "text": "banner" }
  }
]
```

`ConvertToBlocks` scans the input line-by-line and produces semantically appropriate block types:

| Markdown construct | Block Kit block type |
| :--- | :--- |
| `# Heading` through `###### Heading` | `header` (plain\_text) |
| `---`, `***`, `___` | `divider` |
| `![alt](url)` (standalone line) | `image` |
| `[text](url)` (standalone line) | `actions` (button element) |
| `1. item` / `- item` / `* item` | `rich_text` (`rich_text_list` with ordered/bullet style) |
| `> quote` | `context` (mrkdwn elements; images split out) |
| Fenced code blocks (`` ``` `` / `~~~`) | `section` (mrkdwn with `` ``` `` delimiters) |
| Everything else | `section` (mrkdwn), split at blank lines |

Inline images within text remain as mrkdwn links (`<url|alt>`) inside section blocks.

### Ordered and unordered lists

Lists are emitted as `rich_text` blocks with proper list semantics, preserving ordered vs. unordered style:

```go
blocks := md2slack.ConvertToBlocks("1. First item\n2. Second item")
```

```json
[
  {
    "type": "rich_text",
    "elements": [
      {
        "type": "rich_text_list",
        "style": "ordered",
        "items": [
          {
            "type": "rich_text_section",
            "elements": [{ "type": "text", "text": "First item" }]
          },
          {
            "type": "rich_text_section",
            "elements": [{ "type": "text", "text": "Second item" }]
          }
        ]
      }
    ]
  }
]
```

Inline formatting within list items is parsed into structured elements with style flags (`bold`, `italic`, `strikethrough`, `code`) rather than mrkdwn strings.

### Standalone links as buttons

A line containing only a markdown link becomes an `actions` block with a clickable button:

```go
blocks := md2slack.ConvertToBlocks("[Click here](https://example.com)")
```

```json
[
  {
    "type": "actions",
    "elements": [
      {
        "type": "button",
        "text": { "type": "plain_text", "text": "Click here" },
        "url": "https://example.com"
      }
    ]
  }
]
```

Links embedded within text continue to render as mrkdwn links in section blocks.

### Blockquotes with images

Blockquotes containing image references split them into separate context elements:

```go
blocks := md2slack.ConvertToBlocks("> Check this ![icon](https://example.com/icon.png) out")
```

```json
[
  {
    "type": "context",
    "elements": [
      { "type": "mrkdwn", "text": "Check this" },
      { "type": "image", "image_url": "https://example.com/icon.png", "alt_text": "icon" },
      { "type": "mrkdwn", "text": "out" }
    ]
  }
]
```

## Conversion reference

| Markdown | Slack mrkdwn | Notes |
| :--- | :--- | :--- |
| `**bold**` / `__bold__` | `*bold*` | |
| `~~strikethrough~~` | `~strikethrough~` | |
| `[text](url)` | `<url\|text>` | Pipes escaped as `%7C`, nested parens supported |
| `_italic_` | `_italic_` | Underscore italic passes through unchanged |
| `![alt](url)` | `<url\|alt>` | Standalone lines become image blocks in `ConvertToBlocks` |
| `# Heading` | `*Heading*` | All levels h1–h6; header blocks in `ConvertToBlocks` |
| `1. item` / `1) item` | `- item` | Rich text list blocks in `ConvertToBlocks` |
| `* item` / `+ item` | `- item` | Rich text list blocks in `ConvertToBlocks` |
| `` `code` `` | `` `code` `` | Content left untouched |
| ` ``` ` code blocks | ` ``` ` code blocks | Language hint stripped, content preserved |
| `> quote` | `> quote` | Leading `>` preserved, inner `>` escaped |
| `&`, `<`, `>` | `&amp;`, `&lt;`, `&gt;` | Escaped outside code blocks and quotes |

## Types

`ConvertToBlocks` returns `[]Block`. The key types:

| Type | Purpose |
| :--- | :--- |
| `Block` | A single Block Kit layout block (`section`, `header`, `divider`, `image`, `rich_text`, `actions`, `context`) |
| `TextObject` | Text composition object (`mrkdwn` or `plain_text`); also used for image elements in context blocks |
| `RichTextSection` | Section within a `rich_text` block (`rich_text_section`, `rich_text_list`, `rich_text_preformatted`, `rich_text_quote`) |
| `RichTextElement` | Inline element (`text` or `link`) with optional `Style` |
| `RichTextStyle` | Formatting flags: `Bold`, `Italic`, `Strikethrough`, `Code` |
| `ActionElement` | Interactive element (`button`) with `Text` and `URL` |

All types serialize to JSON matching the [Slack Block Kit specification][Block Kit].

## Contributing

Fork the repository and open a pull request. Contributions are welcome!

## License

[MIT](LICENSE)

[mrkdwn]: https://api.slack.com/reference/surfaces/formatting
[Block Kit]: https://api.slack.com/block-kit
