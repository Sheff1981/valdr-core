package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var ErrRPCResponse = errors.New("RPC request failed")

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "http://127.0.0.1:17332"
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(endpoint, "/rpc") {
		endpoint += "/rpc"
	}
	return &Client{
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	request := struct {
		Method string `json:"method"`
		Params any    `json:"params,omitempty"`
	}{
		Method: method,
		Params: params,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxRPCRequestBytes+1))
	if err != nil {
		return err
	}
	if len(raw) > maxRPCRequestBytes {
		return ErrRPCResponse
	}

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *RPCError       `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf(
			"%w: code=%d message=%s",
			ErrRPCResponse,
			envelope.Error.Code,
			envelope.Error.Message,
		)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: HTTP %d", ErrRPCResponse, response.StatusCode)
	}
	if result == nil {
		return nil
	}
	if len(envelope.Result) == 0 {
		return ErrRPCResponse
	}
	return json.Unmarshal(envelope.Result, result)
}
