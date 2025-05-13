package api

import (
	"encoding/json"
	"fmt"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type GenerateRequestParam struct {
	Prompt       string  `json:"prompt"`
	Steps        int     `json:"steps"`
	Scale        float64 `json:"cfg_scale"`
	SamplerIndex string  `json:"sampler_index"`

	Width  int `json:"width"`
	Height int `json:"height"`
}

type TextAnalyzeRequestParam struct {
	Text string `json:"text"`
}

var client = &http.Client{
	Timeout: time.Duration(config.C.APITimeout) * time.Second,
	Transport: &http.Transport{
		ForceAttemptHTTP2: false,
	},
}

func GenerateImage(apiURL string, params GenerateRequestParam, id string) (time.Duration, error) {
	reqURL := fmt.Sprintf("%s/generate?prompt=%s&steps=%d&cfg_scale=%.1f&sampler_index=%s&width=%d&height=%d&id=%s",
		apiURL,
		url.QueryEscape(params.Prompt),
		params.Steps,
		params.Scale,
		url.QueryEscape(params.SamplerIndex),
		params.Width,
		params.Height,
		id,
	)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request", "error", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var r struct {
			Duration float64 `json:"duration"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			slog.Error("Error decoding response JSON", "error", err)
			return 0, err
		}
		slog.Info("Image generated successfully", "id", id, "duration", r.Duration)
		return time.Duration(r.Duration * float64(time.Second)), nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Image generation failed", "response", string(body))
		return 0, fmt.Errorf("image generation failed: %s", string(body))
	}
}

func analyzeText(apiURL string, params TextAnalyzeRequestParam, id string) (time.Duration, error) {
	reqURL := fmt.Sprintf("%s/analyze?text=%s",
		apiURL,
		url.QueryEscape(params.Text),
	)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request", "error", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var r struct {
			Duration float64 `json:"duration"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			slog.Error("Error decoding response JSON", "error", err)
			return 0, err
		}
		slog.Info("Text analyzed successfully", "id", id, "duration", r.Duration)
		return time.Duration(r.Duration * float64(time.Second)), nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Text analyzed failed", "response", string(body))
		return 0, fmt.Errorf("image generation failed: %s", string(body))
	}
}

func GetWorkerCount(apiURL string) (count int, err error) {
	reqURL := fmt.Sprintf("%s/controller/replicas", apiURL)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var r struct {
			Count int `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			slog.Error("Error decoding response JSON", "error", err)
			return 0, err
		}
		return r.Count, nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to retrieve worker count: %s", string(body))
	}
}
