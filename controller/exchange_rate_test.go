package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchUsdToCnyFromSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","rates":{"CNY":7.24,"EUR":0.92}}`))
	}))
	defer server.Close()

	rate, err := fetchUsdToCnyFromSource(server.URL)
	require.NoError(t, err)
	assert.InDelta(t, 7.24, rate, 0.0001)
}

func TestFetchUsdToCnyFromSourceRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{name: "http error", status: http.StatusServiceUnavailable, body: `{}`, wantErr: true},
		{name: "malformed json", status: http.StatusOK, body: `not-json`, wantErr: true},
		{name: "missing cny rate", status: http.StatusOK, body: `{"rates":{"EUR":0.92}}`, wantErr: true},
		{name: "non-positive rate", status: http.StatusOK, body: `{"rates":{"CNY":0}}`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			_, err := fetchUsdToCnyFromSource(server.URL)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFetchLatestUsdToCnyRateFallsBackToNextSource(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	working := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"rates":{"CNY":7.31}}`))
	}))
	defer working.Close()

	previousSources := exchangeRateSourceURLs
	t.Cleanup(func() { exchangeRateSourceURLs = previousSources })
	exchangeRateSourceURLs = []string{failing.URL, working.URL}

	rate, source, err := fetchLatestUsdToCnyRate()
	require.NoError(t, err)
	assert.InDelta(t, 7.31, rate, 0.0001)
	assert.Equal(t, working.URL, source)
}

func TestFetchLatestUsdToCnyRateFailsWhenAllSourcesDown(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()

	previousSources := exchangeRateSourceURLs
	t.Cleanup(func() { exchangeRateSourceURLs = previousSources })
	exchangeRateSourceURLs = []string{failing.URL}

	_, _, err := fetchLatestUsdToCnyRate()
	require.Error(t, err)
}
