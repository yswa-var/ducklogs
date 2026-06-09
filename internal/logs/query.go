package logs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

var lookbackPattern = regexp.MustCompile(`^(\d+)\s*([a-zA-Z]+)$`)

type Query struct {
	LogGroup string
	Region   string
	DBPath   string
	Search   string
	Lookback time.Duration
}

func (q Query) Validate() error {
	switch {
	case strings.TrimSpace(q.LogGroup) == "":
		return fmt.Errorf("log group is required")
	case strings.TrimSpace(q.Region) == "":
		return fmt.Errorf("region is required")
	case strings.TrimSpace(q.DBPath) == "":
		return fmt.Errorf("database path is required")
	case q.Lookback <= 0:
		return fmt.Errorf("time window must be greater than zero")
	default:
		return nil
	}
}

func ParseLookback(raw string) (time.Duration, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return 0, fmt.Errorf("time window is required")
	}

	matches := lookbackPattern.FindStringSubmatch(value)
	if matches == nil {
		return 0, fmt.Errorf("invalid time window %q; use values like 5m, 30m, 6h, or 2d", raw)
	}

	amount, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid time amount %q", matches[1])
	}
	if amount <= 0 {
		return 0, fmt.Errorf("time amount must be greater than zero")
	}

	unit, err := normalizeLookbackUnit(matches[2])
	if err != nil {
		return 0, err
	}

	switch unit {
	case time.Minute:
		return time.Duration(amount) * time.Minute, nil
	case time.Hour:
		return time.Duration(amount) * time.Hour, nil
	case 24 * time.Hour:
		return time.Duration(amount) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported time unit")
	}
}

func normalizeLookbackUnit(raw string) (time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "m", "min", "mins", "minute", "minutes":
		return time.Minute, nil
	case "h", "hr", "hrs", "hour", "hours":
		return time.Hour, nil
	case "d", "day", "days":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported time unit %q; use m, h, or d", raw)
	}
}

func BuildFilterPattern(search string) *string {
	search = strings.TrimSpace(search)
	if search == "" {
		return nil
	}

	// Quote plain multi-word searches so CloudWatch treats them as a phrase.
	if strings.ContainsAny(search, " \t") && !strings.Contains(search, "\"") {
		quoted := fmt.Sprintf("\"%s\"", search)
		return aws.String(quoted)
	}

	return aws.String(search)
}
