package md2slack

import (
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "bold asterisks",
			input:    "This is **bold** text",
			expected: "This is *bold* text",
		},
		{
			name:     "bold underscores",
			input:    "This is __bold__ text",
			expected: "This is *bold* text",
		},
		{
			name:     "heading level 1",
			input:    "# Heading",
			expected: "*Heading*",
		},
		{
			name:     "heading level 2",
			input:    "## Heading",
			expected: "*Heading*",
		},
		{
			name:     "heading level 3",
			input:    "### Heading",
			expected: "*Heading*",
		},
		{
			name:     "link conversion",
			input:    "See [Google](https://google.com) for more",
			expected: "See <https://google.com|Google> for more",
		},
		{
			name:     "strikethrough",
			input:    "This is ~~deleted~~ text",
			expected: "This is ~deleted~ text",
		},
		{
			name:     "numbered list with dot",
			input:    "1. First item",
			expected: "- First item",
		},
		{
			name:     "numbered list with paren",
			input:    "2) Second item",
			expected: "- Second item",
		},
		{
			name:     "indented numbered list",
			input:    "   3. Third item",
			expected: "   - Third item",
		},
		{
			name:     "code block preserved",
			input:    "```go\nfmt.Println(\"hello\")\n```",
			expected: "```\nfmt.Println(\"hello\")\n```",
		},
		{
			name:     "inline code protected",
			input:    "Use `**not bold**` in code",
			expected: "Use `**not bold**` in code",
		},
		{
			name:     "HTML entity escaping ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "HTML entity escaping less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "HTML entity escaping greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "block quote leading > preserved",
			input:    "> This is a quote",
			expected: "> This is a quote",
		},
		{
			name:     "block quote with inner >",
			input:    "> value > other",
			expected: "> value &gt; other",
		},
		{
			name:     "single asterisk bold not converted",
			input:    "This is *already slack bold*",
			expected: "This is *already slack bold*",
		},
		{
			name:     "italic with underscores passes through",
			input:    "_italic_",
			expected: "_italic_",
		},
		{
			name:     "italic with underscores in sentence",
			input:    "This is _italic_ text",
			expected: "This is _italic_ text",
		},
		{
			name:     "bold and italic together",
			input:    "**bold** and _italic_",
			expected: "*bold* and _italic_",
		},
		{
			name:     "italic and bold together",
			input:    "_italic_ and **bold**",
			expected: "_italic_ and *bold*",
		},
		{
			name:     "multiple italic spans",
			input:    "_one_ and _two_ and _three_",
			expected: "_one_ and _two_ and _three_",
		},
		{
			name:     "code block content not escaped",
			input:    "```\na < b && c > d\n```",
			expected: "```\na < b && c > d\n```",
		},
		{
			name:     "multiple bold in one line",
			input:    "**first** and **second**",
			expected: "*first* and *second*",
		},
		{
			name:     "link with special chars in URL",
			input:    "[Click here](https://example.com/path?a=1&b=2)",
			expected: "<https://example.com/path?a=1&amp;b=2|Click here>",
		},
		{
			name:     "unclosed backtick still applies transforms",
			input:    "This has a ` stray backtick and **bold**",
			expected: "This has a ` stray backtick and *bold*",
		},
		{
			name:     "heading with trailing bold",
			input:    "## Summary of **changes**",
			expected: "*Summary of changes*",
		},
		{
			name:     "multiple links in one line",
			input:    "See [A](https://a.com) and [B](https://b.com)",
			expected: "See <https://a.com|A> and <https://b.com|B>",
		},
		{
			name:     "text with only whitespace",
			input:    "   ",
			expected: "   ",
		},
		{
			name:     "wikipedia style link with nested parens",
			input:    "[Go](https://en.wikipedia.org/wiki/Go_(programming_language))",
			expected: "<https://en.wikipedia.org/wiki/Go_(programming_language)|Go>",
		},
		{
			name:     "link with pipe in URL escaped",
			input:    "[Search](https://example.com/q?filter=a|b)",
			expected: "<https://example.com/q?filter=a%7Cb|Search>",
		},
		{
			name:     "image link converted to slack link",
			input:    "![alt text](https://img.com/pic.png)",
			expected: "<https://img.com/pic.png|alt text>",
		},
		{
			name:     "image link with empty alt",
			input:    "![](https://img.com/pic.png)",
			expected: "<https://img.com/pic.png>",
		},
		{
			name:     "image link with empty URL unchanged",
			input:    "![alt]()",
			expected: "![alt]()",
		},
		{
			name:     "link with empty URL unchanged",
			input:    "[text]()",
			expected: "[text]()",
		},
		{
			name:     "image link inline with text",
			input:    "See ![logo](https://img.com/logo.png) here",
			expected: "See <https://img.com/logo.png|logo> here",
		},
		{
			name:     "tilde code fence preserved",
			input:    "~~~\na < b\n~~~",
			expected: "```\na < b\n```",
		},
		{
			name:     "tilde code fence with language",
			input:    "~~~go\nfmt.Println()\n~~~",
			expected: "```\nfmt.Println()\n```",
		},
		{
			name:     "empty code block",
			input:    "```\n```",
			expected: "```\n```",
		},
		{
			name:     "heading with bold and underscores",
			input:    "## Review of __important__ items",
			expected: "*Review of important items*",
		},
		{
			name:     "heading with multiple bolds",
			input:    "## The **quick** and **bold** fox",
			expected: "*The quick and bold fox*",
		},
		{
			name:     "already escaped ampersand preserved",
			input:    "Tom &amp; Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "already escaped lt preserved",
			input:    "a &lt; b",
			expected: "a &lt; b",
		},
		{
			name:     "already escaped gt preserved",
			input:    "a &gt; b",
			expected: "a &gt; b",
		},
		{
			name:     "slack link preserved",
			input:    "<https://example.com|link>",
			expected: "<https://example.com|link>",
		},
		{
			name:     "slack link with entities preserved",
			input:    "See <https://example.com|link> &amp; more",
			expected: "See <https://example.com|link> &amp; more",
		},
		{
			name:     "unordered list asterisk",
			input:    "* First item",
			expected: "- First item",
		},
		{
			name:     "unordered list plus",
			input:    "+ Second item",
			expected: "- Second item",
		},
		{
			name:     "unordered list dash unchanged",
			input:    "- Third item",
			expected: "- Third item",
		},
		{
			name:     "indented unordered list asterisk",
			input:    "   * Nested item",
			expected: "   - Nested item",
		},
		{
			name:     "asterisk bold not list",
			input:    "*bold text*",
			expected: "*bold text*",
		},
		{
			name:     "unordered list with bold content",
			input:    "* **Important** item",
			expected: "- *Important* item",
		},

		// Reference-style links.
		{
			name:     "reference link resolved",
			input:    "See [Google][goog] for search.\n\n[goog]: https://google.com",
			expected: "See <https://google.com|Google> for search.",
		},
		{
			name:     "collapsed reference link",
			input:    "Visit [Google][] today.\n\n[Google]: https://google.com",
			expected: "Visit <https://google.com|Google> today.",
		},
		{
			name:     "reference link case insensitive",
			input:    "See [Docs][DOCS-REF].\n\n[docs-ref]: https://docs.example.com",
			expected: "See <https://docs.example.com|Docs>.",
		},
		{
			name:     "reference image resolved",
			input:    "![logo][logo-ref]\n\n[logo-ref]: https://img.com/logo.png",
			expected: "<https://img.com/logo.png|logo>",
		},
		{
			name:     "definition lines stripped",
			input:    "Text.\n\n[ref]: https://example.com",
			expected: "Text.",
		},
		{
			name:     "undefined reference left as-is",
			input:    "See [unknown][nope] link.",
			expected: "See [unknown][nope] link.",
		},
		{
			name:     "definition inside code fence ignored",
			input:    "```\n[ref]: https://example.com\n```\n\n[text][ref]",
			expected: "```\n[ref]: https://example.com\n```\n\n[text][ref]",
		},
		{
			name:     "angle-bracket URL in definition",
			input:    "[ref]: <https://example.com>\n\n[text][ref]",
			expected: "<https://example.com|text>",
		},
		{
			name:     "definition with title discarded",
			input:    "[ref]: https://example.com \"Example\"\n\n[text][ref]",
			expected: "<https://example.com|text>",
		},
		{
			name:     "first definition wins",
			input:    "[ref]: https://first.com\n[ref]: https://second.com\n\n[text][ref]",
			expected: "<https://first.com|text>",
		},

		// Tables.
		{
			name:     "simple table",
			input:    "| Name | Age |\n|------|-----|\n| Alice | 30 |",
			expected: "```\nName  | Age\n----- | ---\nAlice | 30\n```",
		},
		{
			name:     "table with bold and code cells",
			input:    "| Header | Value |\n|--------|-------|\n| **bold** | `code` |",
			expected: "```\nHeader | Value\n------ | -----\nbold   | code\n```",
		},
		{
			name:     "table between paragraphs",
			input:    "Before.\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\nAfter.",
			expected: "Before.\n\n```\nA   | B\n--- | ---\n1   | 2\n```\n\nAfter.",
		},
		{
			name:     "table at end of input",
			input:    "Text.\n\n| X |\n|---|\n| Y |",
			expected: "Text.\n\n```\nX\n---\nY\n```",
		},
		{
			name:     "table without separator row",
			input:    "| A | B |\n| 1 | 2 |\n| 3 | 4 |",
			expected: "```\nA   | B\n1   | 2\n3   | 4\n```",
		},
		{
			name:     "single column table",
			input:    "| Item |\n|------|\n| One |\n| Two |",
			expected: "```\nItem\n----\nOne\nTwo\n```",
		},
		{
			name:     "table with empty cells",
			input:    "| A | B |\n|---|---|\n|   | 2 |",
			expected: "```\nA   | B\n--- | ---\n    | 2\n```",
		},
		{
			name:     "table with right alignment",
			input:    "| Name | Score |\n|------|------:|\n| Alice | 100 |",
			expected: "```\nName  | Score\n----- | -----\nAlice |   100\n```",
		},
		{
			name:     "table with center alignment",
			input:    "| Name | Status |\n|------|:------:|\n| Test | OK |",
			expected: "```\nName | Status\n---- | ------\nTest |   OK\n```",
		},
		{
			name:     "table with link in cell",
			input:    "| Page |\n|------|\n| [Go](https://go.dev) |",
			expected: "```\nPage\n----\nGo\n```",
		},

		// Links with backticks in text.
		{
			name:     "link with backtick text",
			input:    "[`code`](https://example.com)",
			expected: "<https://example.com|code>",
		},
		{
			name:     "image link with backtick alt",
			input:    "![`alt`](https://img.com/pic.png)",
			expected: "<https://img.com/pic.png|alt>",
		},
		{
			name:     "reference link with backtick text",
			input:    "[`code`][ref]\n\n[ref]: https://example.com",
			expected: "<https://example.com|code>",
		},
		{
			name:     "link with backtick text inline",
			input:    "See [`github.com/org/repo`](https://github.com/org/repo) for details",
			expected: "See <https://github.com/org/repo|github.com/org/repo> for details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.expected {
				t.Errorf("Convert(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvert_RealWorldOutput(t *testing.T) {
	input := `## Summary

I've analyzed the codebase and found **three issues** that need fixing:

1. The ` + "`config.Parse()`" + ` function doesn't handle empty strings
2. Error handling in ` + "`main.go`" + ` is incomplete
3. Tests are missing for the ~~old~~ new parser

### Code Changes

Here's what I changed in the parser:

` + "```go" + `
func Parse(input string) (*Config, error) {
    if input == "" {
        return nil, fmt.Errorf("empty input")
    }
    // handle <special> & "edge" cases
    return parse(input)
}
` + "```" + `

For more details, see [the documentation](https://docs.example.com/parser) and [the PR](https://github.com/org/repo/pull/42).

Tom & Jerry would approve of these changes, since ` + "`x < y`" + ` is now properly validated.`

	expected := `*Summary*

I've analyzed the codebase and found *three issues* that need fixing:

- The ` + "`config.Parse()`" + ` function doesn't handle empty strings
- Error handling in ` + "`main.go`" + ` is incomplete
- Tests are missing for the ~old~ new parser

*Code Changes*

Here's what I changed in the parser:

` + "```" + `
func Parse(input string) (*Config, error) {
    if input == "" {
        return nil, fmt.Errorf("empty input")
    }
    // handle <special> & "edge" cases
    return parse(input)
}
` + "```" + `

For more details, see <https://docs.example.com/parser|the documentation> and <https://github.com/org/repo/pull/42|the PR>.

Tom &amp; Jerry would approve of these changes, since ` + "`x < y`" + ` is now properly validated.`

	got := Convert(input)
	if got != expected {
		t.Errorf("RealWorldOutput mismatch.\nGot:\n%s\n\nWant:\n%s", got, expected)
	}
}

func TestConvert_Idempotent(t *testing.T) {
	// Already-converted Slack mrkdwn should not change on a second pass.
	inputs := []string{
		"*bold text*",
		"_italic text_",
		"*bold* and _italic_",
		"```\ncode block\n```",
		"- list item",
		"~struck~",
		"> quote",
		"plain text",
		"No special chars here",
		"Tom &amp; Jerry",
		"a &lt; b &gt; c",
		"<https://example.com|link>",
		"See <https://example.com|link> &amp; more",
		"<https://example.com>",
		"> block quote with &amp; entity",
		"```\nA | B\n- | -\n1 | 2\n```",
	}

	for _, input := range inputs {
		first := Convert(input)
		second := Convert(first)
		if first != second {
			t.Errorf("Not idempotent for input %q:\n  first:  %q\n  second: %q", input, first, second)
		}
	}
}

// TestConvert_Idempotent_FromMarkdown verifies that Convert output is stable
// when fed back through Convert (markdown → mrkdwn → mrkdwn).
func TestConvert_Idempotent_FromMarkdown(t *testing.T) {
	markdownInputs := []string{
		"## Heading",
		"**bold** text",
		"_italic_ text",
		"**bold** and _italic_",
		"[link](https://example.com)",
		"![img](https://img.com/pic.png)",
		"Tom & Jerry",
		"a < b > c",
		"~~deleted~~ text",
		"1. first\n2. second",
		"* first\n* second",
		"+ first\n+ second",
		"```\ncode & <stuff>\n```",
		"> block quote with & and **bold**",
		"[`code`](https://example.com)",
		"| A | B |\n|---|---|\n| 1 | 2 |",
	}

	for _, md := range markdownInputs {
		first := Convert(md)
		second := Convert(first)
		if first != second {
			t.Errorf("Not idempotent from markdown %q:\n  first:  %q\n  second: %q", md, first, second)
		}
	}
}

func TestFormatTable(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			name:  "simple two column",
			lines: []string{"| A | B |", "|---|---|", "| 1 | 2 |"},
			want:  "A   | B\n--- | ---\n1   | 2",
		},
		{
			name:  "strips bold from cells",
			lines: []string{"| H |", "|---|", "| **bold** |"},
			want:  "H\n----\nbold",
		},
		{
			name:  "right aligned",
			lines: []string{"| Name | Num |", "|------|----:|", "| A | 10 |", "| BC | 5 |"},
			want:  "Name | Num\n---- | ---\nA    |  10\nBC   |   5",
		},
		{
			name:  "center aligned",
			lines: []string{"| X |", "|:---:|", "| Hi |"},
			want:  " X\n---\nHi",
		},
		{
			name:  "no separator row",
			lines: []string{"| A | B |", "| 1 | 2 |"},
			want:  "A   | B\n1   | 2",
		},
		{
			name:  "uneven columns padded",
			lines: []string{"| Short | LongHeader |", "|-------|------------|", "| A | B |"},
			want:  "Short | LongHeader\n----- | ----------\nA     | B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTable(tt.lines)
			if got != tt.want {
				t.Errorf("formatTable()\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// FuzzConvert verifies that Convert never panics on arbitrary input
// and that its output is idempotent.
func FuzzConvert(f *testing.F) {
	f.Add("")
	f.Add("Hello world")
	f.Add("## Heading\n**bold** and [link](https://example.com)")
	f.Add("```\ncode\n```")
	f.Add("Tom & Jerry < > &amp; &lt; &gt;")
	f.Add("![img](https://img.com/pic.png)")
	f.Add("~~~\ncode\n~~~")
	f.Add("> block quote with **bold** & stuff")
	f.Add("<https://example.com|link>")

	f.Fuzz(func(t *testing.T, input string) {
		first := Convert(input)
		second := Convert(first)
		if first != second {
			t.Errorf("Not idempotent:\n  input:  %q\n  first:  %q\n  second: %q", input, first, second)
		}
	})
}
