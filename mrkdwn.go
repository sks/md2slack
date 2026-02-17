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
	reImageLink     = regexp.MustCompile(`!\[([^\]]*)\]\(((?:[^()\s]*(?:\([^()]*\))?)*)\)`)
	reLink          = regexp.MustCompile(`\[([^\]]+)\]\(((?:[^()\s]*(?:\([^()]*\))?)*)\)`)
	reStrikethrough = regexp.MustCompile(`~~(.+?)~~`)
	reNumberedList  = regexp.MustCompile(`^(\s*)\d+[.)]\s+`)
	reSlackLink     = regexp.MustCompile(`<[^\s<>][^<>]*>`)
	reEntity        = regexp.MustCompile(`^&(amp|lt|gt);`)
	reConsecStars   = regexp.MustCompile(`\*{2,}`)
	reConsecUnders  = regexp.MustCompile(`_{2,}`)
)

// Convert transforms standard Markdown into Slack's mrkdwn text format.
//
// It processes the input line-by-line, applying the following transformations
// to text outside of code fences and inline code spans:
//
//   - Headings become bold: ## Title → *Title*
//   - Bold markers collapse: **text** or __text__ → *text*
//   - Links reformat: [text](url) → <url|text>
//   - Image links: ![alt](url) → <url|alt>
//   - Strikethrough simplifies: ~~text~~ → ~text~
//   - Numbered lists normalize: 1. item → - item
//   - Reserved characters escape: & < > → &amp; &lt; &gt;
//   - Block quotes ("> ") pass through with the leading > preserved
//
// Fenced code blocks (``` or ~~~ delimited) and inline code (`…`) are passed through
// unchanged. Convert is idempotent: already-converted mrkdwn is returned
// unmodified.
//
// An empty string returns an empty string.
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

		// Check for code fence (``` or ~~~)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
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
//
// Heading detection runs on the full line first (headings are line-level).
// If a heading is found, its content is cleaned of bold markers then the
// remaining non-code segments are transformed for links/strikethrough/escaping,
// and the result is wrapped in *...*.
func processInlineLine(line string) string {
	// Headings are line-level — detect on the escaped full line.
	escaped := escapeSlackChars(line, line)
	if m := reHeading.FindStringSubmatch(escaped); m != nil && strings.TrimSpace(m[1]) != "" {
		return processHeadingLine(m[1], line)
	}

	if strings.Count(line, "`") < 2 {
		return applyInlineTransforms(line, line)
	}

	segments := strings.Split(line, "`")
	for i := range segments {
		if i%2 == 0 || (i == len(segments)-1 && len(segments)%2 == 0) {
			segments[i] = applyInlineTransforms(segments[i], line)
		}
	}
	return strings.Join(segments, "`")
}

// processHeadingLine handles a heading line by stripping bold markers from the
// heading content, then processing the full content (respecting backtick-delimited
// code segments) for links, strikethrough, and escaping, and wrapping in *...*.
func processHeadingLine(headingContent string, fullLine string) string {
	content := strings.TrimSpace(headingContent)

	// Strip bold markers and edge stars (redundant since heading is bold).
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

	if content == "" {
		return ""
	}

	// Process remaining content for non-bold inline transforms,
	// respecting backtick-delimited code segments.
	if strings.Count(content, "`") < 2 {
		content = applyHeadingSegmentTransforms(content, fullLine)
	} else {
		segments := strings.Split(content, "`")
		for i := range segments {
			if i%2 == 0 || (i == len(segments)-1 && len(segments)%2 == 0) {
				segments[i] = applyHeadingSegmentTransforms(segments[i], fullLine)
			}
		}
		content = strings.Join(segments, "`")
	}

	// Prevent code fences in output.
	if strings.HasPrefix(strings.TrimSpace(content), "```") || strings.HasPrefix(strings.TrimSpace(content), "~~~") {
		content = " " + content
	}

	return "*" + content + "*"
}

// applyHeadingSegmentTransforms applies non-bold inline transforms to a heading
// text segment (escaping, links, strikethrough). Bold is already handled by
// the heading wrapper.
func applyHeadingSegmentTransforms(seg string, fullLine string) string {
	seg = escapeSlackChars(seg, fullLine)
	seg = replaceOutsideSlackLinks(seg, reImageLink, convertImageLink)
	seg = replaceOutsideSlackLinks(seg, reLink, convertLink)
	seg = replaceWithContext(seg, reStrikethrough, "~", "~")
	return seg
}

