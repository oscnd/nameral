package util

import "strings"

func JoinValues(values []*string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = *v
	}
	return strings.Join(parts, " ")
}
