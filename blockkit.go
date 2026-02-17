package md2slack

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
)

// Package-level compiled regexps for inline element parsing.
var (
	reInlineBold      = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reInlineBoldUnder = regexp.MustCompile(`__(.+?)__`)
	reInlineItalic    = regexp.MustCompile(`(?:^|[\s(])_([^_]+?)_(?:$|[\s).,;:!?])`)
	reInlineStrike    = regexp.MustCompile(`~~(.+?)~~`)
	reInlineCode      = regexp.MustCompile("`([^`]+)`")
)

// Block represents a single Slack Block Kit layout block.
//
// For "image" blocks, ImageURL and AltText are required; Title is optional
// and displayed above the image when set.
//
// For "context" blocks, Elements holds the text objects displayed in the block.
//
// For "rich_text" blocks, RichElements holds the rich text sections.
//
// For "actions" blocks, ActionElements holds the interactive elements.
//
// See https://api.slack.com/reference/block-kit/blocks for the full
// Block Kit specification.
type Block struct {
	// Type identifies the block kind (e.g. "section", "divider", "header",
	// "image", "context", "rich_text", "actions").
	Type string `json:"type"`

	// Text holds the block's text content. Nil for block types that don't
	// carry text (e.g. "divider", "context").
	Text *TextObject `json:"text,omitempty"`

	// Elements holds text objects for "context" blocks.
	Elements []TextObject `json:"-"`

	// ImageURL is the URL of the image for "image" blocks.
	ImageURL string `json:"image_url,omitempty"`

	// AltText is the alt text for "image" blocks.
	AltText string `json:"alt_text,omitempty"`

	// Title is an optional title for "image" blocks.
	Title *TextObject `json:"title,omitempty"`

	// RichElements holds rich text section elements for "rich_text" blocks.
	RichElements []RichTextSection `json:"-"`

	// ActionElements holds interactive elements for "actions" blocks.
	ActionElements []ActionElement `json:"-"`
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

	// ImageURL is the URL for "image" type elements in context blocks.
	ImageURL string `json:"image_url,omitempty"`

	// AltText is the alt text for "image" type elements in context blocks.
	AltText string `json:"alt_text,omitempty"`
}

// RichTextSection represents a section within a rich_text block.
//
// Type is one of "rich_text_section", "rich_text_preformatted",
// "rich_text_quote", or "rich_text_list".
type RichTextSection struct {
	// Type identifies the section kind.
	Type string `json:"type"`

	// Elements holds the inline elements for this section.
	// Used by rich_text_section, rich_text_preformatted, and rich_text_quote.
	Elements []RichTextElement `json:"elements,omitempty"`

	// Style is the list style for rich_text_list ("ordered" or "bullet").
	Style string `json:"style,omitempty"`

	// Indent is the indentation level for rich_text_list items.
	Indent int `json:"indent,omitempty"`

	// Items holds the list item sections for rich_text_list.
	// Each item is a rich_text_section.
	Items []RichTextSection `json:"items,omitempty"`
}

// RichTextElement represents an inline element within a rich text section.
//
// Type is one of "text" or "link".
type RichTextElement struct {
	// Type identifies the element kind ("text" or "link").
	Type string `json:"type"`

	// Text is the text content for "text" elements, or display text for "link" elements.
	Text string `json:"text,omitempty"`

	// URL is the URL for "link" elements.
	URL string `json:"url,omitempty"`

	// Style holds formatting flags for the element.
	Style *RichTextStyle `json:"style,omitempty"`
}

// RichTextStyle holds boolean formatting flags for rich text elements.
type RichTextStyle struct {
	// Bold indicates bold formatting.
	Bold bool `json:"bold,omitempty"`

	// Italic indicates italic formatting.
	Italic bool `json:"italic,omitempty"`

	// Strikethrough indicates strikethrough formatting.
	Strikethrough bool `json:"strikethrough,omitempty"`

	// Code indicates inline code formatting.
	Code bool `json:"code,omitempty"`
}

