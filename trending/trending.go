package trending

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/darjun/ghtrending"
	"github.com/tal-tech/go-zero/core/logx"
)

const checkInterval = time.Minute * 10

type (
	Monitor struct {
		name     string
		author   string
		langs    []string
		send     func(string) error
		previous []Position
	}

	Position struct {
		Lang string
		Pos  int
	}
)

func NewMonitor(repo, lang string, sender func(string) error) *Monitor {
	fields := strings.Split(repo, "/")
	return &Monitor{
		author: fields[0],
		name:   fields[1],
		langs:  []string{"", lang},
		send:   sender,
	}
}

func (m *Monitor) Start() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	var buf strings.Builder
	for range ticker.C {
		buf.Reset()
		positions := m.findInTrending()
		if m.checkIfChanged(positions) {
			m.previous = positions
		}
		if len(positions) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintln(m.name))
		for _, pos := range positions {
			if pos.Lang == "" {
				buf.WriteString(fmt.Sprintf("trending: %d\n", pos.Pos))
			} else {
				buf.WriteString(fmt.Sprintf("%s trending: %d\n", pos.Lang, pos.Pos))
			}
		}

		if err := m.send(buf.String()); err != nil {
			logx.Error(err)
		}
	}
}

func (m *Monitor) findInTrending() []Position {
	var positions []Position

	for _, lang := range m.langs {
		repos, err := ghtrending.TrendingRepositories(ghtrending.WithDaily(), ghtrending.WithLanguage(lang))
		if err != nil {
			log.Fatal(err)
		}

		for i, each := range repos {
			if m.name == each.Name && m.author == each.Author {
				positions = append(positions, Position{
					Lang: lang,
					Pos:  i + 1,
				})
			}
		}
	}

	return positions
}

func (m *Monitor) checkIfChanged(positions []Position) bool {
	if len(positions) != len(m.previous) {
		return true
	}

	pm := make(map[string]int)
	for _, pos := range m.previous {
		pm[pos.Lang] = pos.Pos
	}

	for _, pos := range positions {
		if pos.Pos != pm[pos.Lang] {
			return true
		}
	}

	return false
}
