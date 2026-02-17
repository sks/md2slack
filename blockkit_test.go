package md2slack

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertToBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Block
	}{
		{
			name:  "empty input",
			input: "",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: ""}},
			},
		},
		{
			name:  "plain text",
			input: "just plain text",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "just plain text"}},
			},
		},
		{
			name:  "text with entities",
			input: "Tom & Jerry",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Tom &amp; Jerry"}},
			},
		},
		{
			name:  "heading becomes header block",
			input: "## Hello",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Hello"}},
			},
		},
		{
			name:  "heading with bold stripped",
			input: "## **Important** Title",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Important Title"}},
			},
		},
		{
			name:  "heading h1",
			input: "# Top Level",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Top Level"}},
			},
		},
		{
			name:  "horizontal rule dashes",
			input: "---",
			want: []Block{
				{Type: "divider"},
			},
		},
		{
			name:  "horizontal rule asterisks",
			input: "***",
			want: []Block{
				{Type: "divider"},
			},
		},
		{
			name:  "horizontal rule underscores",
			input: "___",
			want: []Block{
				{Type: "divider"},
			},
		},
		{
			name:  "horizontal rule spaced dashes",
			input: "- - -",
			want: []Block{
				{Type: "divider"},
			},
		},
		{
			name:  "standalone image",
			input: "![logo](https://example.com/logo.png)",
			want: []Block{
				{Type: "image", ImageURL: "https://example.com/logo.png", AltText: "logo", Title: &TextObject{Type: "plain_text", Text: "logo"}},
			},
		},
		{
			name:  "standalone image empty alt gets fallback",
			input: "![](https://example.com/pic.png)",
			want: []Block{
				{Type: "image", ImageURL: "https://example.com/pic.png", AltText: " "},
			},
		},
		{
			name:  "inline image stays as mrkdwn link",
			input: "Check this ![icon](https://example.com/icon.png) out",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Check this <https://example.com/icon.png|icon> out"}},
			},
		},
		{
			name:  "code block becomes section with fences",
			input: "```go\nfmt.Println(\"hi\")\n```",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "```\nfmt.Println(\"hi\")\n```"}},
			},
		},
		{
			name:  "empty code block",
			input: "```\n```",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "```\n```"}},
			},
		},
		{
			name:  "multiple paragraphs become multiple sections",
			input: "First paragraph.\n\nSecond paragraph.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "First paragraph."}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Second paragraph."}},
			},
		},
		{
			name:  "consecutive blank lines produce one boundary",
			input: "First.\n\n\n\nSecond.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "First."}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Second."}},
			},
		},
		{
			name:  "heading followed by paragraph",
			input: "## Status\n\nAll systems **operational**.",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Status"}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "All systems *operational*."}},
			},
		},
		{
			name:  "complex document with all block types",
			input: "# Welcome\n\nHello **world**.\n\n---\n\n![banner](https://example.com/banner.png)\n\n```\ncode here\n```\n\nGoodbye.",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Welcome"}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Hello *world*."}},
				{Type: "divider"},
				{Type: "image", ImageURL: "https://example.com/banner.png", AltText: "banner", Title: &TextObject{Type: "plain_text", Text: "banner"}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "```\ncode here\n```"}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Goodbye."}},
			},
		},
		{
			name:  "multi-line paragraph stays as one section",
			input: "Line one.\nLine two.\nLine three.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Line one.\nLine two.\nLine three."}},
			},
		},
		{
			name:  "heading with link",
			input: "## Click [here](https://example.com)",
			want: []Block{
				{Type: "header", Text: &TextObject{Type: "plain_text", Text: "Click [here](https://example.com)"}},
			},
		},
		{
			name:  "blockquote becomes context block",
			input: "> This is a quote",
			want: []Block{
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "This is a quote"}}},
			},
		},
		{
			name:  "multi-line blockquote becomes single context block",
			input: "> Line one\n> Line two",
			want: []Block{
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "Line one\nLine two"}}},
			},
		},
		{
			name:  "blockquote with inline formatting",
			input: "> This is **bold** and [link](https://example.com)",
			want: []Block{
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "This is *bold* and <https://example.com|link>"}}},
			},
		},
		{
			name:  "text then blockquote then text",
			input: "Before.\n\n> A quote\n\nAfter.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Before."}},
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "A quote"}}},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "After."}},
			},
		},
		{
			name:  "empty blockquote line",
			input: "> First\n>\n> Second",
			want: []Block{
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "First\n\nSecond"}}},
			},
		},

		// Feature 2: Ordered lists become rich_text blocks.
		{
			name:  "ordered list becomes rich_text block",
			input: "1. First\n2. Second\n3. Third",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "ordered",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "First"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Second"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Third"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "ordered list with paren style",
			input: "1) Alpha\n2) Beta",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "ordered",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Alpha"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Beta"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "unordered list dash becomes rich_text block",
			input: "- Apple\n- Banana\n- Cherry",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "bullet",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Apple"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Banana"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Cherry"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "unordered list asterisk becomes rich_text block",
			input: "* One\n* Two",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "bullet",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "One"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Two"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "unordered list plus becomes rich_text block",
			input: "+ X\n+ Y",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "bullet",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "X"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Y"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "list with bold content produces styled elements",
			input: "- **Important** item\n- Normal item",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "bullet",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{
									{Type: "text", Text: "Important", Style: &RichTextStyle{Bold: true}},
									{Type: "text", Text: " item"},
								}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Normal item"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "ordered then unordered lists produce separate blocks",
			input: "1. First\n2. Second\n\n- Alpha\n- Beta",
			want: []Block{
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "ordered",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "First"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Second"}}},
							},
						},
					},
				},
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "bullet",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Alpha"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Beta"}}},
							},
						},
					},
				},
			},
		},
		{
			name:  "text then list then text",
			input: "Before.\n\n1. Item one\n2. Item two\n\nAfter.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Before."}},
				{
					Type: "rich_text",
					RichElements: []RichTextSection{
						{
							Type:  "rich_text_list",
							Style: "ordered",
							Items: []RichTextSection{
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Item one"}}},
								{Type: "rich_text_section", Elements: []RichTextElement{{Type: "text", Text: "Item two"}}},
							},
						},
					},
				},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "After."}},
			},
		},

		// Feature 3: Standalone link becomes actions block.
		{
			name:  "standalone link becomes actions block",
			input: "[Click here](https://example.com)",
			want: []Block{
				{
					Type: "actions",
					ActionElements: []ActionElement{
						{
							Type: "button",
							Text: TextObject{Type: "plain_text", Text: "Click here"},
							URL:  "https://example.com",
						},
					},
				},
			},
		},
		{
			name:  "standalone link with surrounding whitespace",
			input: "  [Visit](https://example.com/page)  ",
			want: []Block{
				{
					Type: "actions",
					ActionElements: []ActionElement{
						{
							Type: "button",
							Text: TextObject{Type: "plain_text", Text: "Visit"},
							URL:  "https://example.com/page",
						},
					},
				},
			},
		},
		{
			name:  "link in text stays as section",
			input: "See [here](https://example.com) for details",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "See <https://example.com|here> for details"}},
			},
		},
		{
			name:  "standalone link between paragraphs",
			input: "Before.\n\n[Click](https://example.com)\n\nAfter.",
			want: []Block{
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "Before."}},
				{
					Type: "actions",
					ActionElements: []ActionElement{
						{
							Type: "button",
							Text: TextObject{Type: "plain_text", Text: "Click"},
							URL:  "https://example.com",
						},
					},
				},
				{Type: "section", Text: &TextObject{Type: "mrkdwn", Text: "After."}},
			},
		},

		// Feature 4: Multi-element context blocks with images.
		{
			name:  "blockquote with image produces multi-element context",
			input: "> Check this ![icon](https://example.com/icon.png) out",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "mrkdwn", Text: "Check this"},
						{Type: "image", ImageURL: "https://example.com/icon.png", AltText: "icon"},
						{Type: "mrkdwn", Text: "out"},
					},
				},
			},
		},
		{
			name:  "blockquote with image at start",
			input: "> ![logo](https://example.com/logo.png) Company name",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "image", ImageURL: "https://example.com/logo.png", AltText: "logo"},
						{Type: "mrkdwn", Text: "Company name"},
					},
				},
			},
		},
		{
			name:  "blockquote with image at end",
			input: "> Status: ![ok](https://example.com/ok.png)",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "mrkdwn", Text: "Status:"},
						{Type: "image", ImageURL: "https://example.com/ok.png", AltText: "ok"},
					},
				},
			},
		},
		{
			name:  "blockquote with only image",
			input: "> ![pic](https://example.com/pic.png)",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "image", ImageURL: "https://example.com/pic.png", AltText: "pic"},
					},
				},
			},
		},
		{
			name:  "blockquote with image empty alt gets fallback",
			input: "> ![](https://example.com/icon.png) text",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "image", ImageURL: "https://example.com/icon.png", AltText: " "},
						{Type: "mrkdwn", Text: "text"},
					},
				},
			},
		},
		{
			name:  "blockquote without image stays as single mrkdwn element",
			input: "> Just plain text",
			want: []Block{
				{Type: "context", Elements: []TextObject{{Type: "mrkdwn", Text: "Just plain text"}}},
			},
		},
		{
			name:  "blockquote with multiple images",
			input: "> ![a](https://example.com/a.png) and ![b](https://example.com/b.png)",
			want: []Block{
				{
					Type: "context",
					Elements: []TextObject{
						{Type: "image", ImageURL: "https://example.com/a.png", AltText: "a"},
						{Type: "mrkdwn", Text: "and"},
						{Type: "image", ImageURL: "https://example.com/b.png", AltText: "b"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertToBlocks(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToBlocks(%q):\ngot:  %+v\nwant: %+v", tt.input, formatBlocks(got), formatBlocks(tt.want))
			}
		})
	}
}

func TestParseInlineElements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []RichTextElement
	}{
		{
			name:  "empty string",
			input: "",
			want:  []RichTextElement{{Type: "text", Text: ""}},
		},
		{
			name:  "plain text",
			input: "Hello world",
			want:  []RichTextElement{{Type: "text", Text: "Hello world"}},
		},
		{
			name:  "bold text",
			input: "Hello **world**",
			want: []RichTextElement{
				{Type: "text", Text: "Hello "},
				{Type: "text", Text: "world", Style: &RichTextStyle{Bold: true}},
			},
		},
		{
			name:  "bold with underscores",
			input: "Hello __world__",
			want: []RichTextElement{
				{Type: "text", Text: "Hello "},
				{Type: "text", Text: "world", Style: &RichTextStyle{Bold: true}},
			},
		},
		{
			name:  "strikethrough text",
			input: "Hello ~~world~~",
			want: []RichTextElement{
				{Type: "text", Text: "Hello "},
				{Type: "text", Text: "world", Style: &RichTextStyle{Strikethrough: true}},
			},
		},
		{
			name:  "inline code",
			input: "Use `fmt.Println`",
			want: []RichTextElement{
				{Type: "text", Text: "Use "},
				{Type: "text", Text: "fmt.Println", Style: &RichTextStyle{Code: true}},
			},
		},
		{
			name:  "link",
			input: "See [Google](https://google.com) for more",
			want: []RichTextElement{
				{Type: "text", Text: "See "},
				{Type: "link", URL: "https://google.com", Text: "Google"},
				{Type: "text", Text: " for more"},
			},
		},
		{
			name:  "image link",
			input: "Check ![logo](https://img.com/logo.png) here",
			want: []RichTextElement{
				{Type: "text", Text: "Check "},
				{Type: "link", URL: "https://img.com/logo.png", Text: "logo"},
				{Type: "text", Text: " here"},
			},
		},
		{
			name:  "multiple bold sections",
			input: "**first** and **second**",
			want: []RichTextElement{
				{Type: "text", Text: "first", Style: &RichTextStyle{Bold: true}},
				{Type: "text", Text: " and "},
				{Type: "text", Text: "second", Style: &RichTextStyle{Bold: true}},
			},
		},
		{
			name:  "code takes priority over bold",
			input: "`**not bold**`",
			want: []RichTextElement{
				{Type: "text", Text: "**not bold**", Style: &RichTextStyle{Code: true}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInlineElements(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				gotJSON, _ := json.MarshalIndent(got, "", "  ")
				wantJSON, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("parseInlineElements(%q):\ngot:  %s\nwant: %s", tt.input, gotJSON, wantJSON)
			}
		})
	}
}

