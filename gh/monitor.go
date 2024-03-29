package gh

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stargazers/sender"

	"github.com/google/go-github/v39/github"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	pageSize        = 100
	queueSize       = 100
	dayFormat       = "2006 01-02"
	expectDayLayout = "2006-01-02"
	starAtFormat    = "01-02 15:04:05"
	unstarAtFormat  = "2006 01-02 15:04:05"
)

var (
	stargazers = make(map[string]time.Time)
	dayStars   = make(map[string]int)
	startTime  = time.Now()
	fifo       = collection.NewQueue(queueSize)
)

type Monitor struct {
	cfg    Config
	cli    *github.Client
	sender sender.Sender
}

func NewMonitor(cfg Config, sender sender.Sender) Monitor {
	return Monitor{
		cfg:    cfg,
		cli:    CreateClient(cfg.Token),
		sender: sender,
	}
}

func (m Monitor) Start() {
	owner, project, err := ParseRepo(m.cfg.Repo)
	logx.Must(err)

	stars, err := RequestAll(m.cli, owner, project)
	logx.Must(err)
	stargazers = stars

	if m.cfg.Expect != nil {
		_, err := time.Parse(expectDayLayout, m.cfg.Expect.Date)
		logx.Must(err)
	}

	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for range ticker.C {
		m.refresh(owner, project)
		m.report()
	}
}

func (m Monitor) beginOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func (m Monitor) calculateExpect(buf *strings.Builder, stars int) {
	if m.cfg.Expect == nil {
		return
	}

	if stars > m.cfg.Expect.Stars {
		return
	}

	deadline, err := time.Parse(expectDayLayout, m.cfg.Expect.Date)
	if err != nil || time.Now().After(deadline) {
		return
	}

	diff := deadline.Sub(time.Now()).Hours() / 24
	expect := float64(m.cfg.Expect.Stars-stars) / diff
	fmt.Fprintf(buf, "\nexpect: %.2f per day", expect)
}

func (m Monitor) compare(buf *strings.Builder, total int) {
	for _, comp := range m.cfg.Comparisons {
		owner, project, err := ParseRepo(comp)
		if err != nil {
			logx.Error(err)
			return
		}

		repo, _, err := m.cli.Repositories.Get(context.Background(), owner, project)
		fmt.Fprintf(buf, "\n%s: %d/%d", project, total-*repo.StargazersCount, *repo.StargazersCount)
	}
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
		m.calculateExpect(&builder, *repo.StargazersCount)
		m.compare(&builder, *repo.StargazersCount)
		fifo.Put(builder.String())
	}
}

func (m Monitor) refresh(owner, project string) {
	count, err := m.totalCount(owner, project)
	if err != nil {
		logx.Errorf("refresh - %s", err.Error())
		return
	}

	logx.Infof("stars: %d", count)
	if err := m.requestPage(owner, project, count, (count+pageSize-1)/pageSize); err != nil {
		logx.Error(err)
	}
}

func (m Monitor) report() {
	for !fifo.Empty() {
		val, ok := fifo.Take()
		if !ok {
			break
		}

		if err := m.sender.Send(val.(string)); err != nil {
			fifo.Put(val)
			logx.Error(err)
			break
		}
	}
}

func (m Monitor) reportStarring(owner, project string, total int, gazer *github.Stargazer) {
	if gazer.StarredAt.Time.Before(startTime) {
		return
	}

	ensureOnce(func() error {
		name, followers, err := m.requestNameFollowers(*gazer.User.Login)
		if err != nil {
			logx.Error(err)
			return err
		}

		// refresh count, because users might star after fetching count
		if count, err := m.totalCount(owner, project); err == nil {
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
		m.compare(&builder, total)
		m.calculateExpect(&builder, total)
		text := builder.String()
		fifo.Put(text)
		logx.Infof("star-event: %s", text)

		return nil
	}, time.Minute)
}

func (m Monitor) requestPage(owner, project string, count, page int) error {
	gazers, resp, err := m.cli.Activity.ListStargazers(context.Background(),
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
		m.reportStarring(owner, project, count, gazer)
	}

	if len(gazers) > 0 && gazers[0].StarredAt.Time.After(m.beginOfDay(time.Now())) {
		return m.requestPage(owner, project, count, resp.PrevPage)
	}

	return nil
}

func (m Monitor) requestNameFollowers(id string) (name string, followers int, err error) {
	var user *github.User
	user, err = RequestUser(m.cli, id)
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
	m.compare(&builder, *repo.StargazersCount)
	m.calculateExpect(&builder, *repo.StargazersCount)
	fifo.Put(builder.String())
}

func (m Monitor) totalCount(owner, project string) (int, error) {
	repo, _, err := m.cli.Repositories.Get(context.Background(), owner, project)
	if err != nil {
		return 0, err
	}

	day := time.Now().Format(dayFormat)
	prev := dayStars[day]
	if *repo.StargazersCount < prev {
		stars, err := RequestAll(m.cli, owner, project)
		if err != nil {
			return 0, err
		}

		for k, v := range stargazers {
			if _, ok := stars[k]; ok {
				continue
			}

			name, followers, err := m.requestNameFollowers(k)
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
