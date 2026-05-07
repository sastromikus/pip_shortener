package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
	"github.com/sirupsen/logrus"
)

func Example_postTextPlain() {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.InfoLevel)

	h := handler.NewRouter(svc, "http://localhost:8080", logger, nil, nil, "")
	ts := httptest.NewServer(h)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader("http://example.com"))
	if err != nil {
		fmt.Println("request error")
		return
	}
	req.Header.Set("Content-Type", "text/plain")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("request error")
		return
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("read error")
		return
	}

	out := strings.TrimSpace(string(b))

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(out, "http://") || strings.HasPrefix(out, "https://"))
	fmt.Println(strings.Contains(out, "/"))

	// Output:
	// 201
	// true
	// true
}

func Example_postJSON() {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.InfoLevel)

	h := handler.NewRouter(svc, "http://localhost:8080", logger, nil, nil, "")
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := []byte(`{"url":"https://practicum.yandex.ru/"}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("request error")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("request error")
		return
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("read error")
		return
	}

	var resp struct {
		Result string `json:"result"`
	}
	_ = json.Unmarshal(b, &resp)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(resp.Result, "http://") || strings.HasPrefix(resp.Result, "https://"))
	fmt.Println(strings.Contains(resp.Result, "/"))

	// Output:
	// 201
	// true
	// true
}
