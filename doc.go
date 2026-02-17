// Package md2slack converts standard Markdown into Slack-compatible formats.
//
// It provides two conversion functions:
//
//   - [Convert] transforms Markdown into Slack's mrkdwn text format, suitable
//     for use in message text, attachments, and any Slack API field that accepts
//     mrkdwn.
//
//   - [ConvertToBlocks] transforms Markdown into Slack [Block Kit] blocks,
//     suitable for use in the "blocks" field of Slack API payloads.
//
// # Supported Markdown features
//
//   - Headings (## Heading → *Heading*)
//   - Bold (**text** and __text__ → *text*)
//   - Strikethrough (~~text~~ → ~text~)
//   - Links ([text](url) → <url|text>)
//   - Image links (![alt](url) → <url|alt>)
//   - Numbered lists (1. item → - item)
//   - Fenced code blocks (``` and ~~~ preserved as-is)
//   - Inline code (protected from transformation)
//   - Block quotes (> preserved)
//   - Automatic escaping of &, <, > for Slack
//
// # Idempotency
//
// [Convert] is idempotent: passing already-converted mrkdwn through a second
// time produces the same output. This makes it safe to apply unconditionally
// without tracking whether text has already been converted.
//
// # Zero dependencies
//
// The package uses only the Go standard library.
//
// [Block Kit]: https://api.slack.com/block-kit
package md2slack
