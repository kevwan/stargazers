package gh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v33/github"
	"github.com/tal-tech/go-zero/core/logx"
	"golang.org/x/oauth2"
)

func CreateClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}

func ParseRepo(repo string) (owner, project string, err error) {
	words := strings.Split(repo, "/")
	if len(words) != 2 {
		err = errors.New("repo should be <owner>/<project> format")
		return
	}

	owner = words[0]
	project = words[1]
	return
}

func RequestAll(cli *github.Client, owner, project string) (map[string]time.Time, error) {
	stars := make(map[string]time.Time)
	var page = 1
	for {
		logx.Infof("requesting page %d", page)
		gazers, resp, err := cli.Activity.ListStargazers(context.Background(),
			owner, project, &github.ListOptions{
				Page:    page,
				PerPage: pageSize,
			})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stargazers, error: %v", err)
		}

		for _, gazer := range gazers {
			id := *gazer.User.Login
			if _, ok := stars[id]; !ok {
				stars[id] = gazer.StarredAt.Time
			}
		}

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return stars, nil
}

func RequestUser(cli *github.Client, id string) (*github.User, error) {
	user, _, err := cli.Users.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return user, nil
}
