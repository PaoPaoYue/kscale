package core

import (
	"encoding/csv"
	"github.com/paopaoyue/kscale/job-genrator/api"
	"log/slog"
	"mime/multipart"
	"os"
	"strconv"
	"time"
)

type CSVIterator struct {
	reader       *csv.Reader
	lines        [][]string
	currentIndex int
}

func ReadJobCSV(file multipart.File) (*CSVIterator, error) {
	reader := csv.NewReader(file)

	lines, err := reader.ReadAll()
	if err != nil {
		slog.Error("Error reading CSV header", "err", err)
		return nil, err
	}

	_ = file.Close()

	return &CSVIterator{
		reader:       reader,
		lines:        lines,
		currentIndex: 1, // Skip header
	}, nil
}

func (it *CSVIterator) Next() (Job, bool) {
	if it.currentIndex >= len(it.lines) {
		return Job{}, false
	}
	record := it.lines[it.currentIndex]
	it.currentIndex++
	return Job{
		Id: record[0],
		Param: api.GenerateRequestParam{
			Prompt:       record[1],
			Steps:        parseInt(record[2], 20),
			Scale:        parseFloat(record[3], 7.0),
			SamplerIndex: convertToSamplerIndex(record[4]),
			Width:        parseInt(record[5], 512),
			Height:       parseInt(record[6], 512),
		},
		RequestTime: time.UnixMilli(int64(parseInt(record[8], 0))),
	}, true
}

func (it *CSVIterator) Size() int {
	return len(it.lines) - 1
}

func OpenCSVAndWriteHeader(csvFilePath string, header []string) *os.File {
	file, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error("Error opening CSV file", "err", err)
		return nil
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(header)
	if err != nil {
		slog.Error("Error writing CSV header", "err", err)
	}
	return file
}

func AppendCSV(file *os.File, row []string) {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	err := writer.Write(row)
	if err != nil {
		slog.Error("Error writing CSV row", "err", err)
	}
}

func parseInt(value string, defaultValue int) int {
	if v, err := strconv.Atoi(value); err == nil {
		return v
	}
	return defaultValue
}

func parseFloat(value string, defaultValue float64) float64 {
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		return v
	}
	return defaultValue
}

func formatTimeWithMillis(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func convertToSamplerIndex(value string) string {
	switch value {
	case "1":
		return "DDIM"
	case "2":
		return "PLMS"
	case "3":
		return "Euler"
	case "4":
		return "Euler a"
	case "5":
		return "Heun"
	case "6":
		return "DPM2"
	case "7":
		return "DPM2 a"
	case "8":
		return "DPM++ SDE"
	default:
		return "DPM++ SDE"
	}
}
