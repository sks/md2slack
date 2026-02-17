package md2slack

import (
	"encoding/json"
	"testing"

	"github.com/slack-go/slack"
)

// blockJSON is a test helper that marshals blocks to indented JSON for readable diffs.
func blockJSON(t *testing.T, blocks []slack.Block) string {
	t.Helper()
	data, err := json.MarshalIndent(blocks, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(data)
}

func TestConvert_Basic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, blocks []slack.Block)
	}{
		{
			name:  "empty input returns nil",
			input: "",
			check: func(t *testing.T, blocks []slack.Block) {
				if blocks != nil {
					t.Errorf("expected nil, got %d blocks", len(blocks))
				}
			},
		},
		{
			name:  "plain text becomes rich_text section",
			input: "Hello world",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				rt, ok := blocks[0].(*slack.RichTextBlock)
				if !ok {
					t.Fatalf("expected RichTextBlock, got %T", blocks[0])
				}
				if len(rt.Elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(rt.Elements))
				}
				sec, ok := rt.Elements[0].(*slack.RichTextSection)
				if !ok {
					t.Fatalf("expected RichTextSection, got %T", rt.Elements[0])
				}
				// Goldmark may split text into multiple text nodes.
				// Concatenate all text to verify content.
				var allText string
				for _, elem := range sec.Elements {
					te, ok := elem.(*slack.RichTextSectionTextElement)
					if !ok {
						t.Fatalf("expected RichTextSectionTextElement, got %T", elem)
					}
					allText += te.Text
				}
				if allText != "Hello world" {
					t.Errorf("expected %q, got %q", "Hello world", allText)
				}
			},
		},
		{
			name:  "heading becomes header block",
			input: "## Hello",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				h, ok := blocks[0].(*slack.HeaderBlock)
				if !ok {
					t.Fatalf("expected HeaderBlock, got %T", blocks[0])
				}
				if h.Text.Text != "Hello" {
					t.Errorf("expected %q, got %q", "Hello", h.Text.Text)
				}
			},
		},
		{
			name:  "divider from thematic break",
			input: "---",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				_, ok := blocks[0].(*slack.DividerBlock)
				if !ok {
					t.Fatalf("expected DividerBlock, got %T", blocks[0])
				}
			},
		},
		{
			name:  "standalone image becomes ImageBlock",
			input: "![logo](https://example.com/logo.png)",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				img, ok := blocks[0].(*slack.ImageBlock)
				if !ok {
					t.Fatalf("expected ImageBlock, got %T", blocks[0])
				}
				if img.ImageURL != "https://example.com/logo.png" {
					t.Errorf("expected URL %q, got %q", "https://example.com/logo.png", img.ImageURL)
				}
				if img.AltText != "logo" {
					t.Errorf("expected alt %q, got %q", "logo", img.AltText)
				}
			},
		},
		{
			name:  "standalone link becomes ActionBlock",
			input: "[Click here](https://example.com)",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				ab, ok := blocks[0].(*slack.ActionBlock)
				if !ok {
					t.Fatalf("expected ActionBlock, got %T", blocks[0])
				}
				if ab.Elements == nil || len(ab.Elements.ElementSet) == 0 {
					t.Fatal("expected at least 1 action element")
				}
			},
		},
		{
			name:  "fenced code block becomes preformatted",
			input: "```go\nfmt.Println(\"hello\")\n```",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				rt, ok := blocks[0].(*slack.RichTextBlock)
				if !ok {
					t.Fatalf("expected RichTextBlock, got %T", blocks[0])
				}
				if len(rt.Elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(rt.Elements))
				}
				_, ok = rt.Elements[0].(*slack.RichTextPreformatted)
				if !ok {
					t.Fatalf("expected RichTextPreformatted, got %T", rt.Elements[0])
				}
			},
		},
		{
			name:  "unordered list becomes RichTextBlock with list",
			input: "- Apple\n- Banana\n- Cherry",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				rt, ok := blocks[0].(*slack.RichTextBlock)
				if !ok {
					t.Fatalf("expected RichTextBlock, got %T", blocks[0])
				}
				if len(rt.Elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(rt.Elements))
				}
				list, ok := rt.Elements[0].(*slack.RichTextList)
				if !ok {
					t.Fatalf("expected RichTextList, got %T", rt.Elements[0])
				}
				if list.Style != slack.RTEListBullet {
					t.Errorf("expected bullet style, got %q", list.Style)
				}
				if len(list.Elements) != 3 {
					t.Errorf("expected 3 list items, got %d", len(list.Elements))
				}
			},
		},
		{
			name:  "ordered list becomes RichTextBlock with ordered list",
			input: "1. First\n2. Second",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				rt, ok := blocks[0].(*slack.RichTextBlock)
				if !ok {
					t.Fatalf("expected RichTextBlock, got %T", blocks[0])
				}
				list, ok := rt.Elements[0].(*slack.RichTextList)
				if !ok {
					t.Fatalf("expected RichTextList, got %T", rt.Elements[0])
				}
				if list.Style != slack.RTEListOrdered {
					t.Errorf("expected ordered style, got %q", list.Style)
				}
			},
		},
		{
			name:  "blockquote becomes rich_text with quote",
			input: "> This is a quote",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				rt, ok := blocks[0].(*slack.RichTextBlock)
				if !ok {
					t.Fatalf("expected RichTextBlock, got %T", blocks[0])
				}
				if len(rt.Elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(rt.Elements))
				}
				_, ok = rt.Elements[0].(*slack.RichTextQuote)
				if !ok {
					t.Fatalf("expected RichTextQuote, got %T", rt.Elements[0])
				}
			},
		},
		{
			name:  "table becomes section with code fence",
			input: "| Name | Age |\n|------|-----|\n| Alice | 30 |",
			check: func(t *testing.T, blocks []slack.Block) {
				if len(blocks) != 1 {
					t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
				}
				sec, ok := blocks[0].(*slack.SectionBlock)
				if !ok {
					t.Fatalf("expected SectionBlock, got %T", blocks[0])
				}
				if sec.Text == nil {
					t.Fatal("expected text, got nil")
				}
				if sec.Text.Type != slack.MarkdownType {
					t.Errorf("expected mrkdwn, got %q", sec.Text.Type)
				}
				// Should contain code fence.
				if !containsStr(sec.Text.Text, "```") {
					t.Errorf("expected code fence in table text, got: %q", sec.Text.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert error: %v", err)
			}
			tt.check(t, blocks)
		})
	}
}

