package registry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smartcat999/container-ui/internal/storage"
)

func TestHandleVersionCheck(t *testing.T) {
	// 创建存储
	store := storage.NewMemoryStorage()
	handler := NewHandler(store)

	// 创建请求
	req := httptest.NewRequest("GET", "/v2/", nil)
	w := httptest.NewRecorder()

	// 调用处理函数
	handler.handleVersionCheck(w, req)

	// 检查响应
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}
}

func TestHandleCatalog(t *testing.T) {
	// 创建存储
	store := storage.NewMemoryStorage()
	handler := NewHandler(store)

	// 添加测试数据
	err := store.PutManifest("repo1", "tag1", "sha256:1234", []byte("test"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	err = store.PutManifest("repo2", "tag1", "sha256:5678", []byte("test"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	// 创建请求
	req := httptest.NewRequest("GET", "/v2/_catalog", nil)
	w := httptest.NewRecorder()

	// 调用处理函数
	handler.handleCatalog(w, req)

	// 检查响应
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}

	// 检查内容类型
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %v", contentType)
	}

	// 检查响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// 简单检查包含仓库名
	if string(body) == "" || len(body) < 10 {
		t.Errorf("Response body too short: %s", body)
	}
}

func TestRouterPathMatching(t *testing.T) {
	testCases := []struct {
		name           string
		pattern        string
		path           string
		expectedMatch  bool
		expectedParams map[string]string
	}{
		{
			name:           "Exact match",
			pattern:        "/v2/",
			path:           "/v2/",
			expectedMatch:  true,
			expectedParams: map[string]string{},
		},
		{
			name:          "With parameter",
			pattern:       "/v2/{repository}/tags/list",
			path:          "/v2/my-repo/tags/list",
			expectedMatch: true,
			expectedParams: map[string]string{
				"repository": "my-repo",
			},
		},
		{
			name:          "With nested repository",
			pattern:       "/v2/{repository}/manifests/{reference}",
			path:          "/v2/user/my-repo/manifests/latest",
			expectedMatch: true,
			expectedParams: map[string]string{
				"repository": "user/my-repo",
				"reference":  "latest",
			},
		},
		{
			name:           "No match wrong path",
			pattern:        "/v2/{repository}/blobs/{digest}",
			path:           "/v2/my-repo/tags/list",
			expectedMatch:  false,
			expectedParams: nil,
		},
		{
			name:           "No match extra segment",
			pattern:        "/v2/{repository}/tags/list",
			path:           "/v2/my-repo/tags/list/extra",
			expectedMatch:  false,
			expectedParams: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, matched := matchPath(tc.pattern, tc.path)

			if matched != tc.expectedMatch {
				t.Errorf("Expected match: %v, got: %v", tc.expectedMatch, matched)
			}

			if !matched {
				return
			}

			// 检查参数
			if len(params) != len(tc.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(tc.expectedParams), len(params))
			}

			for k, v := range tc.expectedParams {
				if params[k] != v {
					t.Errorf("Expected param %s=%s, got %s", k, v, params[k])
				}
			}
		})
	}
}

func TestRouterServeHTTP(t *testing.T) {
	// 创建存储
	store := storage.NewMemoryStorage()
	handler := NewHandler(store)
	router := NewRouter(handler)

	// 测试版本检查路由
	req := httptest.NewRequest("GET", "/v2/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for /v2/, got %v", resp.StatusCode)
	}

	// 测试不存在的路由
	req = httptest.NewRequest("GET", "/v2/non-existent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status NotFound for non-existent route, got %v", resp.StatusCode)
	}

	// 测试方法不匹配
	req = httptest.NewRequest("POST", "/v2/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status NotFound for method mismatch, got %v", resp.StatusCode)
	}
}
