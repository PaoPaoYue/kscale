package api

import (
	"bytes"
	"encoding/json"
	"errors"
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

func GetWorkerCount(apiURL string) (running, total int, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/serve/applications/", apiURL))
	if err != nil || resp.StatusCode != http.StatusOK {
		return 0, 0, errors.New("unable to retrieve application configurations")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Applications map[string]map[string]interface{} `json:"applications"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, 0, errors.New("invalid JSON from dashboard")
	}

	app, ok := data.Applications["text2img"]
	if !ok {
		return 0, 0, errors.New("application 'text2img' not found")
	}

	deployments, ok := app["deployments"].(map[string]interface{})
	if !ok {
		return 0, 0, errors.New("invalid deployments format")
	}

	deployment, ok := deployments["image_service"].(map[string]interface{})
	if !ok {
		return 0, 0, errors.New("deployment 'image_service' not found")
	}

	replicas, ok := deployment["replicas"].([]interface{})
	if !ok {
		return 0, 0, errors.New("replicas field missing or invalid")
	}

	total = len(replicas)

	for _, replica := range replicas {
		if rmap, ok := replica.(map[string]interface{}); ok {
			if state, ok := rmap["state"].(string); ok && state == "RUNNING" {
				running++
			}
		}
	}

	return running, total, nil
}

type CalcWorkerCountRequestParam struct {
	Points []DataPoint `json:"points"`
}

type DataPoint struct {
	RunningWorker int     `json:"active_workers"`
	NewJob        int     `json:"num_new_tasks"`
	OngoingJob    int     `json:"num_ongoing_tasks"`
	CompletedJob  int     `json:"num_completed_tasks"`
	AvgDuration   float64 `json:"avg_duration"` // in milliseconds
	AvgDelay      float64 `json:"avg_delay"`    // in milliseconds
}

func CalcWorkerCount(apiURL string, param CalcWorkerCountRequestParam) (int, error) {
	bodyBytes, err := json.Marshal(param)
	if err != nil {
		return 0, errors.New("failed to encode input data")
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/autoscaler/calc", apiURL),
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil || resp.StatusCode != http.StatusOK {
		return 0, errors.New("failed to call autoscaler")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, errors.New("invalid JSON from autoscaler")
	}

	return result.Count, nil
}

func ScaleWorker(apiURL string, count int) error {
	resp, err := http.Get(fmt.Sprintf("%s/api/serve/applications/", apiURL))
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New("unable to retrieve application configurations")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Applications map[string]map[string]interface{} `json:"applications"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return errors.New("invalid JSON from dashboard")
	}

	app, ok := data.Applications["text2img"]
	if !ok {
		return errors.New("application 'text2img' not found")
	}

	deployments, ok := app["deployments"].(map[string]interface{})
	if !ok {
		return errors.New("invalid deployment format")
	}

	deploymentRaw, ok := deployments["image_service"].(map[string]interface{})
	if !ok {
		return errors.New("deployment 'image_service' not found")
	}

	config, ok := deploymentRaw["deployment_config"].(map[string]interface{})
	if !ok {
		config = map[string]interface{}{}
	}

	if _, hasAutoscaling := config["autoscaling_config"]; hasAutoscaling {
		return errors.New("cannot set 'num_replicas' when 'autoscaling_config' is present")
	}

	config["num_replicas"] = count

	payload := map[string]interface{}{
		"applications": []map[string]interface{}{
			{
				"name":        "text2img",
				"import_path": "core.image_service:entrypoint",
				"deployments": []map[string]interface{}{
					{
						"name":              "image_service",
						"deployment_config": config,
					},
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/serve/applications/", apiURL), bytes.NewReader(jsonBytes))
	if err != nil {
		return errors.New("failed to create PUT request")
	}
	req.Header.Set("Content-Type", "application/json")

	putResp, err := http.DefaultClient.Do(req)
	if err != nil || putResp.StatusCode >= 400 {
		return errors.New("failed to update application configuration")
	}
	defer putResp.Body.Close()

	return nil
}
