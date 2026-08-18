package tokendirectory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const testIndexJSON = `{
  "index": {
    "mainnet": {
      "chainId": 1,
      "deprecated": false,
      "tokenLists": {
        "erc20.json": "18a24e9ad50b62bc630dca28b256ecc38b9c040a1321184fee384517e40ae1f7"
      }
    }
  }
}`

const testTokenListJSON = `{
  "name": "Test List",
  "chainId": 1,
  "tokenStandard": "ERC20",
  "tokens": [
    {
      "chainId": 1,
      "address": "0x0000000000000000000000000000000000000000",
      "name": "Ether",
      "symbol": "ETH",
      "decimals": 18
    }
  ]
}`

// testServer is an httptest server that records the number of hits and the
// request paths it received.
type testServer struct {
	*httptest.Server

	mu    sync.Mutex
	hits  int
	paths []string
}

func newTestServer(t *testing.T, status int, body string) *testServer {
	t.Helper()
	return newHandlerTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
}

func newHandlerTestServer(t *testing.T, handler http.HandlerFunc) *testServer {
	t.Helper()
	ts := &testServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		ts.hits++
		ts.paths = append(ts.paths, r.URL.Path)
		ts.mu.Unlock()
		handler(w, r)
	}))
	return ts
}

func (ts *testServer) hitCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.hits
}

func (ts *testServer) requestPaths() []string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return slices.Clone(ts.paths)
}

// withTestSources overrides the primary and fallback base URLs with the given
// test servers for the duration of the test. Tests using this must not run in
// parallel.
func withTestSources(t *testing.T, primary, fallback *testServer) {
	t.Helper()
	oldBase := tokenDirectoryBaseSourceURL
	oldFallback := tokenDirectoryFallbackSourceURL
	tokenDirectoryBaseSourceURL = primary.URL
	tokenDirectoryFallbackSourceURL = fallback.URL
	t.Cleanup(func() {
		tokenDirectoryBaseSourceURL = oldBase
		tokenDirectoryFallbackSourceURL = oldFallback
	})
}

func TestFetchFromURLs(t *testing.T) {
	td := NewTokenDirectory()
	ctx := context.Background()

	t.Run("primary ok", func(t *testing.T) {
		primary := newTestServer(t, http.StatusOK, "primary-body")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		buf, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(buf) != "primary-body" {
			t.Fatalf("expected primary body, got %q", buf)
		}
		if primary.hitCount() != 1 || fallback.hitCount() != 0 {
			t.Fatalf("expected 1 primary hit and 0 fallback hits, got %d/%d", primary.hitCount(), fallback.hitCount())
		}
	})

	t.Run("primary 500 falls back", func(t *testing.T) {
		primary := newTestServer(t, http.StatusInternalServerError, "")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		buf, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(buf) != "fallback-body" {
			t.Fatalf("expected fallback body, got %q", buf)
		}
	})

	t.Run("primary 429 falls back", func(t *testing.T) {
		primary := newTestServer(t, http.StatusTooManyRequests, "")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		buf, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(buf) != "fallback-body" {
			t.Fatalf("expected fallback body, got %q", buf)
		}
	})

	t.Run("primary network error falls back", func(t *testing.T) {
		closed := newTestServer(t, http.StatusOK, "")
		closedURL := closed.URL
		closed.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		buf, err := td.fetchFromURLs(ctx, closedURL, fallback.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(buf) != "fallback-body" {
			t.Fatalf("expected fallback body, got %q", buf)
		}
	})

	t.Run("stalled primary falls back", func(t *testing.T) {
		// shorten the per-attempt timeout so the test stays fast
		oldTimeout := sourceAttemptTimeout
		sourceAttemptTimeout = 200 * time.Millisecond
		defer func() { sourceAttemptTimeout = oldTimeout }()

		// primary accepts the connection but never responds
		primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		start := time.Now()
		buf, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(buf) != "fallback-body" {
			t.Fatalf("expected fallback body, got %q", buf)
		}
		if fallback.hitCount() != 1 {
			t.Fatalf("expected fallback to be used, got %d hits", fallback.hitCount())
		}
		// the stalled primary must have been cut off by the per-attempt timeout,
		// not by the test giving up
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("expected per-attempt timeout to cut off the stalled primary, took %v", elapsed)
		}
	})

	t.Run("stalled sources report ErrSourceTimeout", func(t *testing.T) {
		oldTimeout := sourceAttemptTimeout
		sourceAttemptTimeout = 100 * time.Millisecond
		defer func() { sourceAttemptTimeout = oldTimeout }()

		stall := func() *httptest.Server {
			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			}))
		}
		primary := stall()
		defer primary.Close()
		fallback := stall()
		defer fallback.Close()

		_, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err == nil {
			t.Fatal("expected error when all sources stall")
		}
		if !errors.Is(err, ErrSourceTimeout) {
			t.Fatalf("expected ErrSourceTimeout, got: %v", err)
		}
		// the attempt's deadline must not masquerade as the caller's deadline
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("did not expect context.DeadlineExceeded to be visible, got: %v", err)
		}
	})

	t.Run("canceled context makes no requests", func(t *testing.T) {
		primary := newTestServer(t, http.StatusOK, "primary-body")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, "fallback-body")
		defer fallback.Close()

		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := td.fetchFromURLs(cancelledCtx, primary.URL, fallback.URL); err == nil {
			t.Fatal("expected error with canceled context")
		}
		if primary.hitCount() != 0 || fallback.hitCount() != 0 {
			t.Fatalf("expected no requests with canceled context, got %d/%d hits", primary.hitCount(), fallback.hitCount())
		}
	})

	t.Run("all fail reports both errors", func(t *testing.T) {
		primary := newTestServer(t, http.StatusInternalServerError, "")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusNotFound, "")
		defer fallback.Close()

		_, err := td.fetchFromURLs(ctx, primary.URL, fallback.URL)
		if err == nil {
			t.Fatal("expected error when all URLs fail")
		}
		// both failures should be reported, not just the last one
		if !strings.Contains(err.Error(), primary.URL) || !strings.Contains(err.Error(), fallback.URL) {
			t.Fatalf("expected error to mention both URLs, got: %v", err)
		}
	})

	t.Run("no urls", func(t *testing.T) {
		if _, err := td.fetchFromURLs(ctx); err == nil {
			t.Fatal("expected error when no urls provided")
		}
	})

	t.Run("single url is not limited by failover timeout", func(t *testing.T) {
		oldTimeout := sourceAttemptTimeout
		sourceAttemptTimeout = 20 * time.Millisecond
		defer func() { sourceAttemptTimeout = oldTimeout }()

		server := newHandlerTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte("slow-body"))
		})
		defer server.Close()

		buf, err := td.fetchFromURLs(ctx, server.URL)
		if err != nil {
			t.Fatalf("single URL should honor the caller context, got: %v", err)
		}
		if string(buf) != "slow-body" {
			t.Fatalf("expected slow response body, got %q", buf)
		}
	})
}

