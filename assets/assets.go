// +build dev

package assets

import (
	"net/http"
)

// Assets contains the project's assets.
var Assets http.FileSystem = http.Dir("../frontend/dist")
