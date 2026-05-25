package telebot

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// BotOption configures bot transport and behavior before the first API call.
type BotOption func(*Bot) error

func defaultHTTPClient() *http.Client {
	return http.DefaultClient
}

func (b *Bot) httpClient() *http.Client {
	if b != nil && b.client != nil {
		return b.client
	}
	return defaultHTTPClient()
}

// WithHTTPClient makes Telegram API requests through the provided client.
func WithHTTPClient(client *http.Client) BotOption {
	return func(b *Bot) error {
		if client == nil {
			return fmt.Errorf("telebot: http client is nil")
		}
		b.client = client
		return nil
	}
}

// WithProxy makes Telegram API requests through the provided HTTP(S) proxy URL.
func WithProxy(rawProxyURL string) BotOption {
	return func(b *Bot) error {
		rawProxyURL = strings.TrimSpace(rawProxyURL)
		if rawProxyURL == "" {
			return nil
		}
		if !strings.Contains(rawProxyURL, "://") {
			rawProxyURL = "http://" + rawProxyURL
		}

		proxyURL, err := url.Parse(rawProxyURL)
		if err != nil {
			return fmt.Errorf("telebot: invalid proxy url: %w", err)
		}

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = http.ProxyURL(proxyURL)
		b.client = &http.Client{Transport: transport}
		return nil
	}
}
