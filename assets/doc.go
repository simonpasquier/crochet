package assets

import (
	_ "github.com/shurcooL/vfsgen"          // For Go modules.
	_ "github.com/simonpasquier/modtimevfs" // For Go modules.
)

//go:generate go run -tags=dev assets_generate.go
