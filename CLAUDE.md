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

- `mrkdwn.go` — Core `Convert()` function plus all unexported helpers. Processes input line-by-line: code fences toggle pass-through mode, all other lines go through `processInlineLine` which splits by backtick pairs (protecting inline code) then applies `applyInlineTransforms` (escaping, headings, bold, links, strikethrough, numbered lists in that order). Also defines regex patterns used by both `Convert` and `ConvertToBlocks` (`reStandaloneLink`, `reOrderedListItem`, `reUnorderedListItem`). Contains `resolveReferences()` which pre-processes reference-style links (`[text][ref]`, `![alt][ref]`) by collecting `[ref]: url` definitions into a case-insensitive map and replacing usages with inline form before any other processing. Called at the top of both `Convert` and `ConvertToBlocks`.
- `blockkit.go` — `ConvertToBlocks()` scans input line-by-line and emits semantically appropriate Block Kit blocks: `header`, `divider`, `image`, `section` (mrkdwn), `rich_text` (lists with structured inline elements), `actions` (standalone link buttons), and `context` (blockquotes with optional image splitting). Defines types: `Block`, `TextObject`, `RichTextSection`, `RichTextElement`, `RichTextStyle`, `ActionElement`. Custom `MarshalJSON`/`UnmarshalJSON` on `Block` ensures all element arrays serialize under the `"elements"` JSON key as the Slack API expects. Also contains `parseInlineElements()` which parses markdown text into structured rich text elements with priority-based span resolution (code > links > bold > italic > strikethrough).
- `doc.go` — Package-level godoc.
- `example_test.go` — Runnable godoc examples (external test package `md2slack_test`). The `Example*` functions appear as code samples on pkg.go.dev, and `// Output:` comments make them double as regression tests — `go test` verifies the output stays correct.

## Conventions

- Package stays flat — no subpackages unless API surface grows significantly
- Exported names rely on package qualifier: `md2slack.Convert`, not `md2slack.ConvertMarkdownToSlack`
- All regex patterns are compiled once at package level (`var` blocks in `mrkdwn.go` and `blockkit.go`)
- Tests use table-driven style with descriptive sub-test names
- `Convert` must be idempotent — already-converted mrkdwn passes through unchanged (verified by `TestConvert_Idempotent`)
- JSON output from `Block` must match Slack API field names — context, rich_text, and actions blocks all use `"elements"` as the JSON key (handled by custom marshal/unmarshal methods)