// applyInlineTransforms applies all inline markdown-to-Slack conversions
// to a text segment that is NOT inside inline code.
// fullLine is the original complete line (used for context like block quotes).
func applyInlineTransforms(seg string, fullLine string) string {
	// 1. Escape reserved Slack characters (must happen first, before inserting <url|text>)
	//    Skip leading > for block quote lines.
	seg = escapeSlackChars(seg, fullLine)

	// 2. Bold — **text** → *text*, __text__ → *text*
	//    Context-aware replacement to avoid creating new ** sequences at boundaries.
	//    Operate only outside Slack links to avoid mangling URLs.
	seg = replaceWithContextOutsideSlackLinks(seg, reBoldAsterisks, "*", "*")
	seg = replaceWithContextOutsideSlackLinks(seg, reBoldUnders, "*", "_")

	// 4a. Image links — ![alt](url) → <url|alt>
	//     Slack mrkdwn has no inline image syntax, so images become regular links.
	//     Must run before regular link matching to avoid leaving a stray "!".
	seg = replaceOutsideSlackLinks(seg, reImageLink, convertImageLink)

	// 4b. Links — [text](url) → <url|text>
	//     Uses ReplaceAllStringFunc to escape | in URLs for Slack compatibility.
	seg = replaceOutsideSlackLinks(seg, reLink, convertLink)

	// 5. Strikethrough — ~~text~~ → ~text~
	//    Context-aware replacement to avoid creating new ~~ sequences at boundaries.
	//    Operate only outside Slack links to avoid mangling URLs.
	seg = replaceWithContextOutsideSlackLinks(seg, reStrikethrough, "~", "~")

	// 6. Numbered lists — 1. item / 1) item → - item
	if loc := reNumberedList.FindStringIndex(seg); loc != nil {
		prefix := reNumberedList.FindStringSubmatch(seg)
		seg = prefix[1] + "- " + seg[loc[1]:]
	}

	return seg
}

// replaceWithContext performs regex replacement of double-delimiter bold/underline
// markers (**text** or __text__) into single-delimiter format (*text*), but only
// when it won't create ambiguous sequences at the match boundaries.
// wrap is the output delimiter (e.g. "*"), skip is the source delimiter (e.g. "*" or "_").
func replaceWithContext(s string, re *regexp.Regexp, wrap string, skip string) string {
	matches := re.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	prev := 0
	lastWrote := byte(0) // track the last byte written to detect seam conflicts

	for _, loc := range matches {
		start, end := loc[0], loc[1]
		content := s[loc[2]:loc[3]]

		// Skip if content starts/ends with the source or wrap delimiter,
		// since wrapping would create a double-delimiter sequence.
		if strings.HasPrefix(content, skip) || strings.HasSuffix(content, skip) ||
			strings.HasPrefix(content, wrap) || strings.HasSuffix(content, wrap) {
			b.WriteString(s[prev:end])
			if end > prev {
				lastWrote = s[end-1]
			}
			prev = end
			continue
		}

		// Check what character precedes this replacement in the output.
		gap := s[prev:start]
		var charBefore byte
		if len(gap) > 0 {
			charBefore = gap[len(gap)-1]
		} else {
			charBefore = lastWrote
		}

		// Skip if the character before or after would create a double-delimiter.
		if charBefore == wrap[0] {
			b.WriteString(s[prev:end])
			if end > prev {
				lastWrote = s[end-1]
			}
			prev = end
			continue
		}
		if end < len(s) && string(s[end]) == wrap {
			b.WriteString(s[prev:end])
			if end > prev {
				lastWrote = s[end-1]
			}
			prev = end
			continue
		}

		b.WriteString(gap)
		b.WriteString(wrap)
		b.WriteString(content)
		b.WriteString(wrap)
		lastWrote = wrap[0]
		prev = end
	}
	b.WriteString(s[prev:])
	return b.String()
}

// convertImageLink transforms a markdown image match into Slack link format.
// Slack mrkdwn has no inline image syntax, so ![alt](url) becomes <url|alt>.
// Matches with empty URLs are left unchanged.
func convertImageLink(match string) string {
	m := reImageLink.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	alt := m[1]
	url := strings.ReplaceAll(m[2], "|", "%7C")
	if url == "" {
		return match
	}
	if alt == "" {
		return "<" + url + ">"
	}
	return "<" + url + "|" + alt + ">"
}

