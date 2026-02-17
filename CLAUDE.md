# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

md2slack is a Go library that converts standard Markdown into Slack-compatible formats. Two public functions: `Convert` (mrkdwn text) and `ConvertToBlocks` (Block Kit blocks). Zero external dependencies — stdlib only. Requires Go 1.22+.

## Commands

- **Run all tests:** `go test ./... -v`
- **Run a single test:** `go test ./... -v -run TestConvert/bold_asterisks`
- **Vet:** `go vet ./...`

## Architecture

Single flat package `md2slack` — no subpackages.

- `mrkdwn.go` — Core `Convert()` function plus all unexported helpers. Processes input line-by-line: code fences toggle pass-through mode, table lines are buffered and flushed as column-aligned monospace wrapped in triple backticks (`formatTable`), all other lines go through `processInlineLine` which splits by backtick pairs (protecting inline code) then applies `applyInlineTransforms` (escaping, headings, bold+italic, bold, links, strikethrough, numbered lists in that order). Bold+italic (`***text***` / `___text___`) is converted to `*_text_*` via `convertBoldItalic` and must run before bold to avoid `***` being consumed as `**` + leftover `*`. Also defines regex patterns used by both `Convert` and `ConvertToBlocks` (`reStandaloneLink`, `reOrderedListItem`, `reUnorderedListItem`, `reTableLine`, `reTableSepCell`). Contains table helpers (`isTableLine`, `isTableSeparator`, `splitTableRow`, `stripInlineMarkdown`, `padCell`, `formatTable`) that detect pipe-delimited rows, strip inline markdown from cells, compute column widths via `utf8.RuneCountInString`, and respect alignment markers (`:---` left, `:---:` center, `---:` right) from separator rows. Contains `resolveReferences()` which pre-processes reference-style links (`[text][ref]`, `![alt][ref]`) by collecting `[ref]: url` definitions into a case-insensitive map and replacing usages with inline form before any other processing. Called at the top of both `Convert` and `ConvertToBlocks`. Contains `preExtractLinks()`/`restoreLinkPlaceholders()` which handle links with backtick-wrapped text (`` [`code`](url) ``) by replacing them with NUL-delimited placeholders before backtick-splitting, then restoring the final Slack links after processing.
- `blockkit.go` — `ConvertToBlocks()` scans input line-by-line and emits semantically appropriate Block Kit blocks: `header`, `divider`, `image`, `section` (mrkdwn), `rich_text` (lists with structured inline elements and nested indent support), `actions` (standalone link buttons), `context` (blockquotes with optional image splitting), and tables (section blocks with code-fenced monospace via `formatTable`). Lists use a `listItem` struct with `indent`, `content`, and `style` fields; `flushList` groups consecutive items by (indent level, style) into separate `rich_text_list` sections with `Indent` set for level > 0, all within a single `rich_text` block. Style changes only trigger a flush at indent level 0, so mixed list types (e.g., ordered parent with bullet sub-list) stay in one block. Tables are buffered in `tableBuf` and flushed as section blocks. Defines types: `Block`, `TextObject`, `RichTextSection`, `RichTextElement`, `RichTextStyle`, `ActionElement`. Custom `MarshalJSON`/`UnmarshalJSON` on `Block` ensures all element arrays serialize under the `"elements"` JSON key as the Slack API expects. Also contains `parseInlineElements()` which parses markdown text into structured rich text elements with priority-based span resolution (code > links > bold+italic > bold > italic > strikethrough). Uses `SortStableFunc` to preserve insertion order (priority) for same-position spans, ensuring bold+italic wins over bold when both match `***text***`. Link and image link spans strip backticks from captured text so `` [`code`](url) `` produces clean link elements.
- `doc.go` — Package-level godoc.
- `example_test.go` — Runnable godoc examples (external test package `md2slack_test`). The `Example*` functions appear as code samples on pkg.go.dev, and `// Output:` comments make them double as regression tests — `go test` verifies the output stays correct.

## Conventions

- Package stays flat — no subpackages unless API surface grows significantly
- Exported names rely on package qualifier: `md2slack.Convert`, not `md2slack.ConvertMarkdownToSlack`
- All regex patterns are compiled once at package level (`var` blocks in `mrkdwn.go` and `blockkit.go`)
- Tests use table-driven style with descriptive sub-test names
- `Convert` must be idempotent — already-converted mrkdwn passes through unchanged (verified by `TestConvert_Idempotent`)
- JSON output from `Block` must match Slack API field names — context, rich_text, and actions blocks all use `"elements"` as the JSON key (handled by custom marshal/unmarshal methods)
