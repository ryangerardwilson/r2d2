package web

import (
	"testing"
	"time"
)

func TestOrderPostsPutsTheQueueFirst(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	soon := now.Add(48 * time.Hour)
	later := now.Add(8 * 24 * time.Hour)
	old := now.Add(-24 * time.Hour)
	posts := orderPosts([]Post{
		{Slug: "old", PublishedAt: &old},
		{Slug: "later", PublishedAt: &later},
		{Slug: "soon", PublishedAt: &soon},
	}, now)
	if posts[0].Slug != "soon" || posts[1].Slug != "later" || posts[2].Slug != "old" {
		t.Fatalf("%s %s %s", posts[0].Slug, posts[1].Slug, posts[2].Slug)
	}
}

func TestParseWeekday(t *testing.T) {
	day, ok := parseWeekday("Saturday")
	if !ok || day != 6 {
		t.Fatalf("%d %v", day, ok)
	}
	day, ok = parseWeekday("0")
	if !ok || day != 0 {
		t.Fatalf("%d %v", day, ok)
	}
	if _, ok := parseWeekday("8"); ok {
		t.Fatal("accepted 8")
	}
}

func TestLandingFieldRoundTrip(t *testing.T) {
	home := Home{Hero: "I write essays here, now and then."}
	setField(&home, "hero", "A shorter line.")
	setField(&home, "quote.0.quote", "Nice.")
	if home.Hero != "A shorter line." || home.Quotes[0].Quote != "Nice." {
		t.Fatalf("%#v", home)
	}
}
