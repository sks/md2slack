package md2slack

import (
	"encoding/json"
	"testing"
)

func TestConvertToBlocks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantTType string
		wantText  string
	}{
		{
			name:      "markdown with heading bold and link",
			input:     "## Hello\n\nThis is **bold** and a [link](https://example.com).",
			wantType:  "section",
			wantTType: "mrkdwn",
			wantText:  "*Hello*\n\nThis is *bold* and a <https://example.com|link>.",
		},
		{
			name:      "empty input",
			input:     "",
			wantType:  "section",
			wantTType: "mrkdwn",
			wantText:  "",
		},
		{
			name:      "plain text",
			input:     "just plain text",
			wantType:  "section",
			wantTType: "mrkdwn",
			wantText:  "just plain text",
		},
		{
			name:      "text with entities",
			input:     "Tom & Jerry",
			wantType:  "section",
			wantTType: "mrkdwn",
			wantText:  "Tom &amp; Jerry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := ConvertToBlocks(tt.input)
			if blocks == nil {
				t.Fatal("ConvertToBlocks returned nil")
			}
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}
			block := blocks[0]
			if block.Type != tt.wantType {
				t.Errorf("block type = %q, want %q", block.Type, tt.wantType)
			}
			if block.Text == nil {
				t.Fatal("block.Text is nil")
			}
			if block.Text.Type != tt.wantTType {
				t.Errorf("text type = %q, want %q", block.Text.Type, tt.wantTType)
			}
			if block.Text.Text != tt.wantText {
				t.Errorf("text = %q, want %q", block.Text.Text, tt.wantText)
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