func TestBlock_JSONRoundTrip(t *testing.T) {
	original := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: "Hello *world*",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded []Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 block, got %d", len(decoded))
	}
	if decoded[0].Type != "section" {
		t.Errorf("type = %q, want %q", decoded[0].Type, "section")
	}
	if decoded[0].Text == nil {
		t.Fatal("Text is nil after round-trip")
	}
	if decoded[0].Text.Type != "mrkdwn" {
		t.Errorf("text type = %q, want %q", decoded[0].Text.Type, "mrkdwn")
	}
	if decoded[0].Text.Text != "Hello *world*" {
		t.Errorf("text = %q, want %q", decoded[0].Text.Text, "Hello *world*")
	}
}

func TestBlock_JSONOmitsNilText(t *testing.T) {
	block := Block{Type: "divider"}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"divider"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestBlock_JSONImageBlock(t *testing.T) {
	block := Block{
		Type:     "image",
		ImageURL: "https://example.com/img.png",
		AltText:  "example",
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"image","image_url":"https://example.com/img.png","alt_text":"example"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestBlock_JSONImageBlockWithTitle(t *testing.T) {
	block := Block{
		Type:     "image",
		ImageURL: "https://example.com/img.png",
		AltText:  "example",
		Title:    &TextObject{Type: "plain_text", Text: "Example Image"},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"image","image_url":"https://example.com/img.png","alt_text":"example","title":{"type":"plain_text","text":"Example Image"}}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestBlock_JSONHeaderBlock(t *testing.T) {
	block := Block{
		Type: "header",
		Text: &TextObject{Type: "plain_text", Text: "Hello"},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"header","text":{"type":"plain_text","text":"Hello"}}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestBlock_JSONContextBlock(t *testing.T) {
	block := Block{
		Type: "context",
		Elements: []TextObject{
			{Type: "mrkdwn", Text: "quoted text"},
		},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"context","elements":[{"type":"mrkdwn","text":"quoted text"}]}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestBlock_JSONRichTextBlock(t *testing.T) {
	block := Block{
		Type: "rich_text",
		RichElements: []RichTextSection{
			{
				Type:  "rich_text_list",
				Style: "ordered",
				Items: []RichTextSection{
					{
						Type: "rich_text_section",
						Elements: []RichTextElement{
							{Type: "text", Text: "First"},
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Verify it round-trips correctly.
	var decoded Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Type != "rich_text" {
		t.Errorf("type = %q, want %q", decoded.Type, "rich_text")
	}
	if len(decoded.RichElements) != 1 {
		t.Fatalf("rich_elements length = %d, want 1", len(decoded.RichElements))
	}
	if decoded.RichElements[0].Type != "rich_text_list" {
		t.Errorf("rich_element type = %q, want %q", decoded.RichElements[0].Type, "rich_text_list")
	}
	if decoded.RichElements[0].Style != "ordered" {
		t.Errorf("list style = %q, want %q", decoded.RichElements[0].Style, "ordered")
	}
}

func TestBlock_JSONActionsBlock(t *testing.T) {
	block := Block{
		Type: "actions",
		ActionElements: []ActionElement{
			{
				Type: "button",
				Text: TextObject{Type: "plain_text", Text: "Click"},
				URL:  "https://example.com",
			},
		},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Verify it round-trips correctly.
	var decoded Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Type != "actions" {
		t.Errorf("type = %q, want %q", decoded.Type, "actions")
	}
	if len(decoded.ActionElements) != 1 {
		t.Fatalf("action_elements length = %d, want 1", len(decoded.ActionElements))
	}
	if decoded.ActionElements[0].Type != "button" {
		t.Errorf("element type = %q, want %q", decoded.ActionElements[0].Type, "button")
	}
	if decoded.ActionElements[0].URL != "https://example.com" {
		t.Errorf("url = %q, want %q", decoded.ActionElements[0].URL, "https://example.com")
	}
}

func TestBlock_JSONContextBlockWithImage(t *testing.T) {
	block := Block{
		Type: "context",
		Elements: []TextObject{
			{Type: "mrkdwn", Text: "Status:"},
			{Type: "image", ImageURL: "https://example.com/ok.png", AltText: "ok"},
		},
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Elements) != 2 {
		t.Fatalf("elements length = %d, want 2", len(decoded.Elements))
	}
	if decoded.Elements[0].Type != "mrkdwn" {
		t.Errorf("first element type = %q, want %q", decoded.Elements[0].Type, "mrkdwn")
	}
	if decoded.Elements[1].Type != "image" {
		t.Errorf("second element type = %q, want %q", decoded.Elements[1].Type, "image")
	}
	if decoded.Elements[1].ImageURL != "https://example.com/ok.png" {
		t.Errorf("image_url = %q, want %q", decoded.Elements[1].ImageURL, "https://example.com/ok.png")
	}
}

func TestRichTextElement_JSONStyle(t *testing.T) {
	elem := RichTextElement{
		Type:  "text",
		Text:  "bold",
		Style: &RichTextStyle{Bold: true},
	}
	data, err := json.Marshal(elem)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"text","text":"bold","style":{"bold":true}}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestRichTextElement_JSONNoStyle(t *testing.T) {
	elem := RichTextElement{
		Type: "text",
		Text: "plain",
	}
	data, err := json.Marshal(elem)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"text","text":"plain"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

func TestRichTextElement_JSONLink(t *testing.T) {
	elem := RichTextElement{
		Type: "link",
		URL:  "https://example.com",
		Text: "Example",
	}
	data, err := json.Marshal(elem)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	expected := `{"type":"link","text":"Example","url":"https://example.com"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", data, expected)
	}
}

// formatBlocks is a test helper that produces a readable representation of blocks.
func formatBlocks(blocks []Block) string {
	data, _ := json.MarshalIndent(blocks, "", "  ")
	return string(data)
}
