package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func forwardToDoH(query []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "https://cloudflare-dns.com/dns-query", bytes.NewReader(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
