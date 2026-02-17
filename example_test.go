package md2slack_test

import (
	"fmt"

	"github.com/navidemad/md2slack"
)

func ExampleConvert() {
	markdown := "## Hello\n\nThis is **bold** and a [link](https://example.com)."
	fmt.Println(md2slack.Convert(markdown))
	// Output:
	// *Hello*
	//
	// This is *bold* and a <https://example.com|link>.
}

func ExampleConvertToBlocks() {
	blocks := md2slack.ConvertToBlocks("Hello **world**")
	fmt.Printf("type=%s text_type=%s text=%q\n", blocks[0].Type, blocks[0].Text.Type, blocks[0].Text.Text)
	// Output:
	// type=section text_type=mrkdwn text="Hello *world*"
}
