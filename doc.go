// Package md2slack converts standard Markdown into Slack-compatible formats.
//
// It provides two conversion functions:
//
//   - [Convert] transforms Markdown into Slack's mrkdwn text format, handling
//     headings, bold, links, strikethrough, numbered lists, code blocks, and
//     proper escaping of Slack's reserved characters.
//
//   - [ConvertToBlocks] transforms Markdown into Slack Block Kit blocks.
//     Currently it wraps the mrkdwn output in a single section block; future
//     versions will produce richer block structures.
package md2slack
