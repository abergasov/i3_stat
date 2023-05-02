package utils

import (
	"encoding/json"
	"hash/fnv"
)

func HashInt(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func HashAny(data any) uint32 {
	res, _ := json.Marshal(data)
	return HashInt(string(res))
}
