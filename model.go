package main

import "time"

type viewer struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type pullRequest struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	URL           string `json:"url,omitempty"`
	Branch        string `json:"branch"`
	Author        string `json:"author"`
	Review        string `json:"review"`
	Checks        string `json:"checks"`
	Comments      int    `json:"comments"`
	Additions     int    `json:"additions"`
	Deletions     int    `json:"deletions"`
	Updated       string `json:"updated"`
	MergeTarget   string `json:"mergeTarget"`
	State         string `json:"state"`
	Queued        bool   `json:"queued,omitempty"`
	QueuePosition int    `json:"queuePosition,omitempty"`
	QueueState    string `json:"queueState,omitempty"`
	QueueETA      int    `json:"queueEtaSeconds,omitempty"`

	HeadRepository string    `json:"-"`
	BaseRepository string    `json:"-"`
	RequestedUsers []string  `json:"-"`
	RequestedTeams []string  `json:"-"`
	UpdatedAt      time.Time `json:"-"`
	StackID        string    `json:"-"`
	StackNumber    int       `json:"-"`
}

type stack struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Repository string        `json:"repository"`
	Owner      string        `json:"owner"`
	Initials   string        `json:"initials"`
	Mine       bool          `json:"mine"`
	Assigned   bool          `json:"assigned"`
	Team       bool          `json:"team"`
	Updated    string        `json:"updated"`
	PRs        []pullRequest `json:"prs"`
	UpdatedAt  time.Time     `json:"-"`
}
