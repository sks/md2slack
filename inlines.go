package md2slack

import (
	"github.com/slack-go/slack"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// handleText processes an ast.Text node.
func (ctx *renderContext) handleText(n *ast.Text, entering bool) {
	if !entering {
		return
	}
	text := string(n.Value(ctx.source))

	if ctx.inHeading {
		ctx.headingBuf += text
		if n.SoftLineBreak() {
			ctx.headingBuf += " "
		}
		return
	}

	// Inside a table cell, write to the cell buffer (plain text, no formatting).
	if ctx.inTable && ctx.tableState != nil {
		ctx.tableState.cellBuf.WriteString(text)
		return
	}

	// Inside a link, accumulate text for the link display text.
	if ctx.inLink {
		ctx.linkTextBuf += text
		return
	}

	ctx.addText(text)

	if n.SoftLineBreak() {
		ctx.addText("\n")
	}
	if n.HardLineBreak() {
		ctx.addText("\n")
	}
}

// handleString processes an ast.String node.
func (ctx *renderContext) handleString(n *ast.String, entering bool) {
	if !entering {
		return
	}
	text := string(n.Value)
	if ctx.inHeading {
		ctx.headingBuf += text
		return
	}
	if ctx.inTable && ctx.tableState != nil {
		ctx.tableState.cellBuf.WriteString(text)
		return
	}
	if ctx.inLink {
		ctx.linkTextBuf += text
		return
	}
	ctx.addText(text)
}

// handleEmphasis processes an ast.Emphasis node.
func (ctx *renderContext) handleEmphasis(n *ast.Emphasis, entering bool) {
	if entering {
		s := slack.RichTextSectionTextStyle{}
		switch n.Level {
		case 1:
			s.Italic = true
		case 2:
			s.Bold = true
		}
		ctx.pushStyle(s)

		if ctx.inHeading {
			// For headings we don't want to track child styles,
			// but we still need the stack for proper pop on leave.
			return
		}
	} else {
		ctx.popStyle()
	}
}

// handleCodeSpan processes an ast.CodeSpan node.
func (ctx *renderContext) handleCodeSpan(n *ast.CodeSpan, entering bool) {
	if !entering {
		return
	}

	// Collect all child text.
	var text string
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			text += string(t.Value(ctx.source))
		}
	}

	if ctx.inHeading {
		ctx.headingBuf += text
		return
	}

	if ctx.inTable && ctx.tableState != nil {
		ctx.tableState.cellBuf.WriteString(text)
		return
	}

	if ctx.inLink {
		ctx.linkTextBuf += text
		return
	}

	style := ctx.effectiveStyle()
	if style == nil {
		style = &slack.RichTextSectionTextStyle{Code: true}
	} else {
		style.Code = true
	}
	elem := slack.NewRichTextSectionTextElement(text, style)
	ctx.inlineElements = append(ctx.inlineElements, elem)
}

// handleLink processes an ast.Link node.
func (ctx *renderContext) handleLink(n *ast.Link, entering bool) {
	if entering {
		ctx.inLink = true
		ctx.linkURL = string(n.Destination)
		ctx.linkTextBuf = ""
		if ctx.inHeading {
			ctx.headingHasUnsupported = true
		}
	} else {
		if ctx.inHeading {
			// Link text already accumulated in headingBuf via handleText.
		} else if ctx.inTable && ctx.tableState != nil {
			// Link text already written to cellBuf via handleText.
		} else {
			ctx.addLink(ctx.linkURL, ctx.linkTextBuf)
		}
		ctx.inLink = false
		ctx.linkURL = ""
		ctx.linkTextBuf = ""
	}
}

// handleImage processes an ast.Image node.
func (ctx *renderContext) handleImage(n *ast.Image, entering bool) {
	if entering {
		ctx.inImage = true
		ctx.imageURL = string(n.Destination)
		// Collect alt text from children.
		var alt string
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				alt += string(t.Value(ctx.source))
			}
		}
		ctx.imageAlt = alt

		if ctx.inHeading {
			ctx.headingHasUnsupported = true
			ctx.headingBuf += ctx.imageAlt
		}

		// In tables, write alt text to cell buffer.
		if ctx.inTable && ctx.tableState != nil {
			ctx.tableState.cellBuf.WriteString(alt)
		}
	} else {
		ctx.inImage = false
		ctx.imageURL = ""
		ctx.imageAlt = ""
	}
}

// handleAutoLink processes an ast.AutoLink node.
func (ctx *renderContext) handleAutoLink(n *ast.AutoLink, entering bool) {
	if !entering {
		return
	}
	url := string(n.URL(ctx.source))
	label := string(n.Label(ctx.source))
	if n.AutoLinkType == ast.AutoLinkEmail {
		url = "mailto:" + url
	}

	if ctx.inHeading {
		ctx.headingHasUnsupported = true
		ctx.headingBuf += label
		return
	}

	ctx.addLink(url, label)
}

// handleStrikethrough processes a GFM Strikethrough node.
func (ctx *renderContext) handleStrikethrough(_ *east.Strikethrough, entering bool) {
	if entering {
		ctx.pushStyle(slack.RichTextSectionTextStyle{Strike: true})
	} else {
		ctx.popStyle()
	}
}

// handleTaskCheckBox processes a GFM TaskCheckBox node.
func (ctx *renderContext) handleTaskCheckBox(n *east.TaskCheckBox, entering bool) {
	if !entering {
		return
	}
	if n.IsChecked {
		ctx.addText("☑ ")
	} else {
		ctx.addText("☐ ")
	}
}
