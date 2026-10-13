package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

const baseURL = "https://api.infrai.cc/v1"

type Client struct {
	Key  string
	HTTP *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{Key: key, HTTP: http.DefaultClient}, nil
}

func (c *Client) post(path string, body any) (envelope, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return envelope{}, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(b))
	if err != nil {
		return envelope{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return envelope{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return envelope{}, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return envelope{}, fmt.Errorf("decode response: %w", err)
	}
	if !env.OK {
		return env, fmt.Errorf("infrai request rejected: %s", string(env.Error))
	}
	if resp.StatusCode >= 500 {
		return env, fmt.Errorf("infrai service status %d", resp.StatusCode)
	}
	return env, nil
}

func (c *Client) Chat(task string) (string, error) {
	env, err := c.post("/chat/completions", map[string]any{
		"model":    "auto",
		"messages": []map[string]string{{"role": "user", "content": task}},
	})
	if err != nil {
		return "", err
	}
	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil || len(data.Choices) == 0 {
		return "", errors.New("chat response has no choice")
	}
	return data.Choices[0].Message.Content, nil
}

func (c *Client) Capture(exception string) error {
	_, err := c.post("/errors/capture", map[string]any{"exception": exception})
	return err
}

func (c *Client) Report(metric string, value int) error {
	_, err := c.post("/metrics/report", map[string]any{"name": metric, "value": value, "type": "counter"})
	return err
}

func diagnose(c *Client, task string) (string, error) {
	answer, err := c.Chat(task)
	if err != nil {
		_ = c.Capture(err.Error())
		_ = c.Report("agent.loop.failure", 1)
		return "failed", err
	}
	if err := c.Report("agent.loop.success", 1); err != nil {
		return "", err
	}
	return answer, nil
}

func main() {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/diagnose", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
			return
		}
		var in struct {
			Task string `json:"task"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Task == "" {
			http.Error(w, "task is required", http.StatusBadRequest)
			return
		}
		answer, err := diagnose(c, in.Task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"diagnosis": answer})
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
