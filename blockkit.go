package md2slack

// Block represents a single Slack Block Kit block.
type Block struct {
	Type string      `json:"type"`
	Text *TextObject `json:"text,omitempty"`
}

// TextObject represents a text composition object in Block Kit.
type TextObject struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ConvertToBlocks transforms markdown into Slack Block Kit blocks.
//
// Currently it wraps the Convert output in a single section block with
// mrkdwn text type. Future versions will produce richer block structures
// (e.g. separate header, code, and list blocks).
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
