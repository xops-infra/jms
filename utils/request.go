package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LowApiError struct {
	ErrMsg  string `json:"errmsg"`
	ErrCode uint   `json:"errcode"`
}

func (e *LowApiError) Error() string {
	return fmt.Sprintf("%d:%s", e.ErrCode, e.ErrMsg)
}

type marshaller interface {
	marshal(value any) ([]byte, error)
}

type jsonMarshaller struct{}

func (jm *jsonMarshaller) marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

type requestBuilder interface {
	build(ctx context.Context, method, url string, request any) (*http.Request, error)
	sendRequest(req *http.Request, v any) error
}

type httpRequestBuilder struct {
	marshaller marshaller
}

func newRequestBuilder() *httpRequestBuilder {
	return &httpRequestBuilder{
		marshaller: &jsonMarshaller{},
	}
}

func (b *httpRequestBuilder) build(ctx context.Context, method, url string, request any) (*http.Request, error) {
	if request == nil {
		return http.NewRequestWithContext(ctx, method, url, nil)
	}

	var reqBytes []byte
	reqBytes, err := b.marshaller.marshal(request)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(
		ctx,
		method,
		url,
		bytes.NewBuffer(reqBytes),
	)
}

func (b *httpRequestBuilder) sendRequest(req *http.Request, v any) error {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		body := new(string)
		decodeResponse(res.Body, body)
		return fmt.Errorf("response status code: %d,%s", res.StatusCode, *body)
	}

	defer res.Body.Close()

	return decodeResponse(res.Body, v)
}

func decodeResponse(body io.Reader, v any) error {
	if v == nil {
		return nil
	}

	if result, ok := v.(*string); ok {
		return decodeString(body, result)
	}
	return json.NewDecoder(body).Decode(v)
}

func decodeString(body io.Reader, output *string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	*output = string(b)
	return nil
}
