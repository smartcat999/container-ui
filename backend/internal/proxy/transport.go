package proxy

import (
	"log"
	"net/http"
)

// RedirectFollowingTransport 自动跟随重定向的传输层
type RedirectFollowingTransport struct {
	*http.Transport
	maxRedirects int
}

// NewRedirectFollowingTransport 创建新的自动跟随重定向的传输层
func NewRedirectFollowingTransport(transport *http.Transport, maxRedirects int) *RedirectFollowingTransport {
	return &RedirectFollowingTransport{
		Transport:    transport,
		maxRedirects: maxRedirects,
	}
}

func (t *RedirectFollowingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	origReq := req.Clone(req.Context())
	var resp *http.Response
	var err error

	for redirects := 0; redirects < t.maxRedirects; redirects++ {
		resp, err = t.Transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if !isRedirect(resp.StatusCode) {
			return resp, nil
		}

		location, err := resp.Location()
		if err != nil {
			return resp, nil
		}

		log.Printf("跟随重定向: %s -> %s", req.URL.String(), location.String())
		resp.Body.Close()

		newReq, err := http.NewRequestWithContext(req.Context(), origReq.Method, location.String(), nil)
		if err != nil {
			return nil, err
		}

		copyHeaders(origReq.Header, newReq.Header)
		req = newReq
	}

	return resp, err
}

func isRedirect(statusCode int) bool {
	return statusCode == http.StatusTemporaryRedirect ||
		statusCode == http.StatusMovedPermanently ||
		statusCode == http.StatusFound ||
		statusCode == http.StatusSeeOther
}

func copyHeaders(src, dst http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
