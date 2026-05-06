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
	s := string(b)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.Contains(s, `"result":"http://example/`))

	// Output:
	// 201
	// true
}
