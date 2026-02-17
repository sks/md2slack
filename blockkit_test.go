package md2slack

import "testing"

func TestConvertToBlocks(t *testing.T) {
	input := "## Hello\n\nThis is **bold** and a [link](https://example.com)."

	blocks := ConvertToBlocks(input)

	if blocks == nil {
		t.Fatal("ConvertToBlocks returned nil")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	block := blocks[0]
	if block.Type != "section" {
		t.Errorf("expected block type %q, got %q", "section", block.Type)
	}
	if block.Text == nil {
		t.Fatal("expected block.Text to be non-nil")
	}
	if block.Text.Type != "mrkdwn" {
		t.Errorf("expected text type %q, got %q", "mrkdwn", block.Text.Type)
	}

	expected := Convert(input)
	if block.Text.Text != expected {
		t.Errorf("block text does not match Convert output\n  got:  %q\n  want: %q", block.Text.Text, expected)
	}
}

func TestConvertToBlocks_Empty(t *testing.T) {
	blocks := ConvertToBlocks("")

	if blocks == nil {
		t.Fatal("ConvertToBlocks returned nil for empty input")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Text.Text != "" {
		t.Errorf("expected empty text, got %q", blocks[0].Text.Text)
	}
}
