package md2slack_test

import (
	"encoding/json"
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

func ExampleConvert_codeBlock() {
	markdown := "```go\nif a < b {\n\tfmt.Println(a & b)\n}\n```"
	fmt.Println(md2slack.Convert(markdown))
	// Output:
	// ```
	// if a < b {
	// 	fmt.Println(a & b)
	// }
	// ```
}

func ExampleConvert_escaping() {
	fmt.Println(md2slack.Convert("Tom & Jerry say 1 < 2 > 0"))
	// Output:
	// Tom &amp; Jerry say 1 &lt; 2 &gt; 0
}

func ExampleConvertToBlocks() {
	blocks := md2slack.ConvertToBlocks("Hello **world**")
	fmt.Printf("type=%s text_type=%s text=%q\n", blocks[0].Type, blocks[0].Text.Type, blocks[0].Text.Text)
	// Output:
	// type=section text_type=mrkdwn text="Hello *world*"
}

func ExampleConvertToBlocks_json() {
	blocks := md2slack.ConvertToBlocks("## Status\n\nAll systems **operational**.")
	out, _ := json.MarshalIndent(blocks, "", "  ")
	fmt.Println(string(out))
	// Output:
	// [
	//   {
	//     "type": "header",
	//     "text": {
	//       "type": "plain_text",
	//       "text": "Status"
	//     }
	//   },
	//   {
	//     "type": "section",
	//     "text": {
	//       "type": "mrkdwn",
	//       "text": "All systems *operational*."
	//     }
	//   }
	// ]
}

func ExampleConvertToBlocks_richBlocks() {
	md := "# Welcome\n\nHello **world**.\n\n---\n\n![banner](https://example.com/banner.png)"
	blocks := md2slack.ConvertToBlocks(md)
	out, _ := json.MarshalIndent(blocks, "", "  ")
	fmt.Println(string(out))
	// Output:
	// [
	//   {
	//     "type": "header",
	//     "text": {
	//       "type": "plain_text",
	//       "text": "Welcome"
	//     }
	//   },
	//   {
	//     "type": "section",
	//     "text": {
	//       "type": "mrkdwn",
	//       "text": "Hello *world*."
	//     }
	//   },
	//   {
	//     "type": "divider"
	//   },
	//   {
	//     "type": "image",
	//     "image_url": "https://example.com/banner.png",
	//     "alt_text": "banner",
	//     "title": {
	//       "type": "plain_text",
	//       "text": "banner"
	//     }
	//   }
	// ]
}
