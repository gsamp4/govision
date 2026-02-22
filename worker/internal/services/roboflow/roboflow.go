package roboflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"govision/worker/internal/domain"
)

const defaultTimeout = 30 * time.Second

// APIError represents a non-retryable HTTP error from the Roboflow API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("roboflow returned status %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) Detect(ctx context.Context, imageURL string) (*domain.RoboflowResponse, error) {
	log.Printf("[ROBOFLOW] - Downloading image from URL: %s", imageURL)

	imageBytes, err := c.downloadImage(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	log.Printf("[ROBOFLOW] - Image downloaded (%d bytes). Sending to Roboflow...", len(imageBytes))

	result, err := c.infer(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("roboflow inference failed: %w", err)
	}

	log.Printf("[ROBOFLOW] - Inference completed. %d prediction(s) returned.", len(result.Predictions))
	return result, nil
}

func (c *Client) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			ch <- result{err: err}
			return
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- result{err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ch <- result{err: fmt.Errorf("unexpected status %d when downloading image", resp.StatusCode)}
			return
		}

		data, err := io.ReadAll(resp.Body)
		ch <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.data, r.err
	}
}


func (c *Client) infer(ctx context.Context, imageBytes []byte) (*domain.RoboflowResponse, error) {
	encoded := base64.StdEncoding.EncodeToString(imageBytes)

	url := fmt.Sprintf(
		"https://detect.roboflow.com/%s?api_key=%s",
		c.model,
		c.apiKey,
	)

	body := bytes.NewReader([]byte(encoded))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to roboflow failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBytes)}
	}

	var roboflowResp domain.RoboflowResponse
	if err := json.Unmarshal(respBytes, &roboflowResp); err != nil {
		return nil, fmt.Errorf("failed to decode roboflow response: %w", err)
	}

	return &roboflowResp, nil
}
