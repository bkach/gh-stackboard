package main

import "testing"

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
