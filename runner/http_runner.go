package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTTPRunner struct {
	client   *http.Client
	endpoint string
}

type HTTPRunnerOption func(runner *HTTPRunner)

func WithClient(c *http.Client) HTTPRunnerOption {
	return func(runner *HTTPRunner) {
		runner.client = c
	}
}

func WithEndpoint(endpoint string) HTTPRunnerOption {
	return func(runner *HTTPRunner) {
		runner.endpoint = endpoint
	}
}

func NewHTTPRunner(opts ...HTTPRunnerOption) HTTPRunner {
	h := HTTPRunner{
		client:   http.DefaultClient,
		endpoint: "https://pelevin.gpt.dobro.ai/generate/",
	}

	for _, opt := range opts {
		opt(&h)
	}

	return h
}

func (h HTTPRunner) Query(ctx context.Context, q Query) (r Result, err error) {
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(q)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.endpoint, &buf)
	if err != nil {
		return
	}
	defer req.Body.Close()

	req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4000.1`)
	req.Header.Set("Origin", "https://porfirevich.ru")

	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s, _ := ioutil.ReadAll(resp.Body)
		err = fmt.Errorf("bad http code: %s", string(s))
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&r)
	return
}
