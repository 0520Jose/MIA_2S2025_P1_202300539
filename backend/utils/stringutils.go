package utils

import (
	"strconv"
	"strings"
)

func SplitTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	var out []string
	for _, v := range raw {
		trim := strings.TrimSpace(v)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}


func ParseID(idStr string) int {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1
	}
	return id
}