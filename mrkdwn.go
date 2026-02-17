package md2slack

import (
	"regexp"
	"strings"
)

// Package-level compiled regexps for inline transforms.
var (
	reHeading       = regexp.MustCompile(`^#{1,6}\s+(.+)`)
	reBoldAsterisks = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUnders    = regexp.MustCompile(`__(.+?)__`)
	reLink          = regexp.MustCompile(`\[([^\]]+)\]\(((?:[^()\s]*(?:\([^()]*\))?)*)\)`)
	reStrikethrough = regexp.MustCompile(`~~(.+?)~~`)
	reNumberedList  = regexp.MustCompile(`^(\s*)\d+[.)]\s+`)
)

// Convert transforms standard markdown into Slack's mrkdwn format.
func Convert(input string) string {
	if input == "" {
		return ""
	}

	lines := strings.Split(input, "\n")
	var builder strings.Builder
	builder.Grow(len(input))

	inCodeBlock := false

	for i, line := range lines {
		if i > 0 {
			builder.WriteByte('\n')
		}

		// Check for code fence
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			builder.WriteString("```")
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			// Pass through unchanged inside code blocks
			builder.WriteString(line)
			continue
		}

		// Process inline content for non-code lines
		builder.WriteString(processInlineLine(line))
	}

	return builder.String()
}

// processInlineLine applies inline transforms to a single non-code-block line.
// It splits by backtick pairs so that inline code segments are left untouched.
// If there is an unclosed backtick (odd number of segments), the last segment
// is treated as plain text to avoid leaking unescaped characters to Slack.
func processInlineLine(line string) string {
	if strings.Count(line, "`") < 2 {
		// No backtick pairs — treat the entire line as plain text
		return applyInlineTransforms(line, line)
	}

	segments := strings.Split(line, "`")
	for i := range segments {
		// Even-index segments are outside backticks.
		// Also transform the last segment if there's an unclosed backtick
		// (even number of segments means odd backtick count).
		if i%2 == 0 || (i == len(segments)-1 && len(segments)%2 == 0) {
			segments[i] = applyInlineTransforms(segments[i], line)
		}
	}
	return strings.Join(segments, "`")
}

// applyInlineTransforms applies all inline markdown-to-Slack conversions
// to a text segment that is NOT inside inline code.
// fullLine is the original complete line (used for context like block quotes).
func applyInlineTransforms(seg string, fullLine string) string {
	// 1. Escape reserved Slack characters (must happen first, before inserting <url|text>)
	//    Skip leading > for block quote lines.
	seg = escapeSlackChars(seg, fullLine)

	// 2. Headings — ## Heading → *Heading*
	if m := reHeading.FindStringSubmatch(seg); m != nil {
		return "*" + strings.TrimSpace(m[1]) + "*"
	}

	// 3. Bold — **text** → *text*, __text__ → *text*
	seg = reBoldAsterisks.ReplaceAllString(seg, "*$1*")
	seg = reBoldUnders.ReplaceAllString(seg, "*$1*")

	// 4. Links — [text](url) → <url|text>
	//    Uses ReplaceAllStringFunc to escape | in URLs for Slack compatibility.
	seg = reLink.ReplaceAllStringFunc(seg, convertLink)

	// 5. Strikethrough — ~~text~~ → ~text~
	seg = reStrikethrough.ReplaceAllString(seg, "~$1~")

	// 6. Numbered lists — 1. item / 1) item → - item
	if loc := reNumberedList.FindStringIndex(seg); loc != nil {
		prefix := reNumberedList.FindStringSubmatch(seg)
		seg = prefix[1] + "- " + seg[loc[1]:]
	}

	return seg
}

// convertLink transforms a single markdown link match into Slack format,
// escaping | in the URL since Slack uses | as the delimiter in <url|text>.
func convertLink(match string) string {
	m := reLink.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	text := m[1]
	url := strings.ReplaceAll(m[2], "|", "%7C")
	return "<" + url + "|" + text + ">"
}

// escapeSlackChars escapes &, <, > for Slack.
// For block-quote lines (starting with "> "), the leading > is preserved.
func escapeSlackChars(seg string, fullLine string) string {
	isBlockQuote := strings.HasPrefix(strings.TrimSpace(fullLine), "> ")

	// Always escape & first (so we don't double-escape)
	seg = strings.ReplaceAll(seg, "&", "&amp;")
	seg = strings.ReplaceAll(seg, "<", "&lt;")

	if isBlockQuote {
		// Don't escape the leading > on block quote lines
		// The leading ">" will appear in the first non-code segment.
		// We escape all > except a leading one.
		seg = escapeGtPreservingBlockQuote(seg)
	} else {
		seg = strings.ReplaceAll(seg, ">", "&gt;")
	}

	return seg
}

// escapeGtPreservingBlockQuote escapes > characters but preserves a leading "> ".
func escapeGtPreservingBlockQuote(seg string) string {
	trimmed := strings.TrimLeft(seg, " \t")
	if strings.HasPrefix(trimmed, "> ") {
		// Find the position of the leading >
		idx := strings.Index(seg, ">")
		before := seg[:idx]
		after := seg[idx+1:]
		// Escape any remaining > in the rest
		after = strings.ReplaceAll(after, ">", "&gt;")
		return before + ">" + after
	}
	// No leading block quote marker in this segment
	return strings.ReplaceAll(seg, ">", "&gt;")
}
