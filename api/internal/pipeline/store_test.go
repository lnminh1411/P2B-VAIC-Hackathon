package pipeline

import (
	"testing"

	"github.com/google/uuid"
)

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

func TestValidateBuildRequestParsesIdentifiers(t *testing.T) {
	workspaceID := uuid.NewString()
	actorID := uuid.NewString()
	sourceID := uuid.NewString()

	identifiers, err := validateBuildRequest(workspaceID, BuildRequest{ActorSubject: actorID, SourceIDs: []string{sourceID}})
	if err != nil {
		t.Fatal(err)
	}
	if identifiers.workspace.String() != workspaceID || identifiers.actor.String() != actorID || len(identifiers.sources) != 1 || identifiers.sources[0].String() != sourceID {
		t.Fatalf("identifiers = %#v", identifiers)
	}
}

func TestValidateBuildRequestRejectsInvalidIdentifiers(t *testing.T) {
	validID := uuid.NewString()
	tests := []struct {
		name      string
		workspace string
		request   BuildRequest
		wantError string
	}{
		{name: "workspace", workspace: "invalid", request: BuildRequest{ActorSubject: validID}, wantError: "invalid workspace"},
		{name: "actor", workspace: validID, request: BuildRequest{ActorSubject: "invalid"}, wantError: "invalid actor"},
		{name: "source", workspace: validID, request: BuildRequest{ActorSubject: validID, SourceIDs: []string{"invalid"}}, wantError: "invalid source id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateBuildRequest(test.workspace, test.request)
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("error = %v, want %q", err, test.wantError)
			}
		})
	}
}
