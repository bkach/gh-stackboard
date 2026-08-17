package main

import (
	"testing"
	"time"
)

func TestMockStacksHaveOrderedPullRequests(t *testing.T) {
	if len(mockStacks) == 0 {
		t.Fatal("expected mock stacks")
	}
	for _, stack := range mockStacks {
		if len(stack.PRs) == 0 {
			t.Errorf("stack %q has no pull requests", stack.ID)
		}
		if stack.PRs[0].MergeTarget != "main" {
			t.Errorf("stack %q does not begin at main", stack.ID)
		}
	}
}

func TestBuildStacksFromBaseBranches(t *testing.T) {
	prs := []pullRequest{
		{Number: 101, Title: "Base change", Author: "alice", Branch: "feature-1", HeadRepository: "acme/app", BaseRepository: "acme/app", MergeTarget: "main", Review: "approved", Checks: "passing"},
		{Number: 102, Title: "Top change", Author: "alice", Branch: "feature-2", HeadRepository: "acme/app", BaseRepository: "acme/app", MergeTarget: "feature-1", RequestedUsers: []string{"bob"}},
	}

	stacks := buildStacks("acme/app", "main", "bob", map[string]bool{}, prs)
	if len(stacks) != 1 {
		t.Fatalf("expected one stack, got %d", len(stacks))
	}
	if len(stacks[0].PRs) != 2 || stacks[0].PRs[0].Number != 101 || stacks[0].PRs[1].Number != 102 {
		t.Fatalf("unexpected pull request order: %#v", stacks[0].PRs)
	}
	if !stacks[0].Assigned {
		t.Fatal("expected stack to be assigned to the viewer")
	}
	if stacks[0].Title != "Top change" {
		t.Fatalf("expected top pull request title, got %q", stacks[0].Title)
	}
}

func TestTeamAssignmentUsesViewerMembership(t *testing.T) {
	prs := []pullRequest{{
		Number: 1, Title: "Change", Author: "alice", Branch: "change", HeadRepository: "acme/app", BaseRepository: "acme/app", MergeTarget: "main",
		RequestedTeams: []string{"acme/platform"},
	}}
	stacks := buildStacks("acme/app", "main", "bob", map[string]bool{"acme/platform": true}, prs)
	if !stacks[0].Team {
		t.Fatal("expected stack to match the viewer's requested team")
	}
}

func TestMakeNativeStackKeepsMergedAncestors(t *testing.T) {
	mergedAt := time.Now().Add(-time.Hour)
	native := nativeStack{Number: 205}
	native.Base.Ref = "main"
	native.PullRequests = []nativeStackPullRequest{
		{Number: 201, Title: "Merged foundation", State: "closed", MergedAt: &mergedAt},
		{Number: 202, Title: "Open follow-up", State: "open"},
	}
	native.PullRequests[0].Head.Ref = "feature-1"
	native.PullRequests[0].User.Login = "alice"
	native.PullRequests[1].Head.Ref = "feature-2"
	native.PullRequests[1].User.Login = "alice"

	open := pullRequest{
		Number: 202, Title: "Open follow-up", Branch: "feature-2", Author: "alice", State: "open",
		Review: "approved", Checks: "passing", UpdatedAt: time.Now(), Updated: "just now",
	}
	result := makeNativeStack("acme/app", "main", "alice", map[string]bool{}, native, []pullRequest{open})

	if result.ID != "acme-app-stack-205" {
		t.Fatalf("unexpected native stack id: %q", result.ID)
	}
	if len(result.PRs) != 2 {
		t.Fatalf("expected both native entries, got %d", len(result.PRs))
	}
	if result.PRs[0].State != "merged" || result.PRs[0].Review != "merged" {
		t.Fatalf("expected merged ancestor, got %#v", result.PRs[0])
	}
	if result.PRs[1].MergeTarget != "#201" {
		t.Fatalf("expected durable native ordering, got target %q", result.PRs[1].MergeTarget)
	}
	if !result.Mine {
		t.Fatal("expected stack to belong to the viewer")
	}
}

func TestConvertPullRequestIncludesMergeQueueDetails(t *testing.T) {
	item := graphQLPullRequest{Number: 101, State: "OPEN", UpdatedAt: time.Now()}
	item.MergeQueueEntry = &graphQLMergeQueueEntry{
		Position: 3, State: "AWAITING_CHECKS", EstimatedTimeToMerge: 180,
	}

	pr := convertPullRequest(item, "main")
	if !pr.Queued || pr.QueuePosition != 3 || pr.QueueState != "awaiting_checks" || pr.QueueETA != 180 {
		t.Fatalf("unexpected merge queue details: %#v", pr)
	}
	if pr.State != "open" {
		t.Fatalf("queue membership should not replace PR state, got %q", pr.State)
	}
}
