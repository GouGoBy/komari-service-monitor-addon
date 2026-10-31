//go:build api_only

package public

// APIOnly builds a backend without bundling or serving frontend assets.
const APIOnly = true

var embeddedDistArchive []byte
