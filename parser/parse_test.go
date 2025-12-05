package parser_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	customerrors "agent-scheduler/errors"
	"agent-scheduler/models"
	"agent-scheduler/parser"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	// Helper to create time.Time from "3PM" or "3:04PM" format string, in PST/PDT, today
	parseTime := func(s string) time.Time {
		loc, err := time.LoadLocation("America/Los_Angeles")
		if err != nil {
			panic(err)
		}
		now := time.Now().In(loc)

		layouts := []string{"3:04PM", "3PM"}
		var t time.Time
		for _, layout := range layouts {
			t, err = time.ParseInLocation(layout, s, loc)
			if err == nil {
				break
			}
		}
		if err != nil {
			panic(err)
		}
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}

	tests := map[string]struct {
		input         string
		expectedData  []models.CallData
		expectedError error
	}{
		"ValidInput_SingleLine": {
			input: `
Stanford Hospital, 300, 9:30AM, 7:30PM, 20000, 1
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "Stanford Hospital",
					AverageCallDurationSeconds: 300,
					StartTime:                  parseTime("9:30AM"),
					EndTime:                    parseTime("7:30PM"),
					Location:                   func() *time.Location { l, _ := time.LoadLocation("America/Los_Angeles"); return l }(),
					NumberOfCalls:              20000,
					Priority:                   1,
				},
			},
			expectedError: nil,
		},
		"ValidInput_MultipleLines_WithComments": {
			input: `
# This is a comment
# CustomerName, Duration, Start, End, Calls, Priority
VNS, 120, 6AM, 1PM, 40500, 1
CVS, 180, 11AM, 3PM, 50000, 3
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "VNS",
					AverageCallDurationSeconds: 120,
					StartTime:                  parseTime("6AM"),
					EndTime:                    parseTime("1PM"),
					Location:                   func() *time.Location { l, _ := time.LoadLocation("America/Los_Angeles"); return l }(),
					NumberOfCalls:              40500,
					Priority:                   1,
				},
				{
					CustomerName:               "CVS",
					AverageCallDurationSeconds: 180,
					StartTime:                  parseTime("11AM"),
					EndTime:                    parseTime("3PM"),
					Location:                   func() *time.Location { l, _ := time.LoadLocation("America/Los_Angeles"); return l }(),
					NumberOfCalls:              50000,
					Priority:                   3,
				},
			},
			expectedError: nil,
		},
		"Error_InvalidFieldCount": {
			input: `
Stanford Hospital, 300, 9AM, 7PM, 20000
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidFieldCount,
		},
		"Error_InvalidDuration": {
			input: `
Stanford Hospital, abc, 9AM, 7PM, 20000, 1
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidDuration,
		},
		"Error_InvalidStartTime": {
			input: `
Stanford Hospital, 300, 99AM, 7PM, 20000, 1
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidStartTime,
		},
		"Error_InvalidEndTime": {
			input: `
Stanford Hospital, 300, 9AM, 25PM, 20000, 1
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidEndTime,
		},
		"Error_InvalidNumberOfCalls": {
			input: `
Stanford Hospital, 300, 9AM, 7PM, xyz, 1
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidNumberOfCalls,
		},
		"Error_InvalidPriority": {
			input: `
Stanford Hospital, 300, 9AM, 7PM, 20000, p1
`,
			expectedData:  nil,
			expectedError: customerrors.ErrInvalidPriority,
		},
		"Error_StartTimeAfterEndTime": {
			input: `
Stanford Hospital, 300, 7PM, 9AM, 20000, 1
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "Stanford Hospital",
					AverageCallDurationSeconds: 300,
					StartTime:                  parseTime("7PM"),
					EndTime:                    parseTime("9AM"),
					Location:                   func() *time.Location { l, _ := time.LoadLocation("America/Los_Angeles"); return l }(),
					NumberOfCalls:              20000,
					Priority:                   1,
				},
			},
			expectedError: nil,
		},
		"ValidInput_EasternTime": {
			input: `
#CustomerName, Duration, StartTimeET, EndTimeET, Calls, Priority
VNS, 120, 6AM, 1PM, 40500, 1
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "VNS",
					AverageCallDurationSeconds: 120,
					StartTime: func() time.Time {
						loc, _ := time.LoadLocation("America/New_York")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, loc)
					}(),
					EndTime: func() time.Time {
						loc, _ := time.LoadLocation("America/New_York")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, loc)
					}(),
					Location:      func() *time.Location { l, _ := time.LoadLocation("America/New_York"); return l }(),
					NumberOfCalls: 40500,
					Priority:      1,
				},
			},
			expectedError: nil,
		},
		"ValidInput_MultipleTimezones": {
			input: `
#CustomerName, Duration, StartTimePT, EndTimePT, Calls, Priority
West Coast, 120, 9AM, 5PM, 10000, 1
#CustomerName, Duration, StartTimeET, EndTimeET, Calls, Priority
East Coast, 180, 9AM, 5PM, 15000, 2
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "West Coast",
					AverageCallDurationSeconds: 120,
					StartTime: func() time.Time {
						loc, _ := time.LoadLocation("America/Los_Angeles")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
					}(),
					EndTime: func() time.Time {
						loc, _ := time.LoadLocation("America/Los_Angeles")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, loc)
					}(),
					Location:      func() *time.Location { l, _ := time.LoadLocation("America/Los_Angeles"); return l }(),
					NumberOfCalls: 10000,
					Priority:      1,
				},
				{
					CustomerName:               "East Coast",
					AverageCallDurationSeconds: 180,
					StartTime: func() time.Time {
						loc, _ := time.LoadLocation("America/New_York")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
					}(),
					EndTime: func() time.Time {
						loc, _ := time.LoadLocation("America/New_York")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, loc)
					}(),
					Location:      func() *time.Location { l, _ := time.LoadLocation("America/New_York"); return l }(),
					NumberOfCalls: 15000,
					Priority:      2,
				},
			},
			expectedError: nil,
		},
		"ValidInput_InternationalTimezones": {
			input: `
#CustomerName, Duration, StartTimeAsia/Tokyo, EndTimeAsia/Tokyo, Calls, Priority
Tokyo Office, 300, 9AM, 5PM, 10000, 1
#CustomerName, Duration, StartTimeEurope/London, EndTimeEurope/London, Calls, Priority
London Office, 240, 10AM, 6PM, 8000, 2
`,
			expectedData: []models.CallData{
				{
					CustomerName:               "Tokyo Office",
					AverageCallDurationSeconds: 300,
					StartTime: func() time.Time {
						loc, _ := time.LoadLocation("Asia/Tokyo")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
					}(),
					EndTime: func() time.Time {
						loc, _ := time.LoadLocation("Asia/Tokyo")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, loc)
					}(),
					Location:      func() *time.Location { l, _ := time.LoadLocation("Asia/Tokyo"); return l }(),
					NumberOfCalls: 10000,
					Priority:      1,
				},
				{
					CustomerName:               "London Office",
					AverageCallDurationSeconds: 240,
					StartTime: func() time.Time {
						loc, _ := time.LoadLocation("Europe/London")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
					}(),
					EndTime: func() time.Time {
						loc, _ := time.LoadLocation("Europe/London")
						now := time.Now().In(loc)
						return time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, loc)
					}(),
					Location:      func() *time.Location { l, _ := time.LoadLocation("Europe/London"); return l }(),
					NumberOfCalls: 8000,
					Priority:      2,
				},
			},
			expectedError: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := strings.NewReader(strings.TrimSpace(tt.input))
			got, err := parser.Parse(r)

			if tt.expectedError != nil {
				// Check if it's a wrapped error or string match
				if !errors.Is(err, tt.expectedError) && err.Error() != tt.expectedError.Error() {
					t.Errorf("Parse() error = %v, expectedError %v", err, tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}

			assert.Equal(t, got, tt.expectedData, fmt.Sprintf("Parse() = %v, want %v", got, tt.expectedData))
		})
	}
}
