package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"time"
)

type JSONWriter struct{}

func (JSONWriter) Write(ctx context.Context, reportData Report) ([]byte, error) {
	_ = ctx
	return json.MarshalIndent(reportData, "", "  ")
}

type CSVWriter struct{}

func (CSVWriter) Write(ctx context.Context, reportData Report) ([]byte, error) {
	_ = ctx
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.Write([]string{"key", "source", "timestamp"}); err != nil {
		return nil, err
	}
	for _, result := range reportData.Results {
		if err := writer.Write([]string{result.Key, result.Source, result.Timestamp.Format(time.RFC3339)}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}
