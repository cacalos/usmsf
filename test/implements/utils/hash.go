package utils

import "hash/fnv"

// Hash32 returns 32bit hash code of string
func Hash32(text string) uint32 {
	alg := fnv.New32a()
	alg.Write([]byte(text))
	return alg.Sum32()
}

func Hash32ForMap(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
