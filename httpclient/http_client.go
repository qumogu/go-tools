package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/qumogu/go-tools/logger"
)

const defaultMaxIdleConnsPerHost = 4

var (
	httpMethodMap = map[string]string{
		"get":     http.MethodGet,
		"post":    http.MethodPost,
		"put":     http.MethodPut,
		"delete":  http.MethodDelete,
		"del":     http.MethodDelete,
		"head":    http.MethodHead,
		"patch":   http.MethodPatch,
		"connect": http.MethodConnect,
		"options": http.MethodOptions,
		"trace":   http.MethodTrace,
	}
	methodNotSupportError = errors.New("method not support")
	ApiResponseErr        = errors.New("http api request error")

	JsonHttpHeader = map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
)

type HttpMessage struct {
	Status int               `json:"status"`
	URL    string            `json:"url"`
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Data   string            `json:"data"`
}

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *http.Client
	urlPre string
}

func NewHttpClientTransport(addr string) *Client {
	ctx, fn := context.WithCancel(context.Background())
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
	}
	return &Client{
		ctx:    ctx,
		cancel: fn,
		client: &http.Client{Transport: tr},
		urlPre: addr,
	}
}

func (h *Client) Send(httpMsg *HttpMessage) (*HttpMessage, error) {
	url := h.urlPre + httpMsg.URL
	method, err := checkHttpMethod(httpMsg.Method)
	if err != nil {
		log.Warnw("http client send message check method failed", "addr", h.urlPre, "method", method, "error", err)
		return nil, err
	}

	request, err := http.NewRequestWithContext(h.ctx, method, url, bytes.NewBuffer([]byte(httpMsg.Data)))
	if err != nil {
		log.Warnw("http client send message new request failed", "addr", h.urlPre, "message", httpMsg, "error", err)
		return nil, err
	}
	header := http.Header{}
	for k, v := range httpMsg.Header {
		header.Set(k, v)
	}
	request.Header = header
	request.Context()
	resp, err := h.client.Do(request)
	if err != nil {
		log.Warnw("http client send do request failed", "addr", h.urlPre, "message", httpMsg, "error", err)
		return nil, err
	}

	defer resp.Body.Close()
	binaryBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnw("http client send do request read response failed", "addr", h.urlPre, "message", httpMsg, "error", err)
		return nil, err
	}

	respHeader := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeader[k] = resp.Header.Get(k)
	}

	respMsg := &HttpMessage{
		URL:    request.URL.String(),
		Method: request.Method,
		Header: respHeader,
		Status: resp.StatusCode,
		Data:   string(binaryBody),
	}

	return respMsg, nil
}

func (h *Client) Close() error {
	h.cancel()
	log.Infow("http client closed", "addr", h.urlPre)
	return nil
}

func checkHttpMethod(m string) (string, error) {
	str, ok := httpMethodMap[strings.ToLower(m)]
	if !ok {
		return m, methodNotSupportError
	}

	return str, nil
}
