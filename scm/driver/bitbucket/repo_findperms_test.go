// Copyright 2026 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitbucket

import (
	"context"
	"testing"

	"github.com/h2non/gock"
)

// TestFindPerms_ErrorWhenAllWorkspaces404 tests the case where repo doesn't exist in any workspace
func TestFindPerms_ErrorWhenAllWorkspaces404(t *testing.T) {
	defer gock.Off()

	// Mock workspace list
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws1"}}, {"workspace": {"slug": "ws2"}}], "next": ""}`)

	// ws1: admin check → empty, contributor check → empty, direct fetch → 404
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/nonexistent-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/nonexistent-repo"`).
		MatchParam("role", "contributor").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1/nonexistent-repo").
		Reply(404).
		Type("application/json").
		BodyString(`{"error": {"message": "Repository not found"}}`)

	// ws2: same pattern
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws2").
		MatchParam("q", `full_name="ws2/nonexistent-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws2").
		MatchParam("q", `full_name="ws2/nonexistent-repo"`).
		MatchParam("role", "contributor").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws2/nonexistent-repo").
		Reply(404).
		Type("application/json").
		BodyString(`{"error": {"message": "Repository not found"}}`)

	client, _ := New("https://api.bitbucket.org")
	_, _, err := client.Repositories.FindPerms(context.Background(), "nonexistent-repo")

	if err == nil {
		t.Fatal("Expected error when repository not found in any workspace")
	}

	if err.Error() != "repository nonexistent-repo not found in any workspace" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestFindPerms_ErrorWhenWorkspaceFetchFails tests error handling when workspace fetch fails
func TestFindPerms_ErrorWhenWorkspaceFetchFails(t *testing.T) {
	defer gock.Off()

	// Mock workspace list with error
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(500).
		Type("application/json").
		BodyString(`{"error": {"message": "Internal server error"}}`)

	client, _ := New("https://api.bitbucket.org")
	_, _, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	if err == nil {
		t.Fatal("Expected error when workspace fetch fails")
	}
}

// TestFindPerms_NoPermissionsInAnyWorkspace tests when user has no access to the repo
func TestFindPerms_NoPermissionsInAnyWorkspace(t *testing.T) {
	defer gock.Off()

	// Mock workspace list
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws1"}}], "next": ""}`)

	// ws1: admin check → empty, contributor check → empty, direct fetch → 404 (no access)
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "contributor").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1/test-repo").
		Reply(404).
		Type("application/json").
		BodyString(`{"error": {"message": "Repository not found"}}`)

	client, _ := New("https://api.bitbucket.org")
	_, _, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	if err == nil {
		t.Fatal("Expected error when user has no access to repository")
	}

	if err.Error() != "repository test-repo not found in any workspace" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestFindPerms_NetworkErrorReturnsImmediately tests partial network failure
func TestFindPerms_NetworkErrorReturnsImmediately(t *testing.T) {
	defer gock.Off()

	// Mock workspace list
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws1"}}, {"workspace": {"slug": "ws2"}}], "next": ""}`)

	// ws1 admin check returns 500 (non-404 error) → return immediately
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "admin").
		Reply(500).
		Type("application/json").
		BodyString(`{"error": {"message": "Internal server error"}}`)

	client, _ := New("https://api.bitbucket.org")
	_, res, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	// Should return error immediately, not continue to ws2
	if err == nil {
		t.Fatal("Expected error for non-404 failure")
	}

	if res == nil || res.Status != 500 {
		t.Error("Expected response with 500 status")
	}
}

// TestFindPerms_Continues404ToSuccess tests that iteration continues on 404
func TestFindPerms_Continues404ToSuccess(t *testing.T) {
	defer gock.Off()

	// Mock workspace list
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws1"}}, {"workspace": {"slug": "ws2"}}], "next": ""}`)

	// ws1: admin → empty, contributor → empty, direct fetch → 404
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "contributor").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1/test-repo").
		Reply(404).
		Type("application/json").
		BodyString(`{"error": {"message": "Not found"}}`)

	// ws2: admin check returns repo → user is admin
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws2").
		MatchParam("q", `full_name="ws2/test-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{}]}`)

	client, _ := New("https://api.bitbucket.org")
	perm, _, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !perm.Admin || !perm.Push || !perm.Pull {
		t.Error("Expected admin permissions from ws2")
	}
}

