package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type GenerateRequestParam struct {
	Prompt       string  `json:"prompt"`
	Steps        int     `json:"steps"`
	Scale        float64 `json:"cfg_scale"`
	SamplerIndex string  `json:"sampler_index"`
	Scheduler    string  `json:"scheduler"`

	Width  int `json:"width"`
	Height int `json:"height"`
}

var client = &http.Client{
	Timeout: time.Duration(config.C.APITimeout) * time.Second,
	Transport: &http.Transport{
		ForceAttemptHTTP2: false,
	},
}

func SwitchModel(apiURL, model string) error {
	requestData := map[string]string{"sd_model_checkpoint": model}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		slog.Error("Error marshaling model switch request", "error", err)
		return err
	}

	req, err := http.NewRequest("POST", apiURL+"/sdapi/v1/options", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Error creating model switch request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending model switch request", "error", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		slog.Info("Model switched successfully", "model", model)
		return nil
	} else {
		slog.Error("Failed to switch model", "response", string(respBody))
		return fmt.Errorf("failed to switch model: %s", string(respBody))
	}
}

func GenerateImage(apiURL string, params GenerateRequestParam, filename string) error {
	jsonData, err := json.Marshal(params)
	if err != nil {
		slog.Error("Error marshaling request JSON", "error", err)
		return err
	}

	req, err := http.NewRequest("POST", apiURL+"/sdapi/v1/txt2img", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var r map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			slog.Error("Error decoding response JSON", "error", err)
			return err
		}

		images, ok := r["images"].([]interface{})
		if !ok || len(images) == 0 {
			slog.Error("Invalid image data in response")
			return err
		}
		imageStr, ok := images[0].(string)
		if !ok {
			slog.Error("Invalid image data in response")
			return err
		}

		slog.Info("Image generated successfully", "filename", filename)
		if config.C.SaveImage {
			err := saveImage(filepath.Join(config.C.ImageStorePath, filename), []byte(imageStr))
			if err != nil {
				slog.Error("Error saving image", "error", err)
				return err
			}
		}
		return nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Image generation failed", "response", string(body))
		return fmt.Errorf("image generation failed: %s", string(body))
	}
}

func saveImage(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Error creating image file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("Error saving image: %w", err)
	}
	return nil
}
