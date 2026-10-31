//go:build !api_only

package public

import _ "embed"

const APIOnly = false

//go:embed defaultTheme/dist.tar.zst
var embeddedDistArchive []byte
