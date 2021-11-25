package gh

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/tal-tech/go-zero/core/collection"
	"github.com/tal-tech/go-zero/core/logx"
)

const (
	pageSize       = 100
	queueSize      = 100
	dayFormat      = "2006 01-02"
	starAtFormat   = "01-02 15:04:05"
	unstarAtFormat = "2006 01-02 15:04:05"
)

var (
	stargazers = make(map[string]time.Time)
	dayStars   = make(map[string]int)
	startTime  = time.Now()
	fifo       = collection.NewQueue(queueSize)
)

type Monitor struct {
	cfg  Config
	send func(string) error
}

func NewMonitor(cfg Config, send func(text string) error) Monitor {
	return Monitor{
		cfg:  cfg,
		send: send,
	}
}

func (m Monitor) Start() error {
	owner, project, err := ParseRepo(m.cfg.Repo)
	if err != nil {
		return err
	}

	cli := CreateClient(m.cfg.Token)
	if stars, err := RequestAll(cli, owner, project); err != nil {
		return err
	} else {
		stargazers = stars
	}

	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for range ticker.C {
		m.refresh(cli, owner, project)
		m.report()
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

func (m Monitor) handleResponseError(err error, repo *github.Repository, k string, v time.Time) {
	logx.Error(err)

	if !m.cfg.Verbose {
		return
	}

	switch ve := err.(type) {
	case *github.ErrorResponse:
		if ve.Response.StatusCode != http.StatusNotFound {
			break
		}

		var builder strings.Builder
		fmt.Fprintln(&builder, "account deleted")
		fmt.Fprintf(&builder, "stars: %d\n", *repo.StargazersCount)
		fmt.Fprintf(&builder, "today: %d\n", m.countsToday(*repo.StargazersCount))
		fmt.Fprintf(&builder, "user: %s\n", k)
		fmt.Fprintf(&builder, "starAt: %s", v.Local().Format(unstarAtFormat))
		fifo.Put(builder.String())
	}
}

func (m Monitor) refresh(cli *github.Client, owner, project string) {
	count, err := m.totalCount(cli, owner, project)
	if err != nil {
		logx.Errorf("refresh - %s", err.Error())
		return
	}

	logx.Infof("stars: %d", count)
	if err := m.requestPage(cli, owner, project, count, (count+pageSize-1)/pageSize); err != nil {
		logx.Error(err)
	}
}

func (m Monitor) report() {
	for !fifo.Empty() {
		val, ok := fifo.Take()
		if !ok {
			break
		}

		if err := m.send(val.(string)); err != nil {
			fifo.Put(val)
			logx.Error(err)
			break
		}
	}
}

func (m Monitor) reportStarring(cli *github.Client, owner, project string, total int, gazer *github.Stargazer) {
	if gazer.StarredAt.Time.Before(startTime) {
		return
	}

	ensureOnce(func() error {
		name, followers, err := m.requestNameFollowers(cli, *gazer.User.Login)
		if err != nil {
			logx.Error(err)
			return err
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
		text := builder.String()
		fifo.Put(text)
		logx.Infof("star-event: %s", text)

		return nil
	}, time.Minute)
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

func (m Monitor) reportUnstar(repo *github.Repository, id string, name string, followers int, v time.Time) {
	var builder strings.Builder
	fmt.Fprintln(&builder, "unstar")
	fmt.Fprintf(&builder, "stars: %d\n", *repo.StargazersCount)
	fmt.Fprintf(&builder, "today: %d\n", m.countsToday(*repo.StargazersCount))
	fmt.Fprintf(&builder, "user: %s\n", id)
	if len(name) > 0 {
		fmt.Fprintf(&builder, "name: %s\n", name)
	}
	if followers > 0 {
		fmt.Fprintf(&builder, "followers: %d\n", followers)
	}
	fmt.Fprintf(&builder, "starAt: %s", v.Local().Format(unstarAtFormat))
	fifo.Put(builder.String())
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
				m.handleResponseError(err, repo, k, v)
				continue
			}

			m.reportUnstar(repo, k, name, followers, v)
		}

		stargazers = stars
	}
	dayStars[day] = *repo.StargazersCount

	return *repo.StargazersCount, nil
}

func ensureOnce(fn func() error, interval time.Duration) {
	if err := fn(); err == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := fn(); err == nil {
					return
				}
			}
		}
	}()
}