// TestFindPerms_EmptyWorkspaceList tests when user has no workspaces
func TestFindPerms_EmptyWorkspaceList(t *testing.T) {
	defer gock.Off()

	// Mock empty workspace list
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [], "next": ""}`)

	client, _ := New("https://api.bitbucket.org")
	_, _, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	if err == nil {
		t.Fatal("Expected error when no workspaces available")
	}

	if err.Error() != "repository test-repo not found in any workspace" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestFindPerms_WorkspaceInURLExtractionWithSlash tests that URL workspace is used
// even when repo has workspace/repo format - it extracts just the repo slug
func TestFindPerms_WorkspaceInURLExtractionWithSlash(t *testing.T) {
	defer gock.Off()

	// When workspace is in URL and repo has workspace/repo format,
	// it should use URL workspace and extract just the repo slug
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/url-workspace").
		MatchParam("q", `full_name="url-workspace/actual-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{}]}`)

	client, _ := New("https://api.bitbucket.org/repositories/url-workspace")
	perm, _, err := client.Repositories.FindPerms(context.Background(), "different-workspace/actual-repo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !perm.Admin {
		t.Error("Expected admin permissions")
	}

	// Verify the correct endpoint was called (url-workspace, not different-workspace)
	if !gock.IsDone() {
		pending := gock.Pending()
		if len(pending) > 0 {
			t.Errorf("Expected all mocks to be called, pending: %v", pending[0].Request().URLStruct)
		}
	}
}

// TestFindPerms_PriorityOfWorkspaceResolution tests that URL workspace takes priority
func TestFindPerms_URLWorkspaceTakesPriority(t *testing.T) {
	defer gock.Off()

	// URL workspace should take priority over repo identifier workspace
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/url-workspace").
		MatchParam("q", `full_name="url-workspace/repo-slug"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{}]}`)

	// Should NOT call identifier-workspace even though repo is "identifier-workspace/repo-slug"
	client, _ := New("https://api.bitbucket.org/repositories/url-workspace")
	perm, _, err := client.Repositories.FindPerms(context.Background(), "identifier-workspace/repo-slug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !perm.Admin {
		t.Error("Expected admin permissions from url-workspace")
	}

	// Verify the correct endpoint was called (url-workspace, not identifier-workspace)
	if !gock.IsDone() {
		pending := gock.Pending()
		if len(pending) > 0 {
			t.Errorf("Expected all mocks to be called, pending: %v", pending[0].Request().URLStruct)
		}
	}
}

// TestFindPerms_MultipleWorkspacesMultiplePages tests pagination in workspace fetching
func TestFindPerms_WorkspacePagination(t *testing.T) {
	defer gock.Off()

	// Mock workspace list with multiple pages
	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "1").
		MatchParam("pagelen", "100").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws1"}}], "next": "https://api.bitbucket.org/2.0/user/workspaces?page=2"}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/user/workspaces").
		MatchParam("page", "2").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{"workspace": {"slug": "ws2"}}], "next": ""}`)

	// ws1: admin → empty, contributor → empty, direct fetch → 404
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1").
		MatchParam("q", `full_name="ws1/test-repo"`).
		MatchParam("role", "contributor").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": []}`)

	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws1/test-repo").
		Reply(404).
		Type("application/json").
		BodyString(`{"error": {"message": "Not found"}}`)

	// ws2 (from page 2): admin check returns repo
	gock.New("https://api.bitbucket.org").
		Get("/2.0/repositories/ws2").
		MatchParam("q", `full_name="ws2/test-repo"`).
		MatchParam("role", "admin").
		Reply(200).
		Type("application/json").
		BodyString(`{"values": [{}]}`)

	client, _ := New("https://api.bitbucket.org")
	perm, _, err := client.Repositories.FindPerms(context.Background(), "test-repo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !perm.Admin {
		t.Error("Expected to find admin permissions from workspace on page 2")
	}
}