func TestConvert_InlineFormatting(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		checkFirst func(t *testing.T, elem slack.RichTextSectionElement)
	}{
		{
			name:  "bold text",
			input: "**bold**",
			checkFirst: func(t *testing.T, elem slack.RichTextSectionElement) {
				te, ok := elem.(*slack.RichTextSectionTextElement)
				if !ok {
					t.Fatalf("expected text element, got %T", elem)
				}
				if te.Text != "bold" {
					t.Errorf("expected %q, got %q", "bold", te.Text)
				}
				if te.Style == nil || !te.Style.Bold {
					t.Error("expected bold style")
				}
			},
		},
		{
			name:  "italic text",
			input: "*italic*",
			checkFirst: func(t *testing.T, elem slack.RichTextSectionElement) {
				te, ok := elem.(*slack.RichTextSectionTextElement)
				if !ok {
					t.Fatalf("expected text element, got %T", elem)
				}
				if te.Text != "italic" {
					t.Errorf("expected %q, got %q", "italic", te.Text)
				}
				if te.Style == nil || !te.Style.Italic {
					t.Error("expected italic style")
				}
			},
		},
		{
			name:  "strikethrough text",
			input: "~~struck~~",
			checkFirst: func(t *testing.T, elem slack.RichTextSectionElement) {
				te, ok := elem.(*slack.RichTextSectionTextElement)
				if !ok {
					t.Fatalf("expected text element, got %T", elem)
				}
				if te.Text != "struck" {
					t.Errorf("expected %q, got %q", "struck", te.Text)
				}
				if te.Style == nil || !te.Style.Strike {
					t.Error("expected strikethrough style")
				}
			},
		},
		{
			name:  "inline code",
			input: "`code`",
			checkFirst: func(t *testing.T, elem slack.RichTextSectionElement) {
				te, ok := elem.(*slack.RichTextSectionTextElement)
				if !ok {
					t.Fatalf("expected text element, got %T", elem)
				}
				if te.Text != "code" {
					t.Errorf("expected %q, got %q", "code", te.Text)
				}
				if te.Style == nil || !te.Style.Code {
					t.Error("expected code style")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert error: %v", err)
			}
			if len(blocks) < 1 {
				t.Fatalf("expected at least 1 block, got %d", len(blocks))
			}
			rt, ok := blocks[0].(*slack.RichTextBlock)
			if !ok {
				t.Fatalf("expected RichTextBlock, got %T", blocks[0])
			}
			sec, ok := rt.Elements[0].(*slack.RichTextSection)
			if !ok {
				t.Fatalf("expected RichTextSection, got %T", rt.Elements[0])
			}
			if len(sec.Elements) < 1 {
				t.Fatalf("expected at least 1 element, got %d", len(sec.Elements))
			}
			tt.checkFirst(t, sec.Elements[0])
		})
	}
}

