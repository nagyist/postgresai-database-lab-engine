/*
2019 © Postgres.ai
*/

package mw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Logging(next)

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "next handler should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLogging_DifferentMethods(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "get request", method: http.MethodGet, path: "/api/v1/status"},
		{name: "post request", method: http.MethodPost, path: "/api/v1/clone"},
		{name: "delete request", method: http.MethodDelete, path: "/api/v1/clone/test"},
		{name: "put request", method: http.MethodPut, path: "/api/v1/clone/test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.True(t, called)
		})
	}
}
