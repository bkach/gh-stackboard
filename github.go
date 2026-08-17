package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const pullRequestsQuery = `
query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    nameWithOwner
    defaultBranchRef { name }
    pullRequests(states: OPEN, first: 50, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number title url isDraft updatedAt headRefName baseRefName additions deletions
        author { login }
        headRepository { nameWithOwner }
        baseRepository { nameWithOwner }
        comments { totalCount }
        reviewDecision
        reviewRequests(first: 20) {
          nodes {
            requestedReviewer {
              __typename
              ... on User { login }
              ... on Team { slug organization { login } }
            }
          }
        }
        commits(last: 1) {
          nodes { commit { statusCheckRollup { state } } }
        }
      }
    }
  }
}`

type graphQLResponse struct {
	Data struct {
		Repository struct {
			NameWithOwner    string `json:"nameWithOwner"`
			DefaultBranchRef struct {
				Name string `json:"name"`
			} `json:"defaultBranchRef"`
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []graphQLPullRequest `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}

type graphQLPullRequest struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	IsDraft     bool      `json:"isDraft"`
	UpdatedAt   time.Time `json:"updatedAt"`
	HeadRefName string    `json:"headRefName"`
	BaseRefName string    `json:"baseRefName"`
	Additions   int       `json:"additions"`
	Deletions   int       `json:"deletions"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	HeadRepository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"headRepository"`
	BaseRepository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"baseRepository"`
	Comments struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	ReviewDecision string `json:"reviewDecision"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer struct {
				TypeName     string `json:"__typename"`
				Login        string `json:"login"`
				Slug         string `json:"slug"`
				Organization struct {
					Login string `json:"login"`
				} `json:"organization"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func loadGitHub(repositories []string) (apiResponse, error) {
	login, err := ghOutput("api", "user", "--jq", ".login")
	if err != nil {
		return apiResponse{}, fmt.Errorf("GitHub authentication failed; run `gh auth login`: %w", err)
	}
	login = strings.TrimSpace(login)

	if len(repositories) == 0 {
		repository, inferErr := ghOutput("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
		if inferErr != nil {
			return apiResponse{}, fmt.Errorf("could not infer a repository; pass `--repo OWNER/REPO`: %w", inferErr)
		}
		repositories = []string{strings.TrimSpace(repository)}
	}

	var normalized []string
	seen := map[string]bool{}
	for _, repository := range repositories {
		repository = strings.TrimSpace(repository)
		owner, name, valid := strings.Cut(repository, "/")
		if !valid || owner == "" || name == "" || strings.Contains(name, "/") {
			return apiResponse{}, fmt.Errorf("invalid repository %q; expected OWNER/REPO", repository)
		}
		key := strings.ToLower(repository)
		if !seen[key] {
			seen[key] = true
			normalized = append(normalized, owner+"/"+name)
		}
	}

	teams, teamWarning := fetchViewerTeams()
	type repositoryResult struct {
		requested, repository, defaultBranch string
		prs                                  []pullRequest
		err                                  error
	}
	results := make(chan repositoryResult, len(normalized))
	concurrency := make(chan struct{}, 4)
	for _, repository := range normalized {
		go func() {
			concurrency <- struct{}{}
			defer func() { <-concurrency }()
			owner, name, _ := strings.Cut(repository, "/")
			prs, canonical, defaultBranch, fetchErr := fetchPullRequests(owner, name)
			results <- repositoryResult{requested: repository, repository: canonical, defaultBranch: defaultBranch, prs: prs, err: fetchErr}
		}()
	}

	stacks := []stack{}
	var loadedRepositories, warnings []string
	if teamWarning != "" {
		warnings = append(warnings, teamWarning)
	}
	for range normalized {
		result := <-results
		if result.err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", result.requested, result.err))
			continue
		}
		loadedRepositories = append(loadedRepositories, result.repository)
		stacks = append(stacks, buildStacks(result.repository, result.defaultBranch, login, teams, result.prs)...)
	}
	if len(loadedRepositories) == 0 {
		return apiResponse{}, fmt.Errorf("could not load any repositories: %s", strings.Join(warnings, "; "))
	}
	sort.Slice(stacks, func(i, j int) bool { return stacks[i].UpdatedAt.After(stacks[j].UpdatedAt) })
	sort.Strings(loadedRepositories)
	repositoryLabel := loadedRepositories[0]
	if len(loadedRepositories) > 1 {
		repositoryLabel = fmt.Sprintf("%d repositories", len(loadedRepositories))
	}

	return apiResponse{
		Viewer: viewer{Login: login, Name: login}, Repository: repositoryLabel,
		RefreshedAt: time.Now().Format(time.RFC3339), Source: "github", Warning: strings.Join(warnings, " "), Stacks: stacks,
	}, nil
}

func fetchPullRequests(owner, name string) ([]pullRequest, string, string, error) {
	var all []pullRequest
	var cursor, repository, defaultBranch string
	for {
		args := []string{"api", "graphql", "-f", "query=" + pullRequestsQuery, "-f", "owner=" + owner, "-f", "name=" + name}
		if cursor != "" {
			args = append(args, "-f", "cursor="+cursor)
		}
		output, err := ghOutput(args...)
		if err != nil {
			return nil, "", "", fmt.Errorf("fetch pull requests for %s/%s: %w", owner, name, err)
		}
		var response graphQLResponse
		if err := json.Unmarshal([]byte(output), &response); err != nil {
			return nil, "", "", fmt.Errorf("decode GitHub response: %w", err)
		}
		repo := response.Data.Repository
		if repo.NameWithOwner == "" {
			return nil, "", "", fmt.Errorf("repository %s/%s was not found or is not visible", owner, name)
		}
		repository = repo.NameWithOwner
		defaultBranch = repo.DefaultBranchRef.Name
		for _, item := range repo.PullRequests.Nodes {
			all = append(all, convertPullRequest(item, defaultBranch))
		}
		if !repo.PullRequests.PageInfo.HasNextPage {
			break
		}
		cursor = repo.PullRequests.PageInfo.EndCursor
	}
	return all, repository, defaultBranch, nil
}

func convertPullRequest(item graphQLPullRequest, defaultBranch string) pullRequest {
	review := "waiting"
	if item.IsDraft {
		review = "draft"
	} else if item.ReviewDecision == "APPROVED" {
		review = "approved"
	} else if item.ReviewDecision == "CHANGES_REQUESTED" {
		review = "changes"
	}
	checks := "running"
	if len(item.Commits.Nodes) > 0 && item.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
		switch item.Commits.Nodes[0].Commit.StatusCheckRollup.State {
		case "SUCCESS":
			checks = "passing"
		case "FAILURE", "ERROR":
			checks = "failing"
		}
	}
	var users, teams []string
	for _, request := range item.ReviewRequests.Nodes {
		reviewer := request.RequestedReviewer
		if reviewer.TypeName == "User" {
			users = append(users, reviewer.Login)
		} else if reviewer.TypeName == "Team" {
			teams = append(teams, strings.ToLower(reviewer.Organization.Login+"/"+reviewer.Slug))
		}
	}
	baseRepository := item.BaseRepository.NameWithOwner
	if baseRepository == "" {
		baseRepository = item.HeadRepository.NameWithOwner
	}
	return pullRequest{
		Number: item.Number, Title: item.Title, URL: item.URL, Branch: item.HeadRefName, Author: item.Author.Login,
		Review: review, Checks: checks, Comments: item.Comments.TotalCount, Additions: item.Additions, Deletions: item.Deletions,
		Updated: relativeTime(item.UpdatedAt), MergeTarget: item.BaseRefName, HeadRepository: item.HeadRepository.NameWithOwner,
		BaseRepository: baseRepository, RequestedUsers: users, RequestedTeams: teams, UpdatedAt: item.UpdatedAt,
	}
}

func fetchViewerTeams() (map[string]bool, string) {
	output, err := ghOutput("api", "user/teams", "--paginate", "--jq", `.[] | "\(.organization.login)/\(.slug)"`)
	if err != nil {
		return map[string]bool{}, "Team membership is unavailable; the Team filter may be empty."
	}
	teams := map[string]bool{}
	for _, team := range strings.Fields(output) {
		teams[strings.ToLower(team)] = true
	}
	return teams, ""
}

func ghOutput(args ...string) (string, error) {
	command := exec.Command("gh", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

func buildStacks(repository, defaultBranch, viewerLogin string, viewerTeams map[string]bool, prs []pullRequest) []stack {
	byHead := map[string]int{}
	for index, pr := range prs {
		byHead[branchKey(pr.HeadRepository, pr.Branch)] = index
	}
	children := make(map[int][]int)
	parent := make(map[int]int)
	for index, pr := range prs {
		if parentIndex, ok := byHead[branchKey(pr.BaseRepository, pr.MergeTarget)]; ok && parentIndex != index {
			parent[index] = parentIndex
			children[parentIndex] = append(children[parentIndex], index)
		}
	}
	for key := range children {
		sort.Slice(children[key], func(i, j int) bool { return prs[children[key][i]].Number < prs[children[key][j]].Number })
	}

	visited := map[int]bool{}
	var result []stack
	var appendComponent func(int)
	appendComponent = func(root int) {
		var ordered []pullRequest
		var walk func(int)
		walk = func(index int) {
			if visited[index] {
				return
			}
			visited[index] = true
			pr := prs[index]
			if parentIndex, ok := parent[index]; ok {
				pr.MergeTarget = fmt.Sprintf("#%d", prs[parentIndex].Number)
			} else if pr.MergeTarget == "" {
				pr.MergeTarget = defaultBranch
			}
			ordered = append(ordered, pr)
			for _, child := range children[index] {
				walk(child)
			}
		}
		walk(root)
		if len(ordered) > 0 {
			result = append(result, makeStack(repository, viewerLogin, viewerTeams, ordered))
		}
	}
	for index := range prs {
		if _, hasParent := parent[index]; !hasParent {
			appendComponent(index)
		}
	}
	for index := range prs {
		if !visited[index] {
			appendComponent(index)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}

func makeStack(repository, viewerLogin string, viewerTeams map[string]bool, prs []pullRequest) stack {
	top, root := prs[len(prs)-1], prs[0]
	mine, assigned, team := false, false, false
	for _, pr := range prs {
		mine = mine || strings.EqualFold(pr.Author, viewerLogin)
		for _, user := range pr.RequestedUsers {
			assigned = assigned || strings.EqualFold(user, viewerLogin)
		}
		for _, requestedTeam := range pr.RequestedTeams {
			team = team || viewerTeams[strings.ToLower(requestedTeam)]
		}
	}
	return stack{
		ID: fmt.Sprintf("%s-%d", strings.ReplaceAll(repository, "/", "-"), root.Number), Title: top.Title,
		Repository: repository, Owner: root.Author, Initials: initials(root.Author), Mine: mine, Assigned: assigned,
		Team: team, Updated: top.Updated, PRs: prs, UpdatedAt: top.UpdatedAt,
	}
}

func branchKey(repository, branch string) string { return strings.ToLower(repository + ":" + branch) }

func initials(login string) string {
	runes := []rune(strings.ToUpper(strings.TrimSpace(login)))
	if len(runes) == 0 {
		return "?"
	}
	if len(runes) == 1 {
		return string(runes)
	}
	return string(runes[:2])
}

func relativeTime(value time.Time) string {
	duration := time.Since(value)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
}
