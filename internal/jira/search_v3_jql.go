package jira

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

// jqlSearchV3Response is the body of GET /rest/api/3/search/jql
// (issues-by-JQL, cursor-based). See:
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/
type jqlSearchV3Response struct {
	IsLast         bool            `json:"isLast"`
	Issues         []gojira.Issue  `json:"issues"`
	NextPageToken  string          `json:"nextPageToken"`
	// some tenants may still include this (legacy compatibility)
	Total int `json:"total,omitempty"`
}

// SearchIssuesJQL runs a JQL search using GET /rest/api/3/search/jql.
// This is the Jira Cloud replacement for the removed / rest/api/2/search
// and / rest/api/3/search endpoints used by the go-jira Issue.Search client.
func (c *Client) SearchIssuesJQL(ctx context.Context, jql string, maxResults int, fields []string) ([]gojira.Issue, error) {
	if jql == "" {
		return nil, fmt.Errorf("jql is required")
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	var out []gojira.Issue
	var next string
	for len(out) < maxResults {
		remaining := maxResults - len(out)
		// Jira may cap a single page; request at most `remaining` issues.
		pageSize := remaining
		if pageSize > 100 {
			pageSize = 100
		}

		v := url.Values{}
		v.Set("jql", jql)
		v.Set("maxResults", strconv.Itoa(pageSize))
		if len(fields) > 0 {
			v.Set("fields", strings.Join(fields, ","))
		}
		if next != "" {
			v.Set("nextPageToken", next)
		}

		path := "rest/api/3/search/jql?" + v.Encode()
		req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var body jqlSearchV3Response
		_, err = c.Do(req, &body)
		if err != nil {
			return nil, err
		}
		out = append(out, body.Issues...)

		if body.IsLast || body.NextPageToken == "" {
			break
		}
		next = body.NextPageToken
	}
	return out, nil
}
