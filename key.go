package cachez

import "strings"

// GetKey joins key segments with ":".
func GetKey(keys ...string) string {
	return joinKeyParts(keys...)
}

func joinKeyParts(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		part := strings.Trim(key, ":")
		if part == "" {
			continue
		}
		normalized = append(normalized, part)
	}

	return strings.Join(normalized, ":")
}
