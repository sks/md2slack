package md2slack

import (
	"strings"
	"unicode/utf8"

	"github.com/slack-go/slack"
	"github.com/yuin/goldmark/ast"
)

// handleDocument processes the root Document node.
func (ctx *renderContext) handleDocument(_ *ast.Document, entering bool) {
	// No-op on entering. On leaving, nothing to flush since
	// all blocks should have been emitted by their own handlers.
}

// handleHeading processes an ast.Heading node.
func (ctx *renderContext) handleHeading(n *ast.Heading, entering bool) {
	if entering {
		ctx.inHeading = true
		ctx.headingLevel = n.Level
		ctx.headingBuf = ""
		ctx.headingHasUnsupported = false
		ctx.inlineElements = nil
	} else {
		ctx.inHeading = false
		text := strings.TrimSpace(ctx.headingBuf)

		// Slack header blocks: plain_text only, max 150 chars, no links/images.
		if !ctx.headingHasUnsupported && utf8.RuneCountInString(text) <= 150 {
			ctx.emitBlock(slack.NewHeaderBlock(
				slack.NewTextBlockObject(slack.PlainTextType, text, false, false),
			))
		} else {
			// Fallback: section block with bold mrkdwn.
			mrkdwn := "*" + text + "*"
			ctx.emitBlock(slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, mrkdwn, false, false),
				nil, nil,
			))
		}
		ctx.headingBuf = ""
		ctx.inlineElements = nil
	}
}

// handleParagraph processes an ast.Paragraph node.
func (ctx *renderContext) handleParagraph(n *ast.Paragraph, entering bool) {
	if entering {
		ctx.inlineElements = nil
		ctx.paragraphChildCount = 0
		ctx.isStandaloneImage = false
		ctx.isStandaloneLink = false
		ctx.paragraphImageURL = ""
		ctx.paragraphImageAlt = ""
		ctx.paragraphLinkURL = ""
		ctx.paragraphLinkText = ""

		// Count direct children to detect standalone image/link.
		count := 0
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			count++
		}
		ctx.paragraphChildCount = count

		// Check if single child is an image.
		if count == 1 {
			if img, ok := n.FirstChild().(*ast.Image); ok {
				ctx.isStandaloneImage = true
				ctx.paragraphImageURL = string(img.Destination)
				// Collect alt text from children.
				var alt string
				for c := img.FirstChild(); c != nil; c = c.NextSibling() {
					if t, ok := c.(*ast.Text); ok {
						alt += string(t.Value(ctx.source))
					}
				}
				ctx.paragraphImageAlt = alt
			}
		}

		// Check if single child is a link (not an image).
		if count == 1 {
			if link, ok := n.FirstChild().(*ast.Link); ok {
				ctx.isStandaloneLink = true
				ctx.paragraphLinkURL = string(link.Destination)
				// Collect text from children.
				var text string
				for c := link.FirstChild(); c != nil; c = c.NextSibling() {
					switch v := c.(type) {
					case *ast.Text:
						text += string(v.Value(ctx.source))
					case *ast.CodeSpan:
						for gc := v.FirstChild(); gc != nil; gc = gc.NextSibling() {
							if t, ok := gc.(*ast.Text); ok {
								text += string(t.Value(ctx.source))
							}
						}
					}
				}
				ctx.paragraphLinkText = text
			}
		}

	} else {
		// Leaving paragraph.
		if ctx.inHeading {
			return
		}

		// Standalone image → ImageBlock.
		if ctx.isStandaloneImage {
			alt := ctx.paragraphImageAlt
			if alt == "" {
				alt = " "
			}
			ctx.emitBlock(slack.NewImageBlock(
				ctx.paragraphImageURL, alt, "", nil,
			))
			ctx.inlineElements = nil
			return
		}

		// Standalone link → ActionBlock with button.
		if ctx.isStandaloneLink {
			btn := slack.NewButtonBlockElement(
				"", ctx.paragraphLinkURL,
				slack.NewTextBlockObject(slack.PlainTextType, ctx.paragraphLinkText, false, false),
			)
			btn.URL = ctx.paragraphLinkURL
			ctx.emitBlock(slack.NewActionBlock("", btn))
			ctx.inlineElements = nil
			return
		}

		// Inside a list item → don't emit a block, let the list handler deal with it.
		if ctx.inList() {
			return
		}

		// Inside a blockquote → accumulate, let the blockquote handler deal with it.
		if ctx.inBlockquote {
			if len(ctx.quoteElements) > 0 {
				// Add a newline separator between blockquote paragraphs.
				ctx.quoteElements = append(ctx.quoteElements,
					slack.NewRichTextSectionTextElement("\n", nil))
			}
			ctx.quoteElements = append(ctx.quoteElements, ctx.inlineElements...)
			ctx.inlineElements = nil
			return
		}

		// Normal paragraph → section block with mrkdwn.
		if len(ctx.inlineElements) > 0 {
			sec := slack.NewRichTextSection(ctx.inlineElements...)
			ctx.emitBlock(slack.NewRichTextBlock("", sec))
			ctx.inlineElements = nil
		}
	}
}

