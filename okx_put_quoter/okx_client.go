package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	cfg        Config
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg Config, baseURL string) *Client {
	return &Client{cfg: cfg, baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

type okxEnvelope struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) DoPublic(method, path string, query url.Values, out interface{}) error {
	return c.do(method, path, query, nil, out, false)
}

func (c *Client) DoPrivate(method, path string, query url.Values, body interface{}, out interface{}) error {
	return c.do(method, path, query, body, out, true)
}

func (c *Client) do(method, path string, query url.Values, body interface{}, out interface{}, signed bool) error {
	requestPath := path
	if len(query) > 0 {
		requestPath = path + "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyBytes = b
	}

	req, err := http.NewRequest(method, c.baseURL+requestPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if signed {
		ts := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		preHash := ts + method + requestPath + string(bodyBytes)
		mac := hmac.New(sha256.New, []byte(c.cfg.APISecret))
		mac.Write([]byte(preHash))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		req.Header.Set("OK-ACCESS-KEY", c.cfg.APIKey)
		req.Header.Set("OK-ACCESS-SIGN", sign)
		req.Header.Set("OK-ACCESS-TIMESTAMP", ts)
		req.Header.Set("OK-ACCESS-PASSPHRASE", c.cfg.APIPassphrase)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env okxEnvelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		return fmt.Errorf("decode envelope: %w (body=%s)", err, string(respBytes))
	}
	if env.Code != "0" {
		return fmt.Errorf("okx api error code=%s msg=%s", env.Code, env.Msg)
	}
	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decode data: %w (data=%s)", err, string(env.Data))
		}
	}
	return nil
}
