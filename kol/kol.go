package main

import (
	"flag"
	"fmt"
	"sort"
	"time"

	"stargazers/gh"

	"github.com/google/go-github/v39/github"
	"github.com/schollz/progressbar/v3"
	"github.com/tal-tech/go-zero/core/fx"
	"github.com/tal-tech/go-zero/core/logx"
)

const starAtFormat = "01-02 15:04:05"

var (
	repo  = flag.String("repo", "", "the github repo")
	token = flag.String("token", "", "the github token")
	top   = flag.Int("top", 0, "top kols, default to all")
)

func main() {
	flag.Parse()

	if len(*token) == 0 {
		flag.Usage()
		return
	}

	cli := gh.CreateClient(*token)
	owner, project, err := gh.ParseRepo(*repo)
	logx.Must(err)
	stargazers, err := gh.RequestAll(cli, owner, project)
	logx.Must(err)

	users := collectUsers(cli, stargazers)
	sort.Slice(users, func(i, j int) bool {
		return *users[i].Followers < *users[j].Followers
	})

	var start = 0
	if *top > 0 {
		start = len(users) - *top
	}
	fmt.Printf("\n")
	for _, user := range users[start:] {
		if user.Name != nil {
			fmt.Printf("id: %s, name: %s, followers: %d, starAt: %s\n",
				*user.Login, *user.Name, *user.Followers, stargazers[*user.Login].Format(starAtFormat))
		} else {
			fmt.Printf("id: %s, followers: %d, starAt: %s\n",
				*user.Login, *user.Followers, stargazers[*user.Login].Format(starAtFormat))
		}
	}
}

func collectUsers(cli *github.Client, stargazers map[string]time.Time) []*github.User {
	var users []*github.User
	bar := progressbar.New(len(stargazers))

	for id := range stargazers {
		bar.Add(1)
		user, err := gh.RequestUser(cli, id)
		if err != nil {
			fmt.Printf("failed, id: %s, error: %s\n", id, err.Error())
			continue
		}

		users = append(users, user)
	}

	return users
}

// if too many stargazers, don't use this function, rate limit will be triggered.
func collectUsersFast(cli *github.Client, stargazers map[string]time.Time) []*github.User {
	bar := progressbar.New(len(stargazers))
	items, err := fx.From(func(source chan<- interface{}) {
		for each := range stargazers {
			source <- each
		}
	}).Map(func(item interface{}) interface{} {
		id := item.(string)
		user, err := gh.RequestUser(cli, id)
		if err != nil {
			fmt.Printf("failed, id: %s, error: %s\n", id, err.Error())
			return nil
		}

		return user
	}).Reduce(func(pipe <-chan interface{}) (interface{}, error) {
		var users []*github.User
		for item := range pipe {
			bar.Add(1)
			if item == nil {
				continue
			}
			user := item.(*github.User)
			users = append(users, user)
		}
		return users, nil
	})
	logx.Must(err)

	return items.([]*github.User)
}