func TestConvert_Link(t *testing.T) {
	blocks, err := Convert("See [Google](https://google.com) for more")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
	}
	rt, ok := blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", blocks[0])
	}
	sec, ok := rt.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", rt.Elements[0])
	}

	// Should have: "See " (text), link, " for more" (text)
	if len(sec.Elements) < 3 {
		t.Fatalf("expected at least 3 elements, got %d: %s", len(sec.Elements), blockJSON(t, blocks))
	}

	link, ok := sec.Elements[1].(*slack.RichTextSectionLinkElement)
	if !ok {
		t.Fatalf("expected RichTextSectionLinkElement at index 1, got %T", sec.Elements[1])
	}
	if link.URL != "https://google.com" {
		t.Errorf("expected URL %q, got %q", "https://google.com", link.URL)
	}
	if link.Text != "Google" {
		t.Errorf("expected text %q, got %q", "Google", link.Text)
	}
}

func TestConvert_BoldItalic(t *testing.T) {
	blocks, err := Convert("***bold italic***")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	rt, ok := blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", blocks[0])
	}
	sec, ok := rt.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", rt.Elements[0])
	}
	if len(sec.Elements) < 1 {
		t.Fatalf("expected at least 1 element")
	}
	te, ok := sec.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", sec.Elements[0])
	}
	if te.Text != "bold italic" {
		t.Errorf("expected %q, got %q", "bold italic", te.Text)
	}
	if te.Style == nil || !te.Style.Bold || !te.Style.Italic {
		t.Errorf("expected bold+italic style, got %+v", te.Style)
	}
}

func TestConvert_NestedList(t *testing.T) {
	input := "- Parent\n  - Child A\n  - Child B\n- Another parent"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
	}
	rt, ok := blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", blocks[0])
	}
	// Should have a list with nested items.
	if len(rt.Elements) < 1 {
		t.Fatal("expected at least 1 element in rich text block")
	}
	// The top-level list
	list, ok := rt.Elements[0].(*slack.RichTextList)
	if !ok {
		t.Fatalf("expected RichTextList, got %T", rt.Elements[0])
	}
	if list.Style != slack.RTEListBullet {
		t.Errorf("expected bullet style, got %q", list.Style)
	}
}

func TestConvert_ComplexDocument(t *testing.T) {
	input := "# Welcome\n\nHello **world**.\n\n---\n\n![banner](https://example.com/banner.png)\n\n```\ncode here\n```\n\nGoodbye."
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) < 5 {
		t.Fatalf("expected at least 5 blocks, got %d: %s", len(blocks), blockJSON(t, blocks))
	}

	// Block 0: header
	if _, ok := blocks[0].(*slack.HeaderBlock); !ok {
		t.Errorf("block[0]: expected HeaderBlock, got %T", blocks[0])
	}
	// Block 1: rich_text (paragraph)
	if _, ok := blocks[1].(*slack.RichTextBlock); !ok {
		t.Errorf("block[1]: expected RichTextBlock, got %T", blocks[1])
	}
	// Block 2: divider
	if _, ok := blocks[2].(*slack.DividerBlock); !ok {
		t.Errorf("block[2]: expected DividerBlock, got %T", blocks[2])
	}
	// Block 3: image
	if _, ok := blocks[3].(*slack.ImageBlock); !ok {
		t.Errorf("block[3]: expected ImageBlock, got %T", blocks[3])
	}
	// Block 4: rich_text preformatted
	if _, ok := blocks[4].(*slack.RichTextBlock); !ok {
		t.Errorf("block[4]: expected RichTextBlock, got %T", blocks[4])
	}
}

