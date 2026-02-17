package md2slack

import (
	"strings"
	"unicode/utf8"

	"github.com/slack-go/slack"
	east "github.com/yuin/goldmark/extension/ast"
)

// tableState accumulates table data during AST walking.
type tableState struct {
	alignments []east.Alignment
	headerRow  []string
	dataRows   [][]string
	currentRow []string
	cellBuf    strings.Builder
	inHeader   bool
}

// handleTable processes a GFM Table node.
func (ctx *renderContext) handleTable(n *east.Table, entering bool) {
	if entering {
		ctx.inTable = true
		ctx.tableState = &tableState{
			alignments: n.Alignments,
		}
	} else {
		ctx.inTable = false
		if ctx.tableState == nil {
			return
		}

		text := ctx.tableState.render()
		ctx.emitBlock(slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			nil, nil,
		))
		ctx.tableState = nil
	}
}

// handleTableHeader processes a GFM TableHeader node.
func (ctx *renderContext) handleTableHeader(_ *east.TableHeader, entering bool) {
	if ctx.tableState == nil {
		return
	}
	if entering {
		ctx.tableState.inHeader = true
		ctx.tableState.currentRow = nil
	} else {
		ctx.tableState.headerRow = ctx.tableState.currentRow
		ctx.tableState.currentRow = nil
		ctx.tableState.inHeader = false
	}
}

// handleTableRow processes a GFM TableRow node.
func (ctx *renderContext) handleTableRow(_ *east.TableRow, entering bool) {
	if ctx.tableState == nil {
		return
	}
	if entering {
		ctx.tableState.currentRow = nil
	} else {
		ctx.tableState.dataRows = append(ctx.tableState.dataRows, ctx.tableState.currentRow)
		ctx.tableState.currentRow = nil
	}
}

// handleTableCell processes a GFM TableCell node.
func (ctx *renderContext) handleTableCell(_ *east.TableCell, entering bool) {
	if ctx.tableState == nil {
		return
	}
	if entering {
		ctx.tableState.cellBuf.Reset()
	} else {
		ctx.tableState.currentRow = append(ctx.tableState.currentRow, ctx.tableState.cellBuf.String())
		ctx.tableState.cellBuf.Reset()
	}
}

// render builds a code-fenced monospace table from the accumulated data.
func (ts *tableState) render() string {
	// Collect all rows: header + data.
	var allRows [][]string
	if len(ts.headerRow) > 0 {
		allRows = append(allRows, ts.headerRow)
	}
	allRows = append(allRows, ts.dataRows...)

	if len(allRows) == 0 {
		return "```\n```"
	}

	// Determine column count.
	numCols := 0
	for _, row := range allRows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return "```\n```"
	}

	// Normalize row lengths.
	for i := range allRows {
		for len(allRows[i]) < numCols {
			allRows[i] = append(allRows[i], "")
		}
	}

	// Compute column widths.
	widths := make([]int, numCols)
	for _, row := range allRows {
		for j, cell := range row {
			w := utf8.RuneCountInString(cell)
			if w > widths[j] {
				widths[j] = w
			}
		}
	}
	// Ensure minimum width of 3 for separator dashes.
	for j := range widths {
		if widths[j] < 3 {
			widths[j] = 3
		}
	}

	// Extract alignments.
	aligns := make([]east.Alignment, numCols)
	for i := 0; i < numCols && i < len(ts.alignments); i++ {
		aligns[i] = ts.alignments[i]
	}

	// Build output.
	var out strings.Builder
	out.WriteString("```\n")

	for i, row := range allRows {
		var line strings.Builder
		for j, cell := range row {
			if j > 0 {
				line.WriteString(" | ")
			}
			line.WriteString(padCellAligned(cell, widths[j], aligns[j]))
		}
		out.WriteString(strings.TrimRight(line.String(), " "))
		out.WriteByte('\n')

		// Insert separator after header row.
		if i == 0 && len(ts.headerRow) > 0 {
			var sep strings.Builder
			for j := 0; j < numCols; j++ {
				if j > 0 {
					sep.WriteString(" | ")
				}
				sep.WriteString(strings.Repeat("-", widths[j]))
			}
			out.WriteString(strings.TrimRight(sep.String(), " "))
			out.WriteByte('\n')
		}
	}

	result := strings.TrimRight(out.String(), "\n")
	result += "\n```"
	return result
}

// padCellAligned pads a cell to the given width using the alignment.
func padCellAligned(cell string, width int, align east.Alignment) string {
	n := utf8.RuneCountInString(cell)
	if n >= width {
		return cell
	}
	gap := width - n
	switch align {
	case east.AlignRight:
		return strings.Repeat(" ", gap) + cell
	case east.AlignCenter:
		left := gap / 2
		right := gap - left
		return strings.Repeat(" ", left) + cell + strings.Repeat(" ", right)
	default: // AlignLeft, AlignNone
		return cell + strings.Repeat(" ", gap)
	}
}
