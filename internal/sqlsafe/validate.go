package sqlsafe

import (
	"fmt"
	"regexp"
	"strings"
)

var blockedKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
	"COPY", "INSTALL", "LOAD", "ATTACH", "EXPORT", "PRAGMA",
	"CALL", "DETACH",
}

func ValidateReadOnlySQL(query string) error {
	q := strings.TrimSpace(query)
	upper := strings.ToUpper(q)

	if !(strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")) {
		return fmt.Errorf("only SELECT or WITH queries are allowed")
	}

	for _, word := range blockedKeywords {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
		if pattern.MatchString(q) {
			return fmt.Errorf("blocked SQL keyword: %s", word)
		}
	}

	return nil
}
