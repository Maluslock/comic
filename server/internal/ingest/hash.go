package ingest

import (
	"hash/fnv"
)

// AllcppID computes a deterministic allcpp_id for an event card using
// FNV-1a hash of "name\x00city". The result is a non-negative int32.
func AllcppID(name, city string) int32 {
	h := fnv.New64a()
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write([]byte(city))
	sum := h.Sum64()
	// Fold 64-bit hash into a positive int32.
	return int32(sum & 0x7FFFFFFF)
}
