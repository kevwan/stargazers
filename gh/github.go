package gh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"stargazers/feishu"

	"github.com/google/go-github/v33/github"
	"github.com/tal-tech/go-zero/core/logx"
	"golang.org/x/oauth2"
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
	repo     string
	token    string
	app      string
	secret   string
	receiver string
}

func NewMonitor(repo, token, app, secret, receiver string) Monitor {
	return Monitor{
		repo:     repo,
		token:    token,
		app:      app,
		secret:   secret,
		receiver: receiver,
	}
}

func (m Monitor) Start() error {
	words := strings.Split(m.repo, "/")
	if len(words) != 2 {
		return errors.New("repo should be <owner>/<project> format")
	}

	owner := words[0]
	project := words[1]
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: m.token,
	})
	tc := oauth2.NewClient(context.Background(), ts)
	cli := github.NewClient(tc)

	if stars, err := m.requestAll(cli, owner, project); err != nil {
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

	name, followers, err := m.requestUser(cli, *gazer.User.Login)
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
	fmt.Fprintf(&builder, "user: %s\n", *gazer.User.Login)
	if len(name) > 0 {
		fmt.Fprintf(&builder, "name: %s\n", name)
	}
	fmt.Fprintf(&builder, "followers: %d\n", followers)
	fmt.Fprintf(&builder, "time: %s\n", gazer.StarredAt.Time.Local().Format(starAtFormat))
	fmt.Fprintf(&builder, "today: %d", m.countsToday(total))
	if err := m.send(builder.String()); err != nil {
		logx.Error(err)
	}
}

func (m Monitor) requestAll(cli *github.Client, owner, project string) (map[string]time.Time, error) {
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

func (m Monitor) requestUser(cli *github.Client, id string) (name string, followers int, err error) {
	user, _, err := cli.Users.Get(context.Background(), id)
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

func (m Monitor) send(text string) error {
	return feishu.Send(m.app, m.secret, m.receiver, text)
}

func (m Monitor) totalCount(cli *github.Client, owner, project string) (int, error) {
	repo, _, err := cli.Repositories.Get(context.Background(), owner, project)
	if err != nil {
		return 0, err
	}

	day := time.Now().Format(dayFormat)
	prev := dayStars[day]
	if *repo.StargazersCount < prev {
		stars, err := m.requestAll(cli, owner, project)
		if err != nil {
			return 0, err
		}

		for k, v := range stargazers {
			if _, ok := stars[k]; ok {
				continue
			}

			name, followers, err := m.requestUser(cli, k)
			if err != nil {
				return 0, err
			}

			if len(name) > 0 {
				if err := m.send(fmt.Sprintf("unstar\nid: %s\nname: %s\nfollowers: %d\nstarAt: %s",
					k, name, followers, v.Format(starAtFormat))); err != nil {
					logx.Error(err)
				}
			} else {
				if err := m.send(fmt.Sprintf("unstar\nid: %s\nfollowers: %d\nstarAt: %s",
					k, followers, v.Format(starAtFormat))); err != nil {
					logx.Error(err)
				}
			}
		}

		stargazers = stars
	}
	dayStars[day] = *repo.StargazersCount

	return *repo.StargazersCount, nil
}
