package tls_client

import (
	"time"

	http "github.com/bogdanfinn/fhttp"
)

type HttpClientOption func(config *httpClientConfig)

type TransportOptions struct {
	DisableKeepAlives      bool
	DisableCompression     bool
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	MaxResponseHeaderBytes int64 // Zero means to use a default limit.
	WriteBufferSize        int   // If zero, a default (currently 4KB) is used.
	ReadBufferSize         int   // If zero, a default (currently 4KB) is used.
}

type httpClientConfig struct {
	debug                       bool
	followRedirects             bool
	insecureSkipVerify          bool
	proxyUrl                    string
	serverNameOverwrite         string
	transportOptions            *TransportOptions
	cookieJar                   http.CookieJar
	clientProfile               ClientProfile
	withRandomTlsExtensionOrder bool
	timeout                     time.Duration
}

func WithProxyUrl(proxyUrl string) HttpClientOption {
	return func(config *httpClientConfig) {
		config.proxyUrl = proxyUrl
	}
}

func WithCookieJar(jar http.CookieJar) HttpClientOption {
	return func(config *httpClientConfig) {
		config.cookieJar = jar
	}
}

func WithTimeout(timeout int) HttpClientOption {
	return func(config *httpClientConfig) {
		config.timeout = time.Second * time.Duration(timeout)
	}
}

func WithNotFollowRedirects() HttpClientOption {
	return func(config *httpClientConfig) {
		config.followRedirects = false
	}
}

func WithRandomTLSExtensionOrder() HttpClientOption {
	return func(config *httpClientConfig) {
		config.withRandomTlsExtensionOrder = true
	}
}

func WithDebug() HttpClientOption {
	return func(config *httpClientConfig) {
		config.debug = true
	}
}

func WithTransportOptions(transportOptions *TransportOptions) HttpClientOption {
	return func(config *httpClientConfig) {
		config.transportOptions = transportOptions
	}
}

func WithInsecureSkipVerify() HttpClientOption {
	return func(config *httpClientConfig) {
		config.insecureSkipVerify = true
	}
}

func WithClientProfile(clientProfile ClientProfile) HttpClientOption {
	return func(config *httpClientConfig) {
		config.clientProfile = clientProfile
	}
}

func WithServerNameOverwrite(serverName string) HttpClientOption {
	return func(config *httpClientConfig) {
		config.serverNameOverwrite = serverName
	}
}
