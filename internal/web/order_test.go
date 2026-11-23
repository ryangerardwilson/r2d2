package web

import (
	"errors"
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
	landing := Landing{Hero: "I write essays here, now and then."}
	setField(&landing, "hero", "A shorter line.")
	setField(&landing, "quote.0.quote", "Nice.")
	if landing.Hero != "A shorter line." || landing.Quotes[0].Quote != "Nice." {
		t.Fatalf("%#v", landing)
	}
}

func TestAFailedLandingSaveKeepsTheLoadedPage(t *testing.T) {
	m := model{screen: scrLanding, client: &Client{}, site: site{landing: &Landing{Hero: "Loaded.", Quotes: []Quote{{Quote: "Old."}}}}}
	m.mode, m.kind, m.editKey, m.draft = modePrompt, "field", "quote.0.quote", "Unsaved."
	next, cmd := m.submitPrompt()
	if cmd == nil {
		t.Fatal("no save")
	}
	if got := rowValue(next.(model).rows(), "quote.0.quote"); got != "Unsaved." {
		t.Fatalf("the edit is not shown while it saves: %q", got)
	}
	if m.site.landing.Quotes[0].Quote != "Old." {
		t.Fatal("the edit was written into the cached page")
	}
	next, _ = next.(model).Update(doneMsg{id: next.(model).work[0].id, err: errors.New("the site returned 500")})
	landing := next.(model).site.landing
	if landing.Hero != "Loaded." || landing.Quotes[0].Quote != "Old." {
		t.Fatalf("%#v", landing)
	}
	rows := next.(model).rows()
	if rowValue(rows, "quote.0.quote") != "Old." || rows[len(rows)-1].key != "cover" || next.(model).status != "the site returned 500" {
		t.Fatalf("rows %v status %q", rows, next.(model).status)
	}
}

func rowValue(rows []row, key string) string {
	for _, row := range rows {
		if row.key == key {
			return row.value
		}
	}
	return ""
}
