package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/navidemad/md2slack"
)

func main() {
	var input []byte
	var err error

	if len(os.Args) > 1 {
		input, err = os.ReadFile(os.Args[1])
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	blocks, err := md2slack.Convert(string(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting: %v\n", err)
		os.Exit(1)
	}

	chunks := md2slack.ChunkBlocks(blocks, 50)

	for i, chunk := range chunks {
		if len(chunks) > 1 {
			fmt.Fprintf(os.Stderr, "--- Message %d/%d ---\n", i+1, len(chunks))
		}
		out, err := json.MarshalIndent(chunk, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	}
}
