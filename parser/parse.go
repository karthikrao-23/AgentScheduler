package parser

import (
	"agent-scheduler/errors"
	"agent-scheduler/models"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Parse reads CSV data from the reader and returns a slice of CallData.
// It expects lines starting with '#' to be headers/comments.
// The time fields are expected to be in "3PM" or "3:04PM" format.
// The timezone is determined by the header column (e.g., StartTimePT -> Pacific Time).
// Supports both US timezone codes (PT, ET, CT, MT, UTC) and full IANA timezone names
// (e.g., StartTimeAsia/Tokyo, StartTimeEurope/London) for international timezones.
// Multiple timezone headers can appear throughout the CSV; each sets the timezone
// for all subsequent rows until the next timezone header is encountered.
// Defaults to Pacific Time if not specified.
func Parse(r io.Reader) ([]models.CallData, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	// Set default location to Pacific Time
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return nil, fmt.Errorf("error loading location: %w", err)
	}
	var data []models.CallData
	lineNum := 0

	for {
		record, err := reader.Read()
		lineNum++
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV at line %d: %w", lineNum, err)
		}

		// Handle headers/comments
		if len(record) > 0 && strings.HasPrefix(record[0], "#") {
			// Check for timezone definition in header
			// We expect the 3rd field (index 2) to be StartTimeXX
			if len(record) >= 4 {
				headerTime := strings.TrimSpace(record[2])
				if strings.HasPrefix(headerTime, "StartTime") {
					tzCode := strings.TrimPrefix(headerTime, "StartTime")
					// Only process if we can resolve a timezone from it
					if newLoc, err := getTimezoneLocation(tzCode); err == nil {
						// Update the current timezone for subsequent rows
						loc = newLoc
					}
				}
			}
			continue
		}

		if len(record) != 6 {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    errors.ErrInvalidFieldCount,
			}
		}

		cd := models.CallData{}
		cd.Location = loc
		cd.CustomerName = strings.TrimSpace(record[0])

		cd.AverageCallDurationSeconds, err = strconv.Atoi(strings.TrimSpace(record[1]))
		if err != nil {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    fmt.Errorf("%w: %v", errors.ErrInvalidDuration, err),
			}
		}

		// Parse times using "3:04PM" or "3PM" format
		// Note: This sets the date to the current date to handle DST correctly.
		layouts := []string{"3:04PM", "3PM"}
		var parseErr error

		cd.StartTime, parseErr = parseTime(strings.TrimSpace(record[2]), layouts, loc)
		if parseErr != nil {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    fmt.Errorf("%w: %v", errors.ErrInvalidStartTime, parseErr),
			}
		}

		cd.EndTime, parseErr = parseTime(strings.TrimSpace(record[3]), layouts, loc)
		if parseErr != nil {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    fmt.Errorf("%w: %v", errors.ErrInvalidEndTime, parseErr),
			}
		}

		cd.NumberOfCalls, err = strconv.Atoi(strings.TrimSpace(record[4]))
		if err != nil {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    fmt.Errorf("%w: %v", errors.ErrInvalidNumberOfCalls, err),
			}
		}

		cd.Priority, err = strconv.Atoi(strings.TrimSpace(record[5]))
		if err != nil {
			return nil, &errors.ParseError{
				Line:   lineNum,
				Record: record,
				Err:    fmt.Errorf("%w: %v", errors.ErrInvalidPriority, err),
			}
		}

		data = append(data, cd)
	}

	return data, nil
}

func parseTime(value string, layouts []string, loc *time.Location) (time.Time, error) {
	var lastErr error
	now := time.Now().In(loc)
	for _, layout := range layouts {
		// ParseInLocation uses year 0 if not specified.
		// We want to use the current date to respect DST rules for "today".
		t, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			// Normalize to today's date
			t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func getTimezoneLocation(code string) (*time.Location, error) {
	code = strings.TrimSpace(code)

	// First, try common US timezone abbreviations
	switch code {
	case "PT":
		return time.LoadLocation("America/Los_Angeles")
	case "ET":
		return time.LoadLocation("America/New_York")
	case "CT":
		return time.LoadLocation("America/Chicago")
	case "MT":
		return time.LoadLocation("America/Denver")
	case "UTC":
		return time.UTC, nil
	default:
		// If not a known abbreviation, try to load it as a full IANA timezone name
		// This supports international timezones like "Asia/Tokyo", "Europe/London", etc.
		loc, err := time.LoadLocation(code)
		if err != nil {
			// If that fails too, default to Pacific Time
			return time.LoadLocation("America/Los_Angeles")
		}
		return loc, nil
	}
}
