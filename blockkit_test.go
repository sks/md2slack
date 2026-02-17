package md2slack

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertToBlocks(t *testing.T) {
	tests := []struct {
		name string
		input string
		want []Block
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
				{Type: "image", ImageURL: "https://example.com/logo.png", AltText: "logo"},
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
				{Type: "image", ImageURL: "https://example.com/banner.png", AltText: "banner"},
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

// formatBlocks is a test helper that produces a readable representation of blocks.
func formatBlocks(blocks []Block) string {
	data, _ := json.MarshalIndent(blocks, "", "  ")
	return string(data)
}
