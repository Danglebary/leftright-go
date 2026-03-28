//go:build !amd64 && !arm64

package leftright

// Conservative default for unknown architectures.
const cacheLineBytes = 128
