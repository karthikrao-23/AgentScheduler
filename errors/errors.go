package errors

import "fmt"

// ParseError wraps a specific error with context about where it occurred.
type ParseError struct {
	Line   int
	Record []string
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at line %d: %v (record: %v)", e.Line, e.Err, e.Record)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// Define specific error types for better error handling
var (
	ErrInvalidFieldCount    = fmt.Errorf("invalid field count")
	ErrInvalidDuration      = fmt.Errorf("invalid duration")
	ErrInvalidStartTime     = fmt.Errorf("invalid start time")
	ErrInvalidEndTime       = fmt.Errorf("invalid end time")
	ErrInvalidNumberOfCalls = fmt.Errorf("invalid number of calls")
	ErrInvalidPriority      = fmt.Errorf("invalid priority")
	ErrEmptyRecord          = fmt.Errorf("empty record")
)
