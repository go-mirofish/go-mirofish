package report

import reportstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/report"

func NewProgress(status string, progress int, message string) reportstore.Progress {
	return reportstore.Progress{
		Status:   status,
		Progress: progress,
		Message:  message,
	}
}
