package md2slack

// Block represents a single Slack Block Kit layout block.
//
// See https://api.slack.com/reference/block-kit/blocks for the full
// Block Kit specification.
type Block struct {
	// Type identifies the block kind (e.g. "section", "divider", "header").
	Type string `json:"type"`

	// Text holds the block's text content. Nil for block types that don't
	// carry text (e.g. "divider").
	Text *TextObject `json:"text,omitempty"`
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
// It converts the input using [Convert] and wraps the result in a single
// "section" block with mrkdwn text type. The returned slice is ready for
// use as the "blocks" field in a Slack API payload.
//
// Future versions may produce richer block structures (e.g. separate
// header, code, and list blocks).
func ConvertToBlocks(markdown string) []Block {
	mrkdwn := Convert(markdown)
	return []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: mrkdwn,
			},
		},
	}
}