// handleBlockquote processes an ast.Blockquote node.
func (ctx *renderContext) handleBlockquote(_ *ast.Blockquote, entering bool) {
	if entering {
		ctx.inBlockquote = true
		ctx.quoteElements = nil
	} else {
		ctx.inBlockquote = false
		if len(ctx.quoteElements) > 0 {
			quote := &slack.RichTextQuote{
				Type:     slack.RTEQuote,
				Elements: ctx.quoteElements,
			}
			ctx.emitBlock(slack.NewRichTextBlock("", quote))
		}
		ctx.quoteElements = nil
	}
}

// handleFencedCodeBlock processes an ast.FencedCodeBlock node.
func (ctx *renderContext) handleFencedCodeBlock(n *ast.FencedCodeBlock, entering bool) {
	if !entering {
		return
	}

	// Extract all lines from the code block.
	var buf strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(ctx.source))
	}

	text := buf.String()
	// Trim trailing newline that goldmark includes.
	text = strings.TrimRight(text, "\n")

	pre := &slack.RichTextPreformatted{
		RichTextSection: slack.RichTextSection{
			Type: slack.RTESection,
			Elements: []slack.RichTextSectionElement{
				slack.NewRichTextSectionTextElement(text, nil),
			},
		},
	}
	ctx.emitBlock(slack.NewRichTextBlock("", pre))
}

// handleCodeBlock processes an ast.CodeBlock (indented code block).
func (ctx *renderContext) handleCodeBlock(n *ast.CodeBlock, entering bool) {
	if !entering {
		return
	}

	var buf strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(ctx.source))
	}

	text := strings.TrimRight(buf.String(), "\n")

	pre := &slack.RichTextPreformatted{
		RichTextSection: slack.RichTextSection{
			Type: slack.RTESection,
			Elements: []slack.RichTextSectionElement{
				slack.NewRichTextSectionTextElement(text, nil),
			},
		},
	}
	ctx.emitBlock(slack.NewRichTextBlock("", pre))
}

// handleList processes an ast.List node.
func (ctx *renderContext) handleList(n *ast.List, entering bool) {
	if entering {
		style := slack.RTEListBullet
		if n.IsOrdered() {
			style = slack.RTEListOrdered
		}
		indent := len(ctx.listStack)
		ctx.listStack = append(ctx.listStack, listFrame{
			style:  style,
			indent: indent,
		})
	} else {
		if len(ctx.listStack) == 0 {
			return
		}
		frame := ctx.listStack[len(ctx.listStack)-1]
		ctx.listStack = ctx.listStack[:len(ctx.listStack)-1]

		list := slack.NewRichTextList(frame.style, frame.indent, frame.items...)

		if len(ctx.listStack) == 0 {
			// Top-level list: emit as a RichTextBlock.
			ctx.emitBlock(slack.NewRichTextBlock("", list))
		} else {
			// Nested list: append to parent frame as an element.
			parentFrame := &ctx.listStack[len(ctx.listStack)-1]
			parentFrame.items = append(parentFrame.items, list)
		}
	}
}

// handleListItem processes an ast.ListItem node.
func (ctx *renderContext) handleListItem(_ *ast.ListItem, entering bool) {
	if entering {
		ctx.inlineElements = nil
	} else {
		if len(ctx.listStack) == 0 {
			return
		}
		sec := ctx.flushInlineToSection()
		if sec == nil {
			sec = slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement("", nil),
			)
		}

		frame := &ctx.listStack[len(ctx.listStack)-1]
		frame.items = append(frame.items, sec)
	}
}

// handleThematicBreak processes an ast.ThematicBreak node.
func (ctx *renderContext) handleThematicBreak(_ *ast.ThematicBreak, entering bool) {
	if entering {
		ctx.emitBlock(slack.NewDividerBlock())
	}
}

// handleHTMLBlock is a no-op — we skip raw HTML blocks.
func (ctx *renderContext) handleHTMLBlock(_ *ast.HTMLBlock, _ bool) {
}

// handleTextBlock processes an ast.TextBlock node.
func (ctx *renderContext) handleTextBlock(_ *ast.TextBlock, _ bool) {
}
