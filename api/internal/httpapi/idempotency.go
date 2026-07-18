package httpapi

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxInMemoryIdempotencyKeys = 10_000
	idempotencyRetention       = 24 * time.Hour
)

type idempotencyStore struct {
	mu      sync.Mutex
	records map[string]*idempotencyRecord
}

type idempotencyRecord struct {
	requestHash [32]byte
	status      int
	header      http.Header
	body        []byte
	ready       chan struct{}
	createdAt   time.Time
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{records: make(map[string]*idempotencyRecord)}
}

func (s *Server) idempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" || len(key) > 200 {
			writeError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "A valid Idempotency-Key header is required")
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds 1 MB")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		requestHash := sha256.Sum256(append([]byte(r.Method+"\x00"+r.URL.RequestURI()+"\x00"), body...))
		scope := workspace(r) + "\x00" + key

		record, replayed, ok := s.idempotency.acquire(scope, requestHash)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "IDEMPOTENCY_CAPACITY", "Too many retained mutation keys; retry later")
			return
		}
		if replayed {
			if record.requestHash != requestHash {
				writeError(w, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used for a different request")
				return
			}
			select {
			case <-record.ready:
				if record.status == 0 {
					writeError(w, http.StatusServiceUnavailable, "IDEMPOTENT_OPERATION_RETRY", "Original operation did not complete; retry the request")
					return
				}
				writeRecordedResponse(w, record.status, record.header, record.body)
			case <-r.Context().Done():
				writeError(w, http.StatusRequestTimeout, "REQUEST_CANCELLED", "Request was cancelled while waiting for the original operation")
			}
			return
		}

		captured := &bufferedResponse{header: make(http.Header)}
		completed := false
		defer func() {
			if !completed {
				s.idempotency.abort(scope, record)
			}
		}()
		next.ServeHTTP(captured, r)
		status := captured.status
		if status == 0 {
			status = http.StatusOK
		}
		s.idempotency.complete(scope, record, status, captured.header, captured.body.Bytes())
		completed = true
		writeRecordedResponse(w, status, captured.header, captured.body.Bytes())
	})
}

func (s *idempotencyStore) acquire(scope string, requestHash [32]byte) (*idempotencyRecord, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record, exists := s.records[scope]; exists {
		return record, true, true
	}
	if len(s.records) >= maxInMemoryIdempotencyKeys {
		cutoff := time.Now().Add(-idempotencyRetention)
		for storedScope, stored := range s.records {
			if stored.status != 0 && stored.createdAt.Before(cutoff) {
				delete(s.records, storedScope)
			}
		}
		if len(s.records) >= maxInMemoryIdempotencyKeys {
			return nil, false, false
		}
	}
	record := &idempotencyRecord{requestHash: requestHash, ready: make(chan struct{}), createdAt: time.Now()}
	s.records[scope] = record
	return record, false, true
}

func (s *idempotencyStore) complete(scope string, record *idempotencyRecord, status int, header http.Header, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status >= http.StatusInternalServerError {
		delete(s.records, scope)
	} else {
		record.status = status
		record.header = header.Clone()
		record.body = append([]byte(nil), body...)
	}
	close(record.ready)
}

func (s *idempotencyStore) abort(scope string, record *idempotencyRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records[scope] == record {
		delete(s.records, scope)
		close(record.ready)
	}
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *bufferedResponse) Header() http.Header { return w.header }
func (w *bufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *bufferedResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}

func writeRecordedResponse(w http.ResponseWriter, status int, header http.Header, body []byte) {
	for key, values := range header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