func TestConvert_Table(t *testing.T) {
	input := "| Name | Age |\n|------|-----|\n| Alice | 30 |"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
	}
	sec, ok := blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock, got %T", blocks[0])
	}
	text := sec.Text.Text
	if !containsStr(text, "```") {
		t.Errorf("expected code fence, got: %q", text)
	}
	if !containsStr(text, "Name") {
		t.Errorf("expected 'Name' in table, got: %q", text)
	}
	if !containsStr(text, "Alice") {
		t.Errorf("expected 'Alice' in table, got: %q", text)
	}
}

func TestConvert_TableAlignment(t *testing.T) {
	input := "| Name | Score |\n|------|------:|\n| Alice | 100 |"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	sec := blocks[0].(*slack.SectionBlock)
	text := sec.Text.Text
	// Right-aligned "100" should have leading spaces.
	if !containsStr(text, "100") {
		t.Errorf("expected '100' in table, got: %q", text)
	}
}

func TestChunkBlocks(t *testing.T) {
	tests := []struct {
		name    string
		nBlocks int
		max     int
		want    int // number of chunks
	}{
		{"empty", 0, 50, 0},
		{"under limit", 10, 50, 1},
		{"exact limit", 50, 50, 1},
		{"over limit", 51, 50, 2},
		{"way over", 150, 50, 3},
		{"default max", 60, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := make([]slack.Block, tt.nBlocks)
			for i := range blocks {
				blocks[i] = slack.NewDividerBlock()
			}
			chunks := ChunkBlocks(blocks, tt.max)
			if len(chunks) != tt.want {
				t.Errorf("expected %d chunks, got %d", tt.want, len(chunks))
			}
		})
	}
}

func TestConvert_JSONRoundTrip(t *testing.T) {
	input := "## Hello\n\n**bold** and *italic*\n\n- item 1\n- item 2"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	data, err := json.Marshal(blocks)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// Just verify it doesn't error — the slack-go types handle their own serialization.
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestConvert_HeadingWithLink(t *testing.T) {
	// Headings with links should fall back to section block with bold mrkdwn.
	blocks, err := Convert("## Click [here](https://example.com)")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %s", len(blocks), blockJSON(t, blocks))
	}
	// Should fall back to section block since heading contains a link.
	sec, ok := blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock (fallback for heading with link), got %T", blocks[0])
	}
	if sec.Text.Text != "*Click here*" {
		t.Errorf("expected bold mrkdwn fallback, got %q", sec.Text.Text)
	}
}

func TestConvert_MultiParagraph(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph."
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d: %s", len(blocks), blockJSON(t, blocks))
	}
}

func TestConvert_TaskCheckbox(t *testing.T) {
	input := "- [x] Done\n- [ ] Todo"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if len(blocks) < 1 {
		t.Fatalf("expected at least 1 block, got %d", len(blocks))
	}
	// Should contain checkbox emojis.
	data, _ := json.Marshal(blocks)
	text := string(data)
	if !containsStr(text, "☑") {
		t.Error("expected checked checkbox emoji")
	}
	if !containsStr(text, "☐") {
		t.Error("expected unchecked checkbox emoji")
	}
}

// FuzzConvert verifies that Convert never panics on arbitrary input.
func FuzzConvert(f *testing.F) {
	f.Add("")
	f.Add("Hello world")
	f.Add("## Heading\n**bold** and [link](https://example.com)")
	f.Add("```\ncode\n```")
	f.Add("![img](https://img.com/pic.png)")
	f.Add("~~~\ncode\n~~~")
	f.Add("> block quote with **bold** & stuff")
	f.Add("| A | B |\n|---|---|\n| 1 | 2 |")
	f.Add("- [x] done\n- [ ] todo")
	f.Add("***bold italic***")
	f.Add("~~strikethrough~~")

	f.Fuzz(func(t *testing.T, input string) {
		_, err := Convert(input)
		if err != nil {
			t.Errorf("Convert panicked or errored on %q: %v", input, err)
		}
	})
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || s != "" && findStr(s, sub))
}

func findStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