// ActionElement represents an interactive element within an "actions" block.
type ActionElement struct {
	// Type identifies the element kind (e.g. "button").
	Type string `json:"type"`

	// Text is the display text for the element.
	Text TextObject `json:"text"`

	// URL is the URL for button elements.
	URL string `json:"url,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for Block.
//
// The Slack API expects "context", "rich_text", and "actions" blocks to all
// use the JSON key "elements" for their child arrays, even though the element
// types differ. This method serializes the appropriate Go field (Elements,
// RichElements, or ActionElements) under the unified "elements" key based on
// the block's Type.
func (b Block) MarshalJSON() ([]byte, error) {
	// blockAlias prevents infinite recursion by hiding the MarshalJSON method.
	type blockAlias Block

	switch b.Type {
	case "context":
		if len(b.Elements) == 0 {
			return json.Marshal(struct {
				blockAlias
			}{blockAlias: blockAlias(b)})
		}
		return json.Marshal(struct {
			blockAlias
			Elements []TextObject `json:"elements"`
		}{
			blockAlias: blockAlias(b),
			Elements:   b.Elements,
		})
	case "rich_text":
		if len(b.RichElements) == 0 {
			return json.Marshal(struct {
				blockAlias
			}{blockAlias: blockAlias(b)})
		}
		return json.Marshal(struct {
			blockAlias
			Elements []RichTextSection `json:"elements"`
		}{
			blockAlias:  blockAlias(b),
			Elements: b.RichElements,
		})
	case "actions":
		if len(b.ActionElements) == 0 {
			return json.Marshal(struct {
				blockAlias
			}{blockAlias: blockAlias(b)})
		}
		return json.Marshal(struct {
			blockAlias
			Elements []ActionElement `json:"elements"`
		}{
			blockAlias:  blockAlias(b),
			Elements: b.ActionElements,
		})
	default:
		return json.Marshal(struct {
			blockAlias
		}{blockAlias: blockAlias(b)})
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface for Block.
//
// It reads the "type" field first, then deserializes the "elements" JSON key
// into the appropriate Go field (Elements, RichElements, or ActionElements)
// based on the block type.
func (b *Block) UnmarshalJSON(data []byte) error {
	// blockAlias prevents infinite recursion by hiding the UnmarshalJSON method.
	type blockAlias Block

	// First pass: unmarshal everything except the elements fields (which are
	// tagged json:"-" and thus ignored by the default decoder).
	var alias blockAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*b = Block(alias)

	// Second pass: extract the raw "elements" value and decode into the right field.
	var raw struct {
		Elements json.RawMessage `json:"elements"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Elements == nil {
		return nil
	}

	switch b.Type {
	case "context":
		return json.Unmarshal(raw.Elements, &b.Elements)
	case "rich_text":
		return json.Unmarshal(raw.Elements, &b.RichElements)
	case "actions":
		return json.Unmarshal(raw.Elements, &b.ActionElements)
	}

	return nil
}

