package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxUserResponseBytes = 64 << 10

var (
	ErrInvalidToken        = errors.New("invalid access token")
	ErrVerifierUnavailable = errors.New("identity verifier unavailable")
)

type Principal struct {
	Subject   string   `json:"subject"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

func (p Principal) HasRole(role string) bool {
	for _, candidate := range p.Roles {
		if strings.EqualFold(candidate, role) {
			return true
		}
	}
	return false
}

type Verifier interface {
	Verify(context.Context, string) (Principal, error)
}

type SupabaseVerifier struct {
	userEndpoint   string
	publishableKey string
	client         *http.Client
}

func NewSupabaseVerifier(baseURL, publishableKey string, client *http.Client) (*SupabaseVerifier, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" {
		return nil, errors.New("SUPABASE_URL must be a valid URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return nil, errors.New("SUPABASE_URL must use HTTPS")
	}
	if strings.TrimSpace(publishableKey) == "" {
		return nil, errors.New("SUPABASE_PUBLISHABLE_KEY is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &SupabaseVerifier{
		userEndpoint:   strings.TrimRight(parsed.String(), "/") + "/auth/v1/user",
		publishableKey: strings.TrimSpace(publishableKey),
		client:         client,
	}, nil
}

func (v *SupabaseVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	if token == "" || len(token) > 8192 {
		return Principal{}, ErrInvalidToken
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.userEndpoint, nil)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: create request", ErrVerifierUnavailable)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", v.publishableKey)
	req.Header.Set("Accept", "application/json")
	response, err := v.client.Do(req)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %v", ErrVerifierUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return Principal{}, ErrInvalidToken
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Principal{}, fmt.Errorf("%w: status %d", ErrVerifierUnavailable, response.StatusCode)
	}

	var payload struct {
		ID           string         `json:"id"`
		Email        string         `json:"email"`
		AppMetadata  map[string]any `json:"app_metadata"`
		UserMetadata map[string]any `json:"user_metadata"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxUserResponseBytes))
	if err := decoder.Decode(&payload); err != nil {
		return Principal{}, ErrInvalidToken
	}
	if _, err := uuid.Parse(payload.ID); err != nil || strings.TrimSpace(payload.Email) == "" {
		return Principal{}, ErrInvalidToken
	}
	return Principal{
		Subject:   payload.ID,
		Email:     strings.TrimSpace(payload.Email),
		Name:      metadataString(payload.UserMetadata, "full_name", "name"),
		AvatarURL: metadataString(payload.UserMetadata, "avatar_url", "picture"),
		Roles:     metadataRoles(payload.AppMetadata),
	}, nil
}

func metadataRoles(metadata map[string]any) []string {
	seen := map[string]struct{}{}
	roles := make([]string, 0, 2)
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || len(value) > 64 {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		roles = append(roles, value)
	}
	if role, ok := metadata["role"].(string); ok {
		add(role)
	}
	if values, ok := metadata["roles"].([]any); ok {
		for _, value := range values {
			if role, stringValue := value.(string); stringValue {
				add(role)
			}
		}
	}
	return roles
}

func metadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}
