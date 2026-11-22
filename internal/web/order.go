package web

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func orderPosts(posts []Post, now time.Time) []Post {
	var future, past []Post
	for _, post := range posts {
		if futurePost(post, now) {
			future = append(future, post)
			continue
		}
		past = append(past, post)
	}
	sort.Slice(future, func(i, j int) bool {
		return future[i].PublishedAt.Before(*future[j].PublishedAt)
	})
	sort.Slice(past, func(i, j int) bool {
		return after(past[i].PublishedAt, past[j].PublishedAt)
	})
	return append(future, past...)
}

func after(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.After(*b)
}

// futurePost is true for an essay still in the queue.
func futurePost(post Post, now time.Time) bool {
	return post.PublishedAt != nil && post.PublishedAt.After(now)
}

func whenLine(post Post, loc *time.Location, now time.Time) string {
	if post.PublishedAt == nil {
		return "unscheduled"
	}
	at := post.PublishedAt.In(loc)
	mark := "out"
	if futurePost(post, now) {
		mark = "queue"
	}
	return fmt.Sprintf("%-5s  %s", mark, at.Format("Mon 2 Jan 15:04"))
}

func loadZone(name string) *time.Location {
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

func weekdayName(day int) string {
	if day < 0 || day > 6 {
		return "unknown"
	}
	return time.Weekday(day).String()
}

func parseWeekday(text string) (int, bool) {
	text = strings.TrimSpace(text)
	for day := time.Sunday; day <= time.Saturday; day++ {
		if strings.EqualFold(text, day.String()) {
			return int(day), true
		}
	}
	if len(text) == 1 && text[0] >= '0' && text[0] <= '6' {
		return int(text[0] - '0'), true
	}
	return 0, false
}
