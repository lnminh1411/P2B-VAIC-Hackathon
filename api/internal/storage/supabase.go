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
	segments, err := safeObjectSegments(objectKey)
	if err != nil {
		return "", err
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

func (s *SupabaseSigner) Download(ctx context.Context, objectKey string, maxBytes int64) ([]byte, error) {
	if maxBytes < 1 || maxBytes > 21<<20 {
		return nil, errors.New("invalid download size limit")
	}
	segments, err := safeObjectSegments(objectKey)
	if err != nil {
		return nil, err
	}
	endpoint := s.storageURL + "/object/authenticated/" + url.PathEscape(s.bucket) + "/" + strings.Join(segments, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+s.secretKey)
	request.Header.Set("apikey", s.secretKey)
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("storage download: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("storage download failed with status %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read storage object: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return nil, errors.New("storage object exceeds declared size limit")
	}
	return content, nil
}

func safeObjectSegments(objectKey string) ([]string, error) {
	clean := path.Clean(strings.TrimSpace(objectKey))
	if clean == "." || clean != objectKey || strings.HasPrefix(clean, "/") || !strings.HasSuffix(strings.ToLower(clean), ".pdf") || strings.ContainsAny(clean, "\\ \t\r\n") {
		return nil, errors.New("unsafe storage object key")
	}
	segments := strings.Split(clean, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return segments, nil
}

func isLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}
