package report

import "errors"

var (
	ErrInvalidReportRequest = errors.New("invalid report request")
	ErrReportGeneration     = errors.New("report generation failed")
)
