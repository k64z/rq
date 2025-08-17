package rq

import "testing"

func TestProxyFromURL(t *testing.T) {
	tests := map[string]struct {
		proxyURL string
		want     *ProxyConfig
		wantErr  bool
	}{
		"HTTP proxy without auth": {
			proxyURL: "http://proxy.example.com:8080",
			want: &ProxyConfig{
				Type: ProxyTypeHTTP,
				Host: "proxy.example.com",
				Port: "8080",
			},
		},
		"HTTP proxy with auth": {
			proxyURL: "http://user:pass@proxy.example.com:8080",
			want: &ProxyConfig{
				Type:     ProxyTypeHTTP,
				Host:     "proxy.example.com",
				Port:     "8080",
				Username: "user",
				Password: "pass",
			},
		},
		"SOCKS5 proxy with auth": {
			proxyURL: "socks5://user:pass@proxy.example.com:1080",
			want: &ProxyConfig{
				Type:     ProxyTypeSOCKS5,
				Host:     "proxy.example.com",
				Port:     "1080",
				Username: "user",
				Password: "pass",
			},
		},
		"HTTP proxy with default port": {
			proxyURL: "http://proxy.example.com",
			want: &ProxyConfig{
				Type: "http",
				Host: "proxy.example.com",
				Port: "8080",
			},
		},
		"SOCKS5 proxy with default port": {
			proxyURL: "socks5://proxy.example.com",
			want: &ProxyConfig{
				Type: "socks5",
				Host: "proxy.example.com",
				Port: "1080",
			},
		},
		"invalid URL": {
			proxyURL: "://invalid",
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ProxyFromURL(tt.proxyURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !proxyConfigEqual(got, tt.want) {
				t.Errorf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestProxyConfigURL(t *testing.T) {
	tests := map[string]struct {
		config *ProxyConfig
		want   string
	}{
		"HTTP proxy without auth": {
			config: &ProxyConfig{
				Type: ProxyTypeHTTP,
				Host: "proxy.example.com",
				Port: "8080",
			},
			want: "http://proxy.example.com:8080",
		},
		"HTTP proxy with auth": {
			config: &ProxyConfig{
				Type:     ProxyTypeHTTP,
				Host:     "proxy.example.com",
				Port:     "8080",
				Username: "user",
				Password: "pass",
			},
			want: "http://user:pass@proxy.example.com:8080",
		},
		"SOCKS5 proxy with username only": {
			config: &ProxyConfig{
				Type:     ProxyTypeSOCKS5,
				Host:     "proxy.example.com",
				Port:     "1080",
				Username: "user",
			},
			want: "socks5://user@proxy.example.com:1080",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.config.URL()
			if got.String() != tt.want {
				t.Errorf("got %s want %s", got.String(), tt.want)
			}
		})
	}
}

func TestProxyConfigAddress(t *testing.T) {
	config := &ProxyConfig{
		Host: "proxy.example.com",
		Port: "8080",
	}

	got := config.Address()
	want := "proxy.example.com:8080"

	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func proxyConfigEqual(a, b *ProxyConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Type == b.Type &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.Username == b.Username &&
		a.Password == b.Password
}
