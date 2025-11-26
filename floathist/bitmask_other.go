//go:build !amd64 || !gc
// +build !amd64 !gc

package floathist

func bitmask(data *[32]uint32) uint32 { return bitmaskFallback(data) }
