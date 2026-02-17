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
			expected: "*Summary of **changes***",
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
		"```\ncode block\n```",
		"- list item",
		"~struck~",
		"> quote",
		"plain text",
		"No special chars here",
	}

	for _, input := range inputs {
		first := Convert(input)
		second := Convert(first)
		if first != second {
			t.Errorf("Not idempotent for input %q:\n  first:  %q\n  second: %q", input, first, second)
		}
	}
}
