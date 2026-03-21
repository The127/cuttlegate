package domain

import "hash/fnv"

// Bucket returns a stable bucket index in [0, 99] for the given flag key and user key.
// The same (flagKey, userKey) pair always produces the same value.
// A rollout of n% enables the flag for users where Bucket(flagKey, userKey) < n.
//
// Algorithm: FNV-1a 32-bit hash of flagKey + "\x00" + userKey, modulo 100.
// The null byte separator prevents collision between ("ab", "c") and ("a", "bc").
func Bucket(flagKey, userKey string) int {
	h := fnv.New32a()
	h.Write([]byte(flagKey))
	h.Write([]byte{0x00})
	h.Write([]byte(userKey))
	return int(h.Sum32() % 100)
}