// ConvertToBlocks transforms Markdown into Slack Block Kit blocks.
//
// It scans the input line-by-line, producing semantically appropriate block
// types:
//
//   - Headings (# ... through ###### ...) become "header" blocks with plain_text
//   - Horizontal rules (---, ***, ___) become "divider" blocks
//   - Blockquotes (> text) become "context" blocks with mrkdwn elements;
//     images within blockquotes are split into separate image elements
//   - Standalone images (![alt](url) on their own line) become "image" blocks
//   - Standalone links ([text](url) on their own line) become "actions" blocks
//     with button elements
//   - Fenced code blocks (``` or ~~~) become "section" blocks with code fences
//   - Ordered and unordered lists become "rich_text" blocks with rich_text_list
//   - All other text is accumulated into "section" blocks with mrkdwn, split
//     at blank lines (paragraph boundaries)
//
// Rich text blocks are emitted for list content, providing structured
// representation with proper list styling. A "rich_text" block is also
// available for paragraphs, code blocks, and blockquotes through the
// RichElements field.
//
// Consecutive blockquote lines are merged into a single context block.
// Inline images within blockquote text are split into separate context
// elements. Text segments are processed through [Convert] for inline
// formatting.
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

	// List accumulation state.
	type listItem struct {
		indent  int
		content string
	}
	var listBuf []listItem
	var listStyle string // "ordered" or "bullet"

	flushList := func() {
		if len(listBuf) == 0 {
			return
		}
		items := make([]RichTextSection, 0, len(listBuf))
		for _, li := range listBuf {
			items = append(items, RichTextSection{
				Type:     "rich_text_section",
				Elements: parseInlineElements(li.content),
			})
		}

		// Group by indent level. For now treat all as indent 0.
		block := Block{
			Type: "rich_text",
			RichElements: []RichTextSection{
				{
					Type:  "rich_text_list",
					Style: listStyle,
					Items: items,
				},
			},
		}
		blocks = append(blocks, block)
		listBuf = nil
		listStyle = ""
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code fence toggle.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if inCodeBlock {
				// Closing fence -- emit the code block.
				flushList()
				blocks = appendCodeBlock(blocks, codeBuf)
				codeBuf = nil
				inCodeBlock = false
			} else {
				// Opening fence -- flush any pending content first.
				flushList()
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

		// Block quote -- context block.
		if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
			flushList()
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			// Strip the > prefix.
			var content string
			if trimmed == ">" {
				content = ""
			} else {
				idx := strings.Index(line, ">")
				content = line[idx+1:]
				content = strings.TrimPrefix(content, " ")
			}
			quoteBuf = append(quoteBuf, content)
			continue
		}

		// If we were in a blockquote and hit a non-blockquote line, flush it.
		if len(quoteBuf) > 0 {
			blocks = flushQuoteBuffer(blocks, quoteBuf)
			quoteBuf = nil
		}

		// Horizontal rule -- divider block.
		if reHorizontalRule.MatchString(line) {
			flushList()
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			blocks = append(blocks, Block{Type: "divider"})
			continue
		}

		// Heading -- header block with plain_text.
		if m := reHeading.FindStringSubmatch(trimmed); len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			flushList()
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

		// Standalone image -- image block.
		if m := reStandaloneImage.FindStringSubmatch(line); len(m) > 2 && m[2] != "" {
			flushList()
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

		// Standalone link -- actions block with button.
		if m := reStandaloneLink.FindStringSubmatch(line); len(m) > 2 && m[2] != "" {
			// Make sure it is not also an image link.
			if !reStandaloneImage.MatchString(line) {
				flushList()
				blocks = flushTextBuffer(blocks, textBuf)
				textBuf = nil
				blocks = append(blocks, Block{
					Type: "actions",
					ActionElements: []ActionElement{
						{
							Type: "button",
							Text: TextObject{
								Type: "plain_text",
								Text: m[1],
							},
							URL: m[2],
						},
					},
				})
				continue
			}
		}

		// Ordered list item.
		if m := reOrderedListItem.FindStringSubmatch(line); m != nil {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			indent := len(m[1])
			content := m[2]
			if listStyle == "bullet" {
				// Different list type -- flush previous list.
				flushList()
			}
			listStyle = "ordered"
			listBuf = append(listBuf, listItem{indent: indent, content: content})
			continue
		}

		// Unordered list item.
		if m := reUnorderedListItem.FindStringSubmatch(line); m != nil {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			indent := len(m[1])
			content := m[2]
			if listStyle == "ordered" {
				// Different list type -- flush previous list.
				flushList()
			}
			listStyle = "bullet"
			listBuf = append(listBuf, listItem{indent: indent, content: content})
			continue
		}

		// If we had a list going and hit a non-list line, flush it.
		if len(listBuf) > 0 {
			flushList()
		}

		// Blank line -- paragraph boundary.
		if trimmed == "" {
			blocks = flushTextBuffer(blocks, textBuf)
			textBuf = nil
			continue
		}

		// Regular text -- accumulate.
		textBuf = append(textBuf, line)
	}

	// Flush remaining content.
	if inCodeBlock {
		// Unclosed code fence -- emit what we have.
		blocks = appendCodeBlock(blocks, codeBuf)
	} else {
		flushList()
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
// Convert for inline formatting, and appends a context block. If the quote
// text contains image references, they are split into separate image elements
// in the context block. Returns blocks unchanged if lines is empty.
func flushQuoteBuffer(blocks []Block, lines []string) []Block {
	if len(lines) == 0 {
		return blocks
	}
	raw := strings.Join(lines, "\n")

	// Check if the raw text contains any image references.
	if !reImageLink.MatchString(raw) {
		text := Convert(raw)
		return append(blocks, Block{
			Type: "context",
			Elements: []TextObject{
				{Type: "mrkdwn", Text: text},
			},
		})
	}

	// Split the text around image references.
	elements := splitQuoteWithImages(raw)
	return append(blocks, Block{
		Type:     "context",
		Elements: elements,
	})
}

// splitQuoteWithImages splits blockquote text around image references,
// producing alternating mrkdwn text and image elements.
func splitQuoteWithImages(raw string) []TextObject {
	var elements []TextObject

	matches := reImageLink.FindAllStringSubmatchIndex(raw, -1)
	prev := 0
	for _, loc := range matches {
		// Text before this image.
		before := raw[prev:loc[0]]
		before = strings.TrimSpace(before)
		if before != "" {
			converted := Convert(before)
			elements = append(elements, TextObject{Type: "mrkdwn", Text: converted})
		}

		// Extract alt text and URL from the match.
		alt := raw[loc[2]:loc[3]]
		url := raw[loc[4]:loc[5]]
		if alt == "" {
			alt = " "
		}
		elements = append(elements, TextObject{
			Type:     "image",
			ImageURL: url,
			AltText:  alt,
		})
		prev = loc[1]
	}

	// Text after the last image.
	after := raw[prev:]
	after = strings.TrimSpace(after)
	if after != "" {
		converted := Convert(after)
		elements = append(elements, TextObject{Type: "mrkdwn", Text: converted})
	}

	return elements
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
	return stripHeadingMarkers(content)
}

// parseInlineElements parses a markdown string into a slice of RichTextElement
// values, handling bold, italic, strikethrough, code, links, and image links.
//
// The parser scans the input for recognized markdown patterns and produces
// structured elements with appropriate style flags. Unrecognized text is
// emitted as plain text elements.
func parseInlineElements(text string) []RichTextElement {
	if text == "" {
		return []RichTextElement{{Type: "text", Text: ""}}
	}

	// Collect all non-overlapping spans from inline patterns.
	var spans []span

	// Inline code: `code`
	for _, loc := range reInlineCode.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "text",
				Text: text[loc[2]:loc[3]],
				Style: &RichTextStyle{
					Code: true,
				},
			},
		})
	}

	// Image links: ![alt](url) -- must come before regular links.
	for _, loc := range reImageLink.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "link",
				URL:  text[loc[4]:loc[5]],
				Text: text[loc[2]:loc[3]],
			},
		})
	}

	// Links: [text](url)
	for _, loc := range reLink.FindAllStringSubmatchIndex(text, -1) {
		// Skip if this overlaps with an image link (preceded by !).
		if loc[0] > 0 && text[loc[0]-1] == '!' {
			continue
		}
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "link",
				URL:  text[loc[4]:loc[5]],
				Text: text[loc[2]:loc[3]],
			},
		})
	}

	// Bold: **text**
	for _, loc := range reInlineBold.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "text",
				Text: text[loc[2]:loc[3]],
				Style: &RichTextStyle{
					Bold: true,
				},
			},
		})
	}

	// Bold underscores: __text__
	for _, loc := range reInlineBoldUnder.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "text",
				Text: text[loc[2]:loc[3]],
				Style: &RichTextStyle{
					Bold: true,
				},
			},
		})
	}

	// Italic: _text_ (with word boundary context)
	for _, loc := range reInlineItalic.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "text",
				Text: text[loc[2]:loc[3]],
				Style: &RichTextStyle{
					Italic: true,
				},
			},
		})
	}

	// Strikethrough: ~~text~~
	for _, loc := range reInlineStrike.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, span{
			start: loc[0],
			end:   loc[1],
			elem: RichTextElement{
				Type: "text",
				Text: text[loc[2]:loc[3]],
				Style: &RichTextStyle{
					Strikethrough: true,
				},
			},
		})
	}

	// Remove overlapping spans, keeping the first (highest priority) match.
	spans = removeOverlappingSpans(spans)

	// Sort spans by start position.
	sortSpans(spans)

	// Build elements list by filling gaps with plain text.
	var elements []RichTextElement
	pos := 0
	for _, s := range spans {
		if s.start > pos {
			elements = append(elements, RichTextElement{
				Type: "text",
				Text: text[pos:s.start],
			})
		}
		elements = append(elements, s.elem)
		pos = s.end
	}
	if pos < len(text) {
		elements = append(elements, RichTextElement{
			Type: "text",
			Text: text[pos:],
		})
	}

	if len(elements) == 0 {
		return []RichTextElement{{Type: "text", Text: text}}
	}

	return elements
}

// removeOverlappingSpans removes spans that overlap with earlier (higher priority) spans.
func removeOverlappingSpans(spans []span) []span {
	if len(spans) <= 1 {
		return spans
	}

	// Sort by start position first, then by priority (earlier in the list = higher priority).
	sortSpans(spans)

	var result []span
	for _, s := range spans {
		overlaps := false
		for _, r := range result {
			if s.start < r.end && s.end > r.start {
				overlaps = true
				break
			}
		}
		if !overlaps {
			result = append(result, s)
		}
	}
	return result
}

// span is a helper type for parseInlineElements representing a matched region.
type span struct {
	start, end int
	elem       RichTextElement
}

// sortSpans sorts spans by start position.
func sortSpans(spans []span) {
	slices.SortFunc(spans, func(a, b span) int {
		return a.start - b.start
	})
}