func TestDefaultClientIsPreserved(t *testing.T) {
	td := NewTokenDirectory()
	if td.client != http.DefaultClient {
		t.Fatal("expected NewTokenDirectory to preserve http.DefaultClient")
	}

	customClient := &http.Client{Timeout: time.Second}
	if got := NewTokenDirectory(Options{HTTPClient: customClient}).client; got != customClient {
		t.Fatal("expected the configured HTTP client to be preserved")
	}
}

func TestCallerCancellationDoesNotMarkPrimaryUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	primary := newHandlerTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	})
	defer primary.Close()
	fallback := newTestServer(t, http.StatusOK, "fallback-body")
	defer fallback.Close()

	td := NewTokenDirectory()
	_, err := td.fetchManagedURLs(ctx, primary.URL, fallback.URL, false, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected caller cancellation, got: %v", err)
	}
	if fallback.hitCount() != 0 {
		t.Fatalf("expected caller cancellation to prevent fallback, got %d hits", fallback.hitCount())
	}
	td.mu.Lock()
	preferFallback := td.preferFallback
	td.mu.Unlock()
	if preferFallback {
		t.Fatal("caller cancellation should not mark the primary unavailable")
	}
}

func TestFallbackURLFor(t *testing.T) {
	if got := fallbackURLFor(TokenDirectoryTokenListURL("mainnet", "erc20.json")); got != TokenDirectoryFallbackTokenListURL("mainnet", "erc20.json") {
		t.Fatalf("unexpected fallback token list URL: %s", got)
	}
	if got := fallbackURLFor(TokenDirectoryIndexURL()); got != TokenDirectoryFallbackIndexURL() {
		t.Fatalf("unexpected fallback index URL: %s", got)
	}
	other := "https://example.com/some/list.json"
	if got := fallbackURLFor(other); got != other {
		t.Fatalf("expected non-primary URL unchanged, got %s", got)
	}
	// only the prefix should be rewritten, not occurrences elsewhere in the URL
	embedded := "https://example.com/proxy?u=" + TokenDirectoryTokenListURL("mainnet", "erc20.json")
	if got := fallbackURLFor(embedded); got != embedded {
		t.Fatalf("expected URL with embedded primary URL unchanged, got %s", got)
	}
}

func TestFetchIndexFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("primary ok", func(t *testing.T) {
		primary := newTestServer(t, http.StatusOK, testIndexJSON)
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, testIndexJSON)
		defer fallback.Close()
		withTestSources(t, primary, fallback)

		td := NewTokenDirectory()
		index, err := td.FetchIndex(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(index[1]) != 1 {
			t.Fatalf("expected 1 entry for chain 1, got %d", len(index[1]))
		}
		if primary.hitCount() == 0 || fallback.hitCount() != 0 {
			t.Fatalf("expected primary used and fallback untouched, got %d/%d hits", primary.hitCount(), fallback.hitCount())
		}
		if !slices.Equal(primary.requestPaths(), []string{"/index.json"}) {
			t.Fatalf("unexpected primary request paths: %v", primary.requestPaths())
		}
	})

	t.Run("primary down falls back", func(t *testing.T) {
		primary := newTestServer(t, http.StatusServiceUnavailable, "")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, testIndexJSON)
		defer fallback.Close()
		withTestSources(t, primary, fallback)

		td := NewTokenDirectory()
		index, err := td.FetchIndex(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(index[1]) != 1 {
			t.Fatalf("expected 1 entry for chain 1, got %d", len(index[1]))
		}
		if primary.hitCount() == 0 || fallback.hitCount() == 0 {
			t.Fatalf("expected both primary and fallback to be hit, got %d/%d", primary.hitCount(), fallback.hitCount())
		}
		if !slices.Equal(fallback.requestPaths(), []string{"/index.json"}) {
			t.Fatalf("unexpected fallback request paths: %v", fallback.requestPaths())
		}
	})

	t.Run("malformed primary falls back", func(t *testing.T) {
		primary := newTestServer(t, http.StatusOK, "not-json")
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, testIndexJSON)
		defer fallback.Close()
		withTestSources(t, primary, fallback)

		td := NewTokenDirectory()
		index, err := td.FetchIndex(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(index[1]) != 1 {
			t.Fatalf("expected fallback index entry, got %d", len(index[1]))
		}
		if primary.hitCount() != 1 || fallback.hitCount() != 1 {
			t.Fatalf("expected malformed primary and valid fallback to be tried once, got %d/%d", primary.hitCount(), fallback.hitCount())
		}
	})
}

func TestFetchTokenListFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("index failure makes token lists prefer fallback", func(t *testing.T) {
		primary := newTestServer(t, http.StatusTooManyRequests, "")
		defer primary.Close()
		fallback := newHandlerTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/index.json" {
				_, _ = w.Write([]byte(testIndexJSON))
				return
			}
			_, _ = w.Write([]byte(testTokenListJSON))
		})
		defer fallback.Close()
		withTestSources(t, primary, fallback)

		td := NewTokenDirectory()
		url := TokenDirectoryTokenListURL("mainnet", "erc20.json")
		tokenList, err := td.FetchTokenList(ctx, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokenList.Tokens) != 1 {
			t.Fatalf("expected 1 token, got %d", len(tokenList.Tokens))
		}
		if tokenList.TokenListURL != url {
			t.Fatalf("expected token list URL to remain the primary URL, got %s", tokenList.TokenListURL)
		}
		if !slices.Contains(fallback.requestPaths(), "/index.json") || !slices.Contains(fallback.requestPaths(), "/mainnet/erc20.json") {
			t.Fatalf("unexpected fallback request paths: %v", fallback.requestPaths())
		}
		if paths := primary.requestPaths(); !slices.Equal(paths, []string{"/index.json"}) {
			t.Fatalf("expected the failed primary to be skipped for the token list, paths: %v", paths)
		}
	})

	t.Run("content hash mismatch falls back", func(t *testing.T) {
		primary := newHandlerTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/index.json" {
				_, _ = w.Write([]byte(testIndexJSON))
				return
			}
			_, _ = w.Write([]byte(strings.Replace(testTokenListJSON, "Test List", "Stale List", 1)))
		})
		defer primary.Close()
		fallback := newTestServer(t, http.StatusOK, testTokenListJSON)
		defer fallback.Close()
		withTestSources(t, primary, fallback)

		td := NewTokenDirectory()
		url := TokenDirectoryTokenListURL("mainnet", "erc20.json")
		tokenList, err := td.FetchTokenList(ctx, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenList.Name != "Test List" {
			t.Fatalf("expected hash-matched fallback token list, got %q", tokenList.Name)
		}
		if !slices.Contains(primary.requestPaths(), "/mainnet/erc20.json") || !slices.Contains(fallback.requestPaths(), "/mainnet/erc20.json") {
			t.Fatalf("expected both token-list sources to be tried, got primary=%v fallback=%v", primary.requestPaths(), fallback.requestPaths())
		}
	})
}
