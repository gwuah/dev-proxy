package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func main() {
	proxyURL, _ := url.Parse("http://localhost:7777")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	reqBody := bytes.NewBuffer([]byte(`{"key":"value"}`))

	resp, err := client.Post("https://echo.free.beeceptor.com", "application/json", reqBody)
	if err != nil {
		panic(err)
	}

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(resBody))
}
