package rq

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

type ProxyConfig struct {
	Type     ProxyType
	Host     string
	Port     string
	Username string
	Password string
}

// ProxyFromURL creates a ProxyConfig from a URL string
// Supports formats like:
// - http://proxy.example.com:8080
// - https://user:pass@proxy.example.com:8080
// - socks5://user:pass@proxy.example.com:1080
func ProxyFromURL(proxyURL string) (*ProxyConfig, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	config := &ProxyConfig{
		Type: ProxyType(u.Scheme),
		Host: u.Hostname(),
		Port: u.Port(),
	}

	if u.User != nil {
		config.Username = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
	}

	if config.Port == "" {
		switch config.Type {
		case ProxyTypeHTTP, ProxyTypeHTTPS:
			config.Port = "8080"
		case ProxyTypeSOCKS5:
			config.Port = "1080"
		}
	}

	return config, nil
}

// Address returns the proxy address as host:port
func (p *ProxyConfig) Address() string {
	return net.JoinHostPort(p.Host, p.Port)
}

// URL returns the proxy as *url.URL
func (p *ProxyConfig) URL() *url.URL {
	u := &url.URL{
		Scheme: string(p.Type),
		Host:   p.Address(),
	}

	if p.Username != "" {
		if p.Password != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		} else {
			u.User = url.User(p.Username)
		}
	}

	return u
}

func (p *ProxyConfig) CreateTransport(baseTransport *http.Transport) (*http.Transport, error) {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport.(*http.Transport).Clone()
	} else {
		baseTransport = baseTransport.Clone()
	}

	switch p.Type {
	case ProxyTypeHTTP, ProxyTypeHTTPS:
		baseTransport.Proxy = http.ProxyURL(p.URL())
	case ProxyTypeSOCKS5:
		dialer, err := p.createSOCK5Dialer()
		if err != nil {
			return nil, fmt.Errorf("create SOCKS5 dialer: %w", err)
		}
		baseTransport.DialContext = dialer.DialContext
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", p.Type)
	}

	return baseTransport, nil
}

func (p *ProxyConfig) createSOCK5Dialer() (proxy.ContextDialer, error) {
	var auth *proxy.Auth
	if p.Username != "" {
		auth = &proxy.Auth{
			User:     p.Username,
			Password: p.Password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", p.Address(), auth, &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
		return contextDialer, nil
	}

	return nil, fmt.Errorf("SOCKS5 dialer does not implement ContextDialer")
}

// Proxy creates a new request with proxy configuration
func Proxy(config *ProxyConfig) *Request {
	return New().Proxy(config)
}

// Proxy sets proxy configuration for the request
func (r *Request) Proxy(config *ProxyConfig) *Request {
	if r.err != nil {
		return r
	}

	client := r.client
	if client == nil {
		client = &http.Client{}
	} else {
		client = &http.Client{
			CheckRedirect: client.CheckRedirect,
			Jar:           client.Jar,
			Timeout:       client.Timeout,
		}
	}

	transport, err := config.CreateTransport(getTransport(client))
	if err != nil {
		r.err = fmt.Errorf("configure proxy: %w", err)
		return r
	}

	client.Transport = transport
	r.client = client
	return r
}

// getTransport extracts transport from client
func getTransport(client *http.Client) *http.Transport {
	if client.Transport == nil {
		return nil
	}

	if transport, ok := client.Transport.(*http.Transport); ok {
		return transport
	}

	return nil
}

// ProxyURL creates a new request with proxy from URL string
func ProxyURL(proxyURL string) *Request {
	return New().ProxyURL(proxyURL)
}

// ProxyURL sets proxy configuration from URL string
func (r *Request) ProxyURL(proxyURL string) *Request {
	if r.err != nil {
		return r
	}

	config, err := ProxyFromURL(proxyURL)
	if err != nil {
		r.err = err
		return r
	}

	return r.Proxy(config)
}
