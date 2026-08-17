// Package nbapi is a client for the NetBird Management HTTP API.
//
// It is the server-side counterpart of the Next.js dashboard's
// src/utils/api.tsx: same base URL construction, same bearer-token scheme, same
// error envelope. The difference is that requests originate here rather than in
// the browser, so the token never crosses to the client.
package nbapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// requestIDHeader correlates a response with a server-side request. Older
// Management servers do not set it.
const requestIDHeader = "X-Request-Id"

// Error is the Management API's error envelope.
type Error struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"-"`
}

func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("netbird api: %d %s (request id %s)", e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("netbird api: %d %s", e.Code, e.Message)
}

// IsUnauthorized reports whether err is an authentication failure, which the
// handlers turn into a fresh login rather than an error page.
func IsUnauthorized(err error) bool {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusUnauthorized
}

type Client struct {
	// base is the API root, e.g. "https://api.netbird.io/api".
	base string
	hc   *http.Client
}

func New(apiOrigin string) *Client {
	return &Client{
		base: apiOrigin + "/api",
		hc:   &http.Client{Timeout: 30 * time.Second},
	}
}

// do performs a request against the Management API and decodes the response
// into out. A nil out discards the body, which suits DELETE.
func (c *Client) do(ctx context.Context, token, method, path string, params url.Values, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	target := c.base + path
	if len(params) > 0 {
		target += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("call management api: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return decodeError(res)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// decodeError builds an *Error from a failed response, falling back to the HTTP
// status when the body is not the expected JSON envelope (nginx and other
// proxies in front of Management return HTML).
func decodeError(res *http.Response) error {
	apiErr := &Error{
		Code:      res.StatusCode,
		Message:   res.Status,
		RequestID: res.Header.Get(requestIDHeader),
	}

	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || len(payload) == 0 {
		return apiErr
	}

	var decoded Error
	if err := json.Unmarshal(payload, &decoded); err == nil && decoded.Message != "" {
		apiErr.Message = decoded.Message
		if decoded.Code != 0 {
			apiErr.Code = decoded.Code
		}
	}
	return apiErr
}