// convertLink transforms a single markdown link match into Slack format,
// escaping | in the URL since Slack uses | as the delimiter in <url|text>.
// Matches with empty URLs are left unchanged to avoid creating new patterns
// from partially consumed bracket sequences.
func convertLink(match string) string {
	m := reLink.FindStringSubmatch(match)
	if m == nil {
		return match
	}
	text := m[1]
	url := strings.ReplaceAll(m[2], "|", "%7C")
	if url == "" {
		return match
	}
	return "<" + url + "|" + text + ">"
}

// replaceWithContextOutsideSlackLinks combines replaceWithContext and
// replaceOutsideSlackLinks: applies context-aware replacement only to portions
// of the string that are not inside Slack-format links (<...>).
func replaceWithContextOutsideSlackLinks(s string, re *regexp.Regexp, wrap string, skip string) string {
	slackLinks := reSlackLink.FindAllStringIndex(s, -1)
	if len(slackLinks) == 0 {
		return replaceWithContext(s, re, wrap, skip)
	}

	var b strings.Builder
	b.Grow(len(s))
	prev := 0
	for _, loc := range slackLinks {
		gap := s[prev:loc[0]]
		b.WriteString(replaceWithContext(gap, re, wrap, skip))
		b.WriteString(s[loc[0]:loc[1]])
		prev = loc[1]
	}
	b.WriteString(replaceWithContext(s[prev:], re, wrap, skip))
	return b.String()
}

// replaceOutsideSlackLinks applies a regex replacement only to portions of the
// string that are not inside Slack-format links (<...>).
func replaceOutsideSlackLinks(s string, re *regexp.Regexp, repl func(string) string) string {
	slackLinks := reSlackLink.FindAllStringIndex(s, -1)
	if len(slackLinks) == 0 {
		return re.ReplaceAllStringFunc(s, repl)
	}

	var b strings.Builder
	b.Grow(len(s))
	prev := 0
	for _, loc := range slackLinks {
		// Process the gap before this Slack link
		gap := s[prev:loc[0]]
		b.WriteString(re.ReplaceAllStringFunc(gap, repl))
		// Pass through the Slack link unchanged
		b.WriteString(s[loc[0]:loc[1]])
		prev = loc[1]
	}
	// Process any remaining text after the last Slack link
	b.WriteString(re.ReplaceAllStringFunc(s[prev:], repl))
	return b.String()
}

// escapeSlackChars escapes &, <, > for Slack while preserving already-escaped
// entities (&amp; &lt; &gt;) and Slack-format links (<url|text>).
// For block-quote lines (starting with "> "), the leading > is preserved.
func escapeSlackChars(seg string, fullLine string) string {
	isBlockQuote := strings.HasPrefix(strings.TrimSpace(fullLine), "> ")

	// Escape & but skip already-escaped entities (&amp; &lt; &gt;).
	seg = escapeAmpersand(seg)

	// Escape < and > but preserve Slack-format links (<...>).
	seg = escapeAngleBrackets(seg, isBlockQuote)

	return seg
}

// escapeAmpersand escapes & characters that are not already part of an entity.
func escapeAmpersand(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			if reEntity.MatchString(s[i:]) {
				// Already an entity — pass through as-is
				b.WriteByte('&')
			} else {
				b.WriteString("&amp;")
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// escapeAngleBrackets escapes < and > while preserving Slack-format links.
// If isBlockQuote is true, the leading > of a block quote is also preserved.
func escapeAngleBrackets(seg string, isBlockQuote bool) string {
	// Find all Slack-format links (<...>) to protect them.
	matches := reSlackLink.FindAllStringIndex(seg, -1)
	inSlackLink := func(pos int) bool {
		for _, m := range matches {
			if pos >= m[0] && pos < m[1] {
				return true
			}
		}
		return false
	}

	var b strings.Builder
	b.Grow(len(seg))
	blockQuoteGtHandled := false
	for i := 0; i < len(seg); i++ {
		if inSlackLink(i) {
			b.WriteByte(seg[i])
			continue
		}
		switch seg[i] {
		case '<':
			b.WriteString("&lt;")
		case '>':
			if isBlockQuote && !blockQuoteGtHandled {
				trimBefore := strings.TrimLeft(seg[:i+1], " \t")
				if strings.HasSuffix(trimBefore, "> ") || trimBefore == ">" {
					b.WriteByte('>')
					blockQuoteGtHandled = true
					continue
				}
			}
			b.WriteString("&gt;")
		default:
			b.WriteByte(seg[i])
		}
	}
	return b.String()
}
