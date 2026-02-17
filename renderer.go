package md2slack

import (
	"strings"

	"github.com/slack-go/slack"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Convert parses a Markdown string and returns Slack Block Kit blocks.
//
// It uses goldmark with GFM extensions to parse the markdown into an AST,
// then walks the tree to build structured Slack blocks. The resulting blocks
// can be used directly with the slack-go library to post messages.
//
// Supported markdown features:
//   - Headings (# through ######) → HeaderBlock (plain_text, max 150 chars)
//   - Bold (**text** / __text__) → RichTextSectionTextStyle{Bold: true}
//   - Italic (*text* / _text_) → RichTextSectionTextStyle{Italic: true}
//   - Strikethrough (~~text~~) → RichTextSectionTextStyle{Strike: true}
//   - Inline code (`code`) → RichTextSectionTextStyle{Code: true}
//   - Links ([text](url)) → RichTextSectionLinkElement
//   - Images (![alt](url)) → ImageBlock (standalone) or link element (inline)
//   - Fenced code blocks (```) → RichTextBlock with RichTextPreformatted
//   - Blockquotes (> text) → RichTextBlock with RichTextQuote
//   - Ordered lists (1. item) → RichTextBlock with RichTextList (ordered)
//   - Unordered lists (- item) → RichTextBlock with RichTextList (bullet)
//   - Nested lists → RichTextList with indent levels
//   - Tables (GFM) → SectionBlock with code-fenced monospace
//   - Horizontal rules (---) → DividerBlock
//   - Standalone links ([text](url) alone) → ActionBlock with button
//   - Task checkboxes (- [x] item) → checkbox emoji text
//
// An empty string returns nil and no error.
func Convert(markdown string) ([]slack.Block, error) {
	if strings.TrimSpace(markdown) == "" {
		return nil, nil
	}

	source := []byte(markdown)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	doc := md.Parser().Parse(text.NewReader(source))

	ctx := &renderContext{
		source: source,
	}

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		// Block nodes.
		case *ast.Document:
			ctx.handleDocument(v, entering)
		case *ast.Heading:
			ctx.handleHeading(v, entering)
		case *ast.Paragraph:
			ctx.handleParagraph(v, entering)
		case *ast.Blockquote:
			ctx.handleBlockquote(v, entering)
		case *ast.FencedCodeBlock:
			ctx.handleFencedCodeBlock(v, entering)
			if entering {
				return ast.WalkSkipChildren, nil
			}
		case *ast.CodeBlock:
			ctx.handleCodeBlock(v, entering)
			if entering {
				return ast.WalkSkipChildren, nil
			}
		case *ast.List:
			ctx.handleList(v, entering)
		case *ast.ListItem:
			ctx.handleListItem(v, entering)
		case *ast.ThematicBreak:
			ctx.handleThematicBreak(v, entering)
		case *ast.HTMLBlock:
			ctx.handleHTMLBlock(v, entering)
		case *ast.TextBlock:
			ctx.handleTextBlock(v, entering)

		// Inline nodes.
		case *ast.Text:
			ctx.handleText(v, entering)
		case *ast.String:
			ctx.handleString(v, entering)
		case *ast.Emphasis:
			ctx.handleEmphasis(v, entering)
		case *ast.CodeSpan:
			ctx.handleCodeSpan(v, entering)
			if entering {
				return ast.WalkSkipChildren, nil
			}
		case *ast.Link:
			ctx.handleLink(v, entering)
		case *ast.Image:
			ctx.handleImage(v, entering)
			if entering {
				return ast.WalkSkipChildren, nil
			}
		case *ast.AutoLink:
			ctx.handleAutoLink(v, entering)

		// GFM extension nodes.
		case *east.Table:
			ctx.handleTable(v, entering)
		case *east.TableHeader:
			ctx.handleTableHeader(v, entering)
		case *east.TableRow:
			ctx.handleTableRow(v, entering)
		case *east.TableCell:
			ctx.handleTableCell(v, entering)
		case *east.Strikethrough:
			ctx.handleStrikethrough(v, entering)
		case *east.TaskCheckBox:
			ctx.handleTaskCheckBox(v, entering)
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	return ctx.blocks, nil
}

// ChunkBlocks splits a slice of blocks into chunks of at most maxPerMessage.
// This is useful for respecting Slack's 50-block-per-message limit.
// If maxPerMessage is <= 0, it defaults to 50.
func ChunkBlocks(blocks []slack.Block, maxPerMessage int) [][]slack.Block {
	if maxPerMessage <= 0 {
		maxPerMessage = 50
	}
	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) <= maxPerMessage {
		return [][]slack.Block{blocks}
	}

	var chunks [][]slack.Block
	for len(blocks) > 0 {
		end := maxPerMessage
		if end > len(blocks) {
			end = len(blocks)
		}
		chunks = append(chunks, blocks[:end])
		blocks = blocks[end:]
	}
	return chunks
}
