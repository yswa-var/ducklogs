package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func Download(ctx context.Context, query Query) error {
	if err := query.Validate(); err != nil {
		return err
	}

	end := time.Now()
	start := end.Add(-query.Lookback)

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(query.Region))
	if err != nil {
		return err
	}

	client := cloudwatchlogs.NewFromConfig(awsCfg)

	store, err := OpenDuckDB(ctx, query.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	var nextToken *string
	var lineNo int64

	for {
		resp, err := client.FilterLogEvents(ctx, BuildFilterLogEventsInput(query, start, end, nextToken))
		if err != nil {
			return err
		}

		logEvents := make([]LogEvent, 0, len(resp.Events))
		for _, event := range resp.Events {
			ts := time.UnixMilli(aws.ToInt64(event.Timestamp)).Format(time.RFC3339)
			stream := aws.ToString(event.LogStreamName)
			msg := aws.ToString(event.Message)

			lineNo++
			rawLine := fmt.Sprintf("[%s] [%s] %s", ts, stream, msg)
			logEvent, err := ParseLogLine(lineNo, rawLine)
			if err != nil {
				return err
			}

			logEvents = append(logEvents, logEvent)
		}

		if err := store.InsertBatch(ctx, logEvents); err != nil {
			return err
		}

		if resp.NextToken == nil || aws.ToString(resp.NextToken) == aws.ToString(nextToken) {
			break
		}

		nextToken = resp.NextToken
	}

	return nil
}

func BuildFilterLogEventsInput(query Query, start, end time.Time, nextToken *string) *cloudwatchlogs.FilterLogEventsInput {
	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(query.LogGroup),
		StartTime:    aws.Int64(start.UnixMilli()),
		EndTime:      aws.Int64(end.UnixMilli()),
		NextToken:    nextToken,
	}

	if filterPattern := BuildFilterPattern(query.Search); filterPattern != nil {
		input.FilterPattern = filterPattern
	}

	return input
}
