package trending

import (
	"fmt"
	"strings"
	"time"

	"stargazers/sender"

	"github.com/andygrunwald/go-trending"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	checkInterval = time.Minute * 10
	dailyRange    = "daily"
	weeklyRange   = "weekly"
	monthlyRange  = "monthly"
)

type (
	Trending struct {
		Language   string   `json:"language,default=Go"`
		DateRanges []string `json:"dateRanges"`
	}

	Monitor struct {
		name       string
		author     string
		langs      []string
		dateRanges []string
		sender     sender.Sender
		previous   []Position
	}

	Position struct {
		Lang  string
		Range string
		Pos   int
	}
)

func NewMonitor(repo string, trend Trending, sender sender.Sender) *Monitor {
	fields := strings.Split(repo, "/")
	return &Monitor{
		author:     fields[0],
		name:       fields[1],
		langs:      []string{"", trend.Language},
		dateRanges: trend.DateRanges,
		sender:     sender,
	}
}

func (m *Monitor) Start() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	var buf strings.Builder
	for range ticker.C {
		buf.Reset()
		positions := m.findInTrending()
		if !m.checkIfChanged(positions) {
			continue
		}

		m.previous = positions
		if len(positions) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintln(m.name))
		for _, pos := range positions {
			switch pos.Range {
			case dailyRange:
				if pos.Lang == "" {
					buf.WriteString(fmt.Sprintf("daily trending: %d\n", pos.Pos))
				} else {
					buf.WriteString(fmt.Sprintf("%s daily trending: %d\n", pos.Lang, pos.Pos))
				}
			case weeklyRange:
				if pos.Lang == "" {
					buf.WriteString(fmt.Sprintf("weekly trending: %d\n", pos.Pos))
				} else {
					buf.WriteString(fmt.Sprintf("%s weekly trending: %d\n", pos.Lang, pos.Pos))
				}
			case monthlyRange:
				if pos.Lang == "" {
					buf.WriteString(fmt.Sprintf("monthly trending: %d\n", pos.Pos))
				} else {
					buf.WriteString(fmt.Sprintf("%s monthly trending: %d\n", pos.Lang, pos.Pos))
				}
			}
		}

		if err := m.sender.Send(buf.String()); err != nil {
			logx.Error(err)
		}
	}
}

func (m *Monitor) findInTrending() (positions []Position) {
	trend := trending.NewTrending()
	for _, dateRange := range m.dateRanges {
		for _, lang := range m.langs {
			var repos []trending.Project
			var err error
			switch dateRange {
			case dailyRange:
				repos, err = trend.GetProjects(trending.TimeToday, lang)
			case weeklyRange:
				repos, err = trend.GetProjects(trending.TimeWeek, lang)
			case monthlyRange:
				repos, err = trend.GetProjects(trending.TimeMonth, lang)
			}
			if err != nil {
				if e := m.sender.Send(err.Error()); err != nil {
					logx.Error(e)
				}
				return
			}

			for i, each := range repos {
				if m.name == each.RepositoryName && m.author == each.Owner {
					positions = append(positions, Position{
						Lang:  lang,
						Range: dateRange,
						Pos:   i + 1,
					})
				}
			}
		}
	}

	return
}

func (m *Monitor) checkIfChanged(positions []Position) bool {
	if len(positions) != len(m.previous) {
		return true
	}

	pm := make(map[string]int)
	for _, pos := range m.previous {
		pm[pos.Lang+pos.Range] = pos.Pos
	}

	for _, pos := range positions {
		if pos.Pos != pm[pos.Lang+pos.Range] {
			return true
		}
	}

	return false
}
