package pipeline

import "testing"

func TestValidateRefreshRequestRequiresNewSources(t *testing.T) {
	if err := ValidateRefreshRequest(BuildRequest{}); err == nil {
		t.Fatal("expected refresh to require at least one source")
	}
	if err := ValidateRefreshRequest(BuildRequest{SourceIDs: []string{"one"}}); err != nil {
		t.Fatalf("valid refresh rejected: %v", err)
	}
}

func TestValidateRefreshRequestLimitsSourceCount(t *testing.T) {
	request := BuildRequest{SourceIDs: make([]string, 11)}
	if err := ValidateRefreshRequest(request); err == nil {
		t.Fatal("expected refresh source limit error")
	}
}
