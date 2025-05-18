package api

import (
	"log/slog"
	"testing"
)

func TestGenerateImage(t *testing.T) {
	param := GenerateRequestParam{
		Prompt:       "a futuristic city at sunset",
		Steps:        25,
		Scale:        7.5,
		SamplerIndex: "DPM++ 2M Karras",
		Width:        512,
		Height:       512,
	}
	id := "test-123"

	duration, err := GenerateImage("http://localhost:8000", param, id)
	if err != nil {
		t.Errorf("GenerateImage failed: %v", err)
	}
	if duration <= 0 {
		t.Errorf("Expected positive duration, got %v", duration)
	}
}

func TestGetWorkerCount(t *testing.T) {
	running, total, err := GetWorkerCount("http://localhost:8265")
	if err != nil {
		t.Errorf("GetWorkerCount failed: %v", err)
	}
	if total < running {
		t.Errorf("Invalid state: running=%d > total=%d", running, total)
	}
	slog.Info("GetWorkerCount", "running", running, "total", total)
}

func TestCalcWorkerCount(t *testing.T) {
	param := CalcWorkerCountRequestParam{
		Time: 10,
		Points: []DataPoint{
			{
				RunningWorker: 3,
				NewJob:        5,
				OngoingJob:    5,
				CompletedJob:  7,
				AvgDuration:   2250.0,
				AvgDelay:      6000.0,
			},
		},
	}

	count, err := CalcWorkerCount("http://localhost:8000", param)
	if err != nil {
		t.Errorf("CalcWorkerCount failed: %v", err)
	}
	if count < 0 {
		t.Errorf("Invalid count: %d", count)
	}
	slog.Info("CalcWorkerCount", "count", count)
}

func TestScaleWorker(t *testing.T) {
	err := ScaleWorker("http://localhost:8265", 1)
	if err != nil {
		t.Errorf("ScaleWorker failed: %v", err)
	}
}
