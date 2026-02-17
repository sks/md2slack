package md2slack

import (
	"strings"

	"github.com/slack-go/slack"
)

const (
	// maxSectionTextLen is the Slack API limit for section block text.
	maxSectionTextLen = 3000

	// maxFieldsPerSection is the Slack API limit for section block fields.
	maxFieldsPerSection = 10
)

// splitSectionText splits text longer than max chars at the nearest newline
// boundary, returning multiple section blocks.
func splitSectionText(text string, max int) []*slack.SectionBlock {
	if len(text) <= max {
		return []*slack.SectionBlock{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
				nil, nil,
			),
		}
	}

	var blocks []*slack.SectionBlock
	for len(text) > 0 {
		if len(text) <= max {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
				nil, nil,
			))
			break
		}

		// Find nearest newline before the limit.
		cutoff := max
		idx := strings.LastIndex(text[:cutoff], "\n")
		if idx > 0 {
			cutoff = idx + 1
		}

		chunk := text[:cutoff]
		text = text[cutoff:]

		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, strings.TrimRight(chunk, "\n"), false, false),
			nil, nil,
		))
	}
	return blocks
}

// splitFields splits a slice of field text objects into groups of at most max,
// each group becoming the fields of a section block.
func splitFields(fields []*slack.TextBlockObject, max int) [][]*slack.TextBlockObject {
	if len(fields) <= max {
		return [][]*slack.TextBlockObject{fields}
	}

	var groups [][]*slack.TextBlockObject
	for len(fields) > 0 {
		end := max
		if end > len(fields) {
			end = len(fields)
		}
		groups = append(groups, fields[:end])
		fields = fields[end:]
	}
	return groups
}
