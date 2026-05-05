package handler_test

import (
	"bytes"
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

	h := handler.NewRouter(svc, "http://example", logger, nil, nil)
	ts := httptest.NewServer(h)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader("http://example.com"))
	req.Header.Set("Content-Type", "text/plain")

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(string(b), "http://example/"))

	// Output:
	// 201
	// true
}

func Example_postJSON() {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.InfoLevel)

	h := handler.NewRouter(svc, "http://example", logger, nil, nil)
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := []byte(`{"url":"https://practicum.yandex.ru/"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/shorten", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	s := string(b)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.Contains(s, `"result":"http://example/`))

	// Output:
	// 201
	// true
}