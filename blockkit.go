package md2slack

import "strings"

// Block represents a single Slack Block Kit layout block.
//
// For "image" blocks, ImageURL and AltText are required; Title is optional
// and displayed above the image when set.
//
// For "context" blocks, Elements holds the text objects displayed in the block.
//
// See https://api.slack.com/reference/block-kit/blocks for the full
// Block Kit specification.
type Block struct {
	// Type identifies the block kind (e.g. "section", "divider", "header", "image", "context").
	Type string `json:"type"`

	// Text holds the block's text content. Nil for block types that don't
	// carry text (e.g. "divider", "context").
	Text *TextObject `json:"text,omitempty"`

	// Elements holds text objects for "context" blocks.
	Elements []TextObject `json:"elements,omitempty"`

	// ImageURL is the URL of the image for "image" blocks.
	ImageURL string `json:"image_url,omitempty"`

	// AltText is the alt text for "image" blocks.
	AltText string `json:"alt_text,omitempty"`

	// Title is an optional title for "image" blocks.
	Title *TextObject `json:"title,omitempty"`
}

// TextObject represents a Slack Block Kit text composition object.
//
// See https://api.slack.com/reference/block-kit/composition-objects#text
// for the full specification.
type TextObject struct {
	// Type is either "mrkdwn" or "plain_text".
	Type string `json:"type"`

	// Text is the content string, formatted according to Type.
	Text string `json:"text"`
}

// ConvertToBlocks transforms Markdown into Slack Block Kit blocks.
//
// It scans the input line-by-line, producing semantically appropriate block
// types:
//
//   - Headings (# … through ###### …) become "header" blocks with plain_text
//   - Horizontal rules (---, ***, ___) become "divider" blocks
//   - Blockquotes (> text) become "context" blocks with mrkdwn elements
//   - Standalone images (![alt](url) on their own line) become "image" blocks
//   - Fenced code blocks (``` or ~~~) become "section" blocks with code fences
//   - All other text is accumulated into "section" blocks with mrkdwn, split
//     at blank lines (paragraph boundaries)
//
// Consecutive blockquote lines are merged into a single context block.
// Inline images within text remain as mrkdwn links in section blocks.
// Text segments are processed through [Convert] for inline formatting.
//
// An empty string returns a single empty section block for backward
// compatibility.
func ConvertToBlocks(markdown string) []Block {
	if markdown == "" {
		return []Block{
			{
				Type: "section",
				Text: &TextObject{
					Type: "mrkdwn",
					Text: "",
				},
			},
		}
	}

	lines := strings.Split(markdown, "\n")
	var blocks []Block
	var textBuf []string
	var quoteBuf []string
	inCodeBlock := false
	var codeBuf []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code fence toggle.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if inCodeBlock {
				// Closing fence — emit the code block.
				blocks = appendCodeBlock(blocks, codeBuf)
				codeBuf = nil
				inCodeBlock = false
			} else {
				// Opening fence — flush any pending text first.
				blocks = flushTextBuffer(blocks, textBuf)
				textBuf = nil
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			codeBuf = append(codeBuf, line)
			continue
		}

		// Block quote → context block.
		if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			// Strip the > prefix.
			var content string
			if trimmed == ">" {
				content = ""
			} else {
				idx := strings.Index(line, ">")
				content = line[idx+1:]
				if strings.HasPrefix(content, " ") {
					content = content[1:]
				}
			}
			quoteBuf = append(quoteBuf, content)
			continue
		}

		// If we were in a blockquote and hit a non-blockquote line, flush it.
		if len(quoteBuf) > 0 {
			blocks = flushQuoteBuffer(blocks, quoteBuf)
			quoteBuf = nil
		}

		// Horizontal rule → divider block.
		if reHorizontalRule.MatchString(line) {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			blocks = append(blocks, Block{Type: "divider"})
			continue
		}

		// Heading → header block with plain_text.
		if m := reHeading.FindStringSubmatch(trimmed); m != nil && strings.TrimSpace(m[1]) != "" {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			blocks = append(blocks, Block{
				Type: "header",
				Text: &TextObject{
					Type: "plain_text",
					Text: extractHeadingText(m[1]),
				},
			})
			continue
		}

		// Standalone image → image block.
		if m := reStandaloneImage.FindStringSubmatch(line); m != nil && m[2] != "" {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			alt := m[1]
			if alt == "" {
				alt = " "
			}
			block := Block{
				Type:     "image",
				ImageURL: m[2],
				AltText:  alt,
			}
			if m[1] != "" {
				block.Title = &TextObject{
					Type: "plain_text",
					Text: m[1],
				}
			}
			blocks = append(blocks, block)
			continue
		}

		// Blank line → paragraph boundary.
		if trimmed == "" {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			continue
		}

		// Regular text — accumulate.
		textBuf = append(textBuf, line)
	}

	// Flush remaining content.
	if inCodeBlock {
		// Unclosed code fence — emit what we have.
		blocks = appendCodeBlock(blocks, codeBuf)
	} else {
		blocks = flushTextBuffer(blocks, textBuf)
	}
	if len(quoteBuf) > 0 {
		blocks = flushQuoteBuffer(blocks, quoteBuf)
	}

	if len(blocks) == 0 {
		return []Block{
			{
				Type: "section",
				Text: &TextObject{
					Type: "mrkdwn",
					Text: "",
				},
			},
		}
	}

	return blocks
}

// flushTextBuffer joins accumulated lines, runs them through Convert, and
// appends a section block. Returns blocks unchanged if lines is empty.
func flushTextBuffer(blocks []Block, lines []string) []Block {
	if len(lines) == 0 {
		return blocks
	}
	text := Convert(strings.Join(lines, "\n"))
	return append(blocks, Block{
		Type: "section",
		Text: &TextObject{
			Type: "mrkdwn",
			Text: text,
		},
	})
}

// flushQuoteBuffer joins accumulated blockquote lines, runs them through
// Convert for inline formatting, and appends a context block. Returns blocks
// unchanged if lines is empty.
func flushQuoteBuffer(blocks []Block, lines []string) []Block {
	if len(lines) == 0 {
		return blocks
	}
	text := Convert(strings.Join(lines, "\n"))
	return append(blocks, Block{
		Type: "context",
		Elements: []TextObject{
			{Type: "mrkdwn", Text: text},
		},
	})
}

// appendCodeBlock wraps code lines in ``` delimiters and appends as a section block.
func appendCodeBlock(blocks []Block, lines []string) []Block {
	var text string
	if len(lines) == 0 {
		text = "```\n```"
	} else {
		text = "```\n" + strings.Join(lines, "\n") + "\n```"
	}
	return append(blocks, Block{
		Type: "section",
		Text: &TextObject{
			Type: "mrkdwn",
			Text: text,
		},
	})
}

// extractHeadingText strips # prefix and bold markers from a heading content
// string, returning plain text suitable for a header block's plain_text field.
func extractHeadingText(content string) string {
	content = strings.TrimSpace(content)

	// Strip bold markers (redundant in headers).
	for {
		prev := content
		content = reConsecStars.ReplaceAllString(content, "")
		content = reConsecUnders.ReplaceAllString(content, "")
		content = strings.TrimLeft(content, "*")
		content = strings.TrimRight(content, "*")
		content = strings.TrimSpace(content)
		// Strip nested heading markers.
		if inner := reHeading.FindStringSubmatch(content); inner != nil {
			content = strings.TrimSpace(inner[1])
		}
		if content == prev {
			break
		}
	}

	return content
}
