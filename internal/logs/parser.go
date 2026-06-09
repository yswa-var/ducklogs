package logs

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogEvent struct {
	LineNo         int64
	RawLine        string
	CloudwatchTS   string
	LogStream      string
	AppTS          time.Time
	Level          string
	AppName        string
	ProcessID      string
	ThreadID       string
	TrackingID     string
	RealmID        string
	SourceLocation string
	Module         string
	FunctionName   string
	LineNumber     *int
	EventName      string
	Status         string
	IterationCount *int
	Message        string
	AttrsJSON      string
}

var (
	logRe = regexp.MustCompile(
		`^\[([^\]]+)\]\s+\[([^\]]+)\]\s+` +
			`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+)\s+\|\s+` +
			`([A-Z]+)\s+\|\s+` +
			`([^|]+)\s+\|\s+` +
			`([^|]+)\s+\|\s+` +
			`([^|]+)\s+\|\s+` +
			`(?:(\S+)\s+\|\s+(\S+)\s+\|\s+)?` +
			`(.+?)\s+-\s+(.*)$`,
	)
	sourceLocationRe = regexp.MustCompile(`^(.+):([^:]+):(\d+)$`)
	attrRe           = regexp.MustCompile(`(?:^|[\s,])([A-Za-z_][A-Za-z0-9_]*)=("[^"]*"|'[^']*'|[^\s,]+)`)
	eventNameRe      = regexp.MustCompile(`^\[([A-Z][A-Z0-9_]*)\]`)
)

func ParseLogLine(lineNo int64, rawLine string) (LogEvent, error) {
	matches := logRe.FindStringSubmatch(rawLine)
	if matches == nil {
		return LogEvent{}, fmt.Errorf("line %d does not match log format", lineNo)
	}

	appTS, err := time.Parse("2006-01-02 15:04:05.999999999", matches[3])
	if err != nil {
		return LogEvent{}, fmt.Errorf("line %d has invalid app timestamp: %w", lineNo, err)
	}

	event := LogEvent{
		LineNo:         lineNo,
		RawLine:        rawLine,
		CloudwatchTS:   strings.TrimSpace(matches[1]),
		LogStream:      strings.TrimSpace(matches[2]),
		AppTS:          appTS,
		Level:          strings.TrimSpace(matches[4]),
		AppName:        strings.TrimSpace(matches[5]),
		ProcessID:      strings.TrimSpace(matches[6]),
		ThreadID:       strings.TrimSpace(matches[7]),
		TrackingID:     strings.TrimSpace(matches[8]),
		RealmID:        strings.TrimSpace(matches[9]),
		SourceLocation: strings.TrimSpace(matches[10]),
		Message:        strings.TrimSpace(matches[11]),
	}

	event.Module, event.FunctionName, event.LineNumber = splitSourceLocation(event.SourceLocation)

	attrs := parseMessageAttrs(event.Message)
	event.EventName = stringAttr(attrs, "event_name")
	if event.EventName == "" {
		event.EventName = eventNameFromMessage(event.Message)
	}
	event.Status = stringAttr(attrs, "status")
	event.IterationCount = intAttr(attrs, "iteration_count")
	if event.TrackingID == "" {
		event.TrackingID = stringAttr(attrs, "tracking_id")
	}
	if event.RealmID == "" {
		event.RealmID = stringAttr(attrs, "realm_id")
	}

	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return LogEvent{}, fmt.Errorf("line %d has invalid attrs: %w", lineNo, err)
	}
	event.AttrsJSON = string(attrsJSON)

	return event, nil
}

func splitSourceLocation(sourceLocation string) (string, string, *int) {
	matches := sourceLocationRe.FindStringSubmatch(strings.TrimSpace(sourceLocation))
	if matches == nil {
		return sourceLocation, "", nil
	}

	lineNumber, err := strconv.Atoi(matches[3])
	if err != nil {
		return sourceLocation, "", nil
	}

	return matches[1], matches[2], &lineNumber
}

func parseMessageAttrs(message string) map[string]any {
	attrs := make(map[string]any)

	for _, match := range attrRe.FindAllStringSubmatch(message, -1) {
		key := match[1]
		value := strings.Trim(match[2], `"'`)

		if intValue, err := strconv.Atoi(value); err == nil {
			attrs[key] = intValue
			continue
		}

		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			attrs[key] = floatValue
			continue
		}

		attrs[key] = value
	}

	return attrs
}

func eventNameFromMessage(message string) string {
	matches := eventNameRe.FindStringSubmatch(strings.TrimSpace(message))
	if matches == nil {
		return ""
	}
	return matches[1]
}

func stringAttr(attrs map[string]any, key string) string {
	value, ok := attrs[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func intAttr(attrs map[string]any, key string) *int {
	value, ok := attrs[key]
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case int:
		return &typed
	case float64:
		intValue := int(typed)
		if typed == float64(intValue) {
			return &intValue
		}
	}

	return nil
}
