// Program to generate the static assets.

// +build ignore

package main

import (
	"log"
	"time"

	"github.com/shurcooL/vfsgen"
	"github.com/simonpasquier/modtimevfs"

	"github.com/simonpasquier/crochet/assets"
)

func main() {
	fs := modtimevfs.New(assets.Assets, time.Unix(1, 0))
	err := vfsgen.Generate(fs, vfsgen.Options{
		PackageName:  "assets",
		BuildTags:    "!dev",
		VariableName: "Assets",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
