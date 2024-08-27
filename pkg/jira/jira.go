package jira

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRA_TOKEN_CONFIG_KEY = "jira_token"
	JIRA_TOKEN_ENV_KEY    = "JIRA_API_TOKEN"
)

type client struct {
	client *jira.Client
}

type Client interface {
	GetIssues() ([]*Issue, error)
}

type Issue struct {
	Status          string
	Priority        string
	LastUpdatedTime time.Time
}

func NewClient(jiraURL string) (*client, error) {
	if jiraURL == "" {
		return nil, fmt.Errorf("JIRA URL cannot be empty")
	}

	var jiratoken string
	if os.Getenv(JIRA_TOKEN_ENV_KEY) != "" {
		jiratoken = os.Getenv(JIRA_TOKEN_ENV_KEY)
	}

	if jiratoken == "" {
		return nil, fmt.Errorf("JIRA token is not defined.")
	}

	tp := jira.PATAuthTransport{
		Token: jiratoken,
	}
	jiraClient, err := jira.NewClient(tp.Client(), jiraURL)
	if err != nil {
		return nil, err
	}
	c := client{
		client: jiraClient,
	}
	return &c, nil
}

func (c *client) GetIssues() ([]*Issue, error) {
	var issues []*Issue
	jql := "filter = 12346875"

	searchOpts := &jira.SearchOptions{
		MaxResults: 1000,
	}

	issueList, _, err := c.client.Issue.Search(jql, searchOpts)

	if err != nil {
		return nil, err
	}

	for _, jiraIssue := range issueList {
		issue := &Issue{
			Priority:        strings.ToLower(jiraIssue.Fields.Priority.Name),
			Status:          strings.ToLower(jiraIssue.Fields.Status.Name),
			LastUpdatedTime: time.Time(jiraIssue.Fields.Updated),
		}
		issues = append(issues, issue)
	}

	return issues, nil
}
