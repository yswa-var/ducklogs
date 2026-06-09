package logs

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestBuildFilterPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		search string
		want   *string
	}{
		{
			name:   "empty search is ignored",
			search: "   ",
			want:   nil,
		},
		{
			name:   "single word search is passed through",
			search: "ERROR",
			want:   aws.String("ERROR"),
		},
		{
			name:   "multi word search is quoted",
			search: "vendor not found",
			want:   aws.String("\"vendor not found\""),
		},
		{
			name:   "quoted pattern is preserved",
			search: "\"already quoted\"",
			want:   aws.String("\"already quoted\""),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildFilterPattern(tt.search)
			if aws.ToString(got) != aws.ToString(tt.want) {
				t.Fatalf("BuildFilterPattern(%q) = %q, want %q", tt.search, aws.ToString(got), aws.ToString(tt.want))
			}
		})
	}
}

func TestBuildFilterLogEventsInputIncludesOptionalSearch(t *testing.T) {
	t.Parallel()

	start := time.Unix(10, 0)
	end := time.Unix(20, 0)
	nextToken := aws.String("next-page")

	input := BuildFilterLogEventsInput(Query{
		LogGroup: "group",
		Search:   "critical failure",
	}, start, end, nextToken)

	if aws.ToString(input.LogGroupName) != "group" {
		t.Fatalf("unexpected log group: %q", aws.ToString(input.LogGroupName))
	}
	if aws.ToInt64(input.StartTime) != start.UnixMilli() {
		t.Fatalf("unexpected start time: %d", aws.ToInt64(input.StartTime))
	}
	if aws.ToInt64(input.EndTime) != end.UnixMilli() {
		t.Fatalf("unexpected end time: %d", aws.ToInt64(input.EndTime))
	}
	if aws.ToString(input.NextToken) != "next-page" {
		t.Fatalf("unexpected next token: %q", aws.ToString(input.NextToken))
	}
	if aws.ToString(input.FilterPattern) != "\"critical failure\"" {
		t.Fatalf("unexpected filter pattern: %q", aws.ToString(input.FilterPattern))
	}
}

func TestParseLookback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{name: "minutes shorthand", raw: "5m", want: 5 * time.Minute},
		{name: "minutes longhand", raw: "30 mins", want: 30 * time.Minute},
		{name: "hours", raw: "6h", want: 6 * time.Hour},
		{name: "days", raw: "2days", want: 48 * time.Hour},
		{name: "missing unit", raw: "30", wantErr: true},
		{name: "unsupported unit", raw: "5w", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseLookback(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseLookback(%q) expected error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLookback(%q) returned error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("ParseLookback(%q) = %s, want %s", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseLogLineWithTrackingAndAttrs(t *testing.T) {
	t.Parallel()

	rawLine := "[2026-05-12T13:39:08+05:30] [ecs/Main/0a6e937efb854bbf9a6efb18e8432782] 2026-05-12 08:09:08.284 | INFO | app | 1 | 139844682019584 | af01198314485493 | 9341456787191792 | calculate_orchestrator.langgraph_flows.bill.nodes:node_parse_and_validate:278 - [ITERATION_START] tracking_id=af01198314485493 status=INITIAL iteration_count=1 bill_currency=CAD exchange_rate=1"

	event, err := ParseLogLine(42, rawLine)
	if err != nil {
		t.Fatalf("ParseLogLine returned error: %v", err)
	}

	if event.LineNo != 42 {
		t.Fatalf("line number = %d, want 42", event.LineNo)
	}
	if event.CloudwatchTS != "2026-05-12T13:39:08+05:30" {
		t.Fatalf("cloudwatch timestamp = %q", event.CloudwatchTS)
	}
	if event.Level != "INFO" {
		t.Fatalf("level = %q", event.Level)
	}
	if event.TrackingID != "af01198314485493" {
		t.Fatalf("tracking id = %q", event.TrackingID)
	}
	if event.RealmID != "9341456787191792" {
		t.Fatalf("realm id = %q", event.RealmID)
	}
	if event.Module != "calculate_orchestrator.langgraph_flows.bill.nodes" {
		t.Fatalf("module = %q", event.Module)
	}
	if event.FunctionName != "node_parse_and_validate" {
		t.Fatalf("function name = %q", event.FunctionName)
	}
	if event.LineNumber == nil || *event.LineNumber != 278 {
		t.Fatalf("line number = %v, want 278", event.LineNumber)
	}
	if event.EventName != "ITERATION_START" {
		t.Fatalf("event name = %q", event.EventName)
	}
	if event.Status != "INITIAL" {
		t.Fatalf("status = %q", event.Status)
	}
	if event.IterationCount == nil || *event.IterationCount != 1 {
		t.Fatalf("iteration count = %v, want 1", event.IterationCount)
	}
	if event.AttrsJSON == "" || event.AttrsJSON == "{}" {
		t.Fatalf("attrs json was not populated: %q", event.AttrsJSON)
	}
}

func TestParseLogLineWithoutTrackingFields(t *testing.T) {
	t.Parallel()

	rawLine := "[2026-03-17T15:34:23+05:30] [ecs/Main/dc9f603651ca41c9af7b103a4c92e388] 2026-03-17 10:04:23.309 | INFO | app | 1 | 139869345088256 | calculate_orchestrator.langgraph_flows.bill.qbo_refs:ensure_vendor:473 - Vendor Mendoza Inc not found in QBO, using suffix search ...."

	event, err := ParseLogLine(7, rawLine)
	if err != nil {
		t.Fatalf("ParseLogLine returned error: %v", err)
	}

	if event.TrackingID != "" {
		t.Fatalf("tracking id = %q, want empty", event.TrackingID)
	}
	if event.SourceLocation != "calculate_orchestrator.langgraph_flows.bill.qbo_refs:ensure_vendor:473" {
		t.Fatalf("source location = %q", event.SourceLocation)
	}
	if event.FunctionName != "ensure_vendor" {
		t.Fatalf("function name = %q", event.FunctionName)
	}
}
