package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const maxStorageResponseBytes = 32 << 10

type SupabaseSigner struct {
	storageURL string
	secretKey  string
	bucket     string
	client     *http.Client
}

func NewSupabaseSigner(baseURL, secretKey, bucket string, client *http.Client) (*SupabaseSigner, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" {
		return nil, errors.New("SUPABASE_URL must be a valid URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return nil, errors.New("SUPABASE_URL must use HTTPS")
	}
	if strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("SUPABASE_SECRET_KEY is required")
	}
	if bucket == "" || strings.ContainsAny(bucket, "/\\") {
		return nil, errors.New("SUPABASE_STORAGE_BUCKET is invalid")
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &SupabaseSigner{storageURL: strings.TrimRight(parsed.String(), "/") + "/storage/v1", secretKey: strings.TrimSpace(secretKey), bucket: bucket, client: client}, nil
}

func (s *SupabaseSigner) CreateUploadURL(ctx context.Context, objectKey string) (string, error) {
	clean := path.Clean(strings.TrimSpace(objectKey))
	if clean == "." || clean != objectKey || strings.HasPrefix(clean, "/") || !strings.HasSuffix(strings.ToLower(clean), ".pdf") || strings.ContainsAny(clean, "\\ \t\r\n") {
		return "", errors.New("unsafe storage object key")
	}
	segments := strings.Split(clean, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	endpoint := s.storageURL + "/object/upload/sign/" + url.PathEscape(s.bucket) + "/" + strings.Join(segments, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("{}"))
	if err != nil {
		return "", fmt.Errorf("create storage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("apikey", s.secretKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("storage request failed with status %d", response.StatusCode)
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxStorageResponseBytes)).Decode(&payload); err != nil || !strings.HasPrefix(payload.URL, "/object/upload/sign/") {
		return "", errors.New("storage returned an invalid signed URL")
	}
	return s.storageURL + payload.URL, nil
}

func isLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}
