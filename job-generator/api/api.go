package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	slog.Info("Generating image", "url", reqURL)

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
			Image    string  `json:"image"`
			Duration float64 `json:"duration"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			slog.Error("Error decoding response JSON", "error", err)
			return 0, err
		}

		if r.Image == "" {
			slog.Error("Invalid image data in response")
			return 0, fmt.Errorf("empty image in response")
		}

		slog.Info("Image generated successfully", "id", id, "duration", r.Duration)
		if config.C.SaveImage {
			imageData, err := base64.StdEncoding.DecodeString(r.Image)
			if err != nil {
				slog.Error("Error decoding base64 image", "error", err)
				return 0, err
			}
			err = saveImage(filepath.Join(config.C.ImageStorePath, id), imageData)
			if err != nil {
				slog.Error("Error saving image", "error", err)
				return 0, err
			}
		}
		return time.Duration(r.Duration * float64(time.Second)), nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Image generation failed", "response", string(body))
		return 0, fmt.Errorf("image generation failed: %s", string(body))
	}
}

func saveImage(filename string, data []byte) error {
	file, err := os.Create(filename + ".png")
	if err != nil {
		return fmt.Errorf("error creating image file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("error saving image: %w", err)
	}
	return nil
}
