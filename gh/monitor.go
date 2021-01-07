package gh

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v33/github"
	"github.com/tal-tech/go-zero/core/logx"
)

const (
	pageSize     = 100
	timeInterval = time.Minute
	dayFormat    = "2006 01-02"
	starAtFormat = "01-02 15:04:05"
)

var (
	stargazers = make(map[string]time.Time)
	dayStars   = make(map[string]int)
	startTime  = time.Now()
)

type Monitor struct {
	repo  string
	token string
	send  func(string) error
}

func NewMonitor(repo, token string, send func(text string) error) Monitor {
	return Monitor{
		repo:  repo,
		token: token,
		send:  send,
	}
}

func (m Monitor) Start() error {
	owner, project, err := ParseRepo(m.repo)
	if err != nil {
		return err
	}

	cli := CreateClient(m.token)
	if stars, err := RequestAll(cli, owner, project); err != nil {
		return err
	} else {
		stargazers = stars
	}

	ticker := time.NewTicker(timeInterval)
	defer ticker.Stop()
	for range ticker.C {
		logx.Info("requesting update")
		count, err := m.totalCount(cli, owner, project)
		if err != nil {
			if err := m.send(err.Error()); err != nil {
				logx.Error(err)
			}
			continue
		}

		logx.Infof("stars: %d", count)
		err = m.requestPage(cli, owner, project, count, count/pageSize+1)
		if err != nil {
			if err := m.send(err.Error()); err != nil {
				logx.Error(err)
			}
		}
	}

	return nil
}

func (m Monitor) beginOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func (m Monitor) countsToday(total int) int {
	yesterday := time.Now().Add(-time.Hour * 24).Format(dayFormat)
	if stars, ok := dayStars[yesterday]; ok {
		return total - stars
	}

	var count int
	bod := m.beginOfDay(time.Now())
	for _, t := range stargazers {
		if t.After(bod) {
			count++
		}
	}

	return count
}

func (m Monitor) reportStarring(cli *github.Client, owner, project string, total int, gazer *github.Stargazer) {
	if gazer.StarredAt.Time.Before(startTime) {
		return
	}

	name, followers, err := m.requestNameFollowers(cli, *gazer.User.Login)
	if err != nil {
		if err := m.send(err.Error()); err != nil {
			logx.Error(err)
		}
		return
	}

	// refresh count, because users might star after fetching count
	if count, err := m.totalCount(cli, owner, project); err == nil {
		total = count
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "stars: %d\n", total)
	fmt.Fprintf(&builder, "today: %d\n", m.countsToday(total))
	fmt.Fprintf(&builder, "user: %s\n", *gazer.User.Login)
	if len(name) > 0 {
		fmt.Fprintf(&builder, "name: %s\n", name)
	}
	if followers > 0 {
		fmt.Fprintf(&builder, "followers: %d\n", followers)
	}
	fmt.Fprintf(&builder, "time: %s", gazer.StarredAt.Time.Local().Format(starAtFormat))
	if err := m.send(builder.String()); err != nil {
		logx.Error(err)
	}
}

func (m Monitor) requestPage(cli *github.Client, owner, project string, count, page int) error {
	gazers, resp, err := cli.Activity.ListStargazers(context.Background(),
		owner, project, &github.ListOptions{
			Page:    page,
			PerPage: pageSize,
		})
	if err != nil {
		return fmt.Errorf("failed to fetch stargazers, error: %v", err)
	}

	for _, gazer := range gazers {
		id := *gazer.User.Login
		if _, ok := stargazers[id]; ok {
			continue
		}

		stargazers[id] = gazer.StarredAt.Time
		m.reportStarring(cli, owner, project, count, gazer)
	}

	if len(gazers) > 0 && gazers[0].StarredAt.Time.After(m.beginOfDay(time.Now())) {
		return m.requestPage(cli, owner, project, count, resp.PrevPage)
	}

	return nil
}

func (m Monitor) requestNameFollowers(cli *github.Client, id string) (name string, followers int, err error) {
	var user *github.User
	user, err = RequestUser(cli, id)
	if err != nil {
		if err := m.send(err.Error()); err != nil {
			logx.Error(err)
		}
		return
	}

	if user != nil {
		if user.Name != nil {
			name = *user.Name
		}
		if user.Followers != nil {
			followers = *user.Followers
		}
	}

	return
}

func (m Monitor) totalCount(cli *github.Client, owner, project string) (int, error) {
	repo, _, err := cli.Repositories.Get(context.Background(), owner, project)
	if err != nil {
		return 0, err
	}

	day := time.Now().Format(dayFormat)
	prev := dayStars[day]
	if *repo.StargazersCount < prev {
		stars, err := RequestAll(cli, owner, project)
		if err != nil {
			return 0, err
		}

		for k, v := range stargazers {
			if _, ok := stars[k]; ok {
				continue
			}

			name, followers, err := m.requestNameFollowers(cli, k)
			if err != nil {
				if err := m.send(err.Error()); err != nil {
					logx.Error(err)
				}
				continue
			}

			var builder strings.Builder
			fmt.Fprintln(&builder, "unstar")
			fmt.Fprintf(&builder, "stars: %d\n", *repo.StargazersCount)
			fmt.Fprintf(&builder, "today: %d\n", m.countsToday(*repo.StargazersCount))
			fmt.Fprintf(&builder, "user: %s\n", k)
			if len(name) > 0 {
				fmt.Fprintf(&builder, "name: %s\n", name)
			}
			if followers > 0 {
				fmt.Fprintf(&builder, "followers: %d\n", followers)
			}
			fmt.Fprintf(&builder, "starAt: %s", v.Format(starAtFormat))
			if err := m.send(builder.String()); err != nil {
				logx.Error(err)
			}
		}

		stargazers = stars
	}
	dayStars[day] = *repo.StargazersCount

	return *repo.StargazersCount, nil
}
