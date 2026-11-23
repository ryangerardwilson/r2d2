package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func press(t *testing.T, m tea.Model, keys string) (model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, r := range keys {
		m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m.(model), cmd
}

// fakeSite records each request and answers with the handler a test gives it.
type fakeSite struct {
	mu       sync.Mutex
	requests []string
	srv      *httptest.Server
}

func newFakeSite(t *testing.T, answer func(w http.ResponseWriter, r *http.Request)) *fakeSite {
	t.Helper()
	f := &fakeSite{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.mu.Unlock()
		answer(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeSite) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func days(n int) *time.Time {
	at := time.Now().Add(time.Duration(n) * 24 * time.Hour).UTC().Truncate(time.Second)
	return &at
}

// queue is three future essays and one already out, as the site last answered.
func queue() []Post {
	return orderPosts([]Post{
		{Slug: "a", Title: "A", PublishedAt: days(1)},
		{Slug: "b", Title: "B", PublishedAt: days(8)},
		{Slug: "c", Title: "C", PublishedAt: days(15)},
		{Slug: "z", Title: "Z", PublishedAt: days(-1)},
	}, time.Now())
}

func titles(posts []Post) string {
	var out []string
	for _, post := range posts {
		out = append(out, post.Title)
	}
	return strings.Join(out, " ")
}

func slugs(posts []Post) string {
	var out []string
	for _, post := range posts {
		out = append(out, post.Slug)
	}
	return strings.Join(out, " ")
}

func writeQueue(w http.ResponseWriter, posts []Post, renamed ...Rename) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(QueueAnswer{Post: posts[0], Posts: posts, Schedule: &Schedule{Weekday: 6, Time: "04:00", Zone: "Asia/Kolkata"}, Renamed: renamed})
}

func essayScreen(f *fakeSite) model {
	return model{origin: f.srv.URL, client: &Client{Origin: f.srv.URL, Token: "rgw_test"}, screen: scrEssays, site: site{posts: queue()}, width: 80, height: 24}
}

func TestKeysStayLiveAndAMoveShowsAtOnceThenTakesTheSiteList(t *testing.T) {
	arrived, release := make(chan struct{}), make(chan struct{})
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/posts/b/up" {
			http.NotFound(w, r)
			return
		}
		close(arrived)
		<-release
		// the site's own answer: both essays renamed for their new dates
		writeQueue(w, []Post{
			{Slug: "z", Title: "Z", PublishedAt: days(-1)},
			{Slug: "c", Title: "C", PublishedAt: days(15)},
			{Slug: "a-8", Title: "A", PublishedAt: days(8)},
			{Slug: "b-1", Title: "B (proofread)", PublishedAt: days(1)},
		}, Rename{From: "b", To: "b-1"}, Rename{From: "a", To: "a-8"})
	})
	m := essayScreen(f)
	m.cursor = 1
	m, cmd := press(t, m, "u")
	if cmd == nil {
		t.Fatal("the move did not go to the worker")
	}
	if titles(m.essays()) != "B A C Z" || m.cursor != 0 || !strings.Contains(markedLine(m.View()), "B") || !strings.Contains(m.View(), "moving") {
		t.Fatalf("the move is not shown at once: %s cursor %d\n%s", titles(m.essays()), m.cursor, m.View())
	}
	if titles(m.site.posts) != "A B C Z" || !m.site.posts[0].PublishedAt.Equal(*queue()[0].PublishedAt) {
		t.Fatal("the cached site list was edited in place")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	<-arrived
	start := time.Now()
	m, _ = press(t, m, "jj")
	if m.cursor != 2 {
		t.Fatalf("j was dropped while the site answered: cursor %d", m.cursor)
	}
	m, _ = press(t, m, "k?")
	if m.cursor != 1 || !m.help {
		t.Fatalf("k or ? was dropped: cursor %d help %v", m.cursor, m.help)
	}
	m, _ = press(t, m, "?")
	if time.Since(start) > time.Second {
		t.Fatal("keys waited on the request")
	}
	close(release)
	next, cmd := m.Update(<-done)
	m = next.(model)
	if cmd != nil || len(m.work) != 0 {
		t.Fatalf("work left %v", m.work)
	}
	if slugs(m.site.posts) != "b-1 a-8 c z" || titles(m.essays()) != "B (proofread) A C Z" || m.status != "moved" {
		t.Fatalf("the cache is not the site's list: %s / %s status %q", slugs(m.site.posts), titles(m.essays()), m.status)
	}
	if got := f.seen(); len(got) != 1 || got[0] != "POST /api/v1/posts/b/up" {
		t.Fatalf("one move took %v", got)
	}
	if strings.Contains(m.View(), "moving") {
		t.Fatalf("still moving\n%s", m.View())
	}
}

func TestAFailedMoveRevertsAndNamesTheError(t *testing.T) {
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"the queue changed while saving, try again"}`))
	})
	m := essayScreen(f)
	m.cursor = 2
	m, cmd := press(t, m, "u")
	if titles(m.essays()) != "A C B Z" {
		t.Fatalf("pending %s", titles(m.essays()))
	}
	next, _ := m.Update(cmd())
	m = next.(model)
	if titles(m.essays()) != "A B C Z" || titles(m.site.posts) != "A B C Z" || len(m.work) != 0 {
		t.Fatalf("not reverted: %s", titles(m.essays()))
	}
	if m.status != "the queue changed while saving, try again" || !strings.Contains(m.View(), "try again") {
		t.Fatalf("status %q", m.status)
	}
}

func TestWritesRunInOrderAndFollowRenames(t *testing.T) {
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/posts/c/up":
			writeQueue(w, []Post{
				{Slug: "a", Title: "A", PublishedAt: days(1)},
				{Slug: "c-8", Title: "C", PublishedAt: days(8)},
				{Slug: "b-15", Title: "B", PublishedAt: days(15)},
				{Slug: "z", Title: "Z", PublishedAt: days(-1)},
			}, Rename{From: "c", To: "c-8"}, Rename{From: "b", To: "b-15"})
		case "/api/v1/posts/c-8/up":
			writeQueue(w, []Post{
				{Slug: "c-1", Title: "C", PublishedAt: days(1)},
				{Slug: "a-8", Title: "A", PublishedAt: days(8)},
				{Slug: "b-15", Title: "B", PublishedAt: days(15)},
				{Slug: "z", Title: "Z", PublishedAt: days(-1)},
			}, Rename{From: "c-8", To: "c-1"}, Rename{From: "a", To: "a-8"})
		default:
			http.NotFound(w, r)
		}
	})
	m := essayScreen(f)
	m.cursor = 2
	m, first := press(t, m, "u")
	m, second := press(t, m, "u")
	if first == nil || second != nil || len(m.work) != 2 {
		t.Fatalf("the second move did not wait its turn: %v %v %d", first, second, len(m.work))
	}
	if titles(m.essays()) != "C A B Z" || m.cursor != 0 || !strings.Contains(m.View(), "moving, 1 more") {
		t.Fatalf("pending %s cursor %d\n%s", titles(m.essays()), m.cursor, m.View())
	}
	next, cmd := m.Update(first())
	m = next.(model)
	if cmd == nil || titles(m.essays()) != "C A B Z" {
		t.Fatalf("after the first answer %s", titles(m.essays()))
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if slugs(m.site.posts) != "c-1 a-8 b-15 z" || len(m.work) != 0 || m.status != "moved" {
		t.Fatalf("%s %q", slugs(m.site.posts), m.status)
	}
	if got := strings.Join(f.seen(), ","); got != "POST /api/v1/posts/c/up,POST /api/v1/posts/c-8/up" {
		t.Fatalf("requests %s", got)
	}
}

func TestADeleteHidesTheRowAndAnOlderSiteIsReadOnce(t *testing.T) {
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "DELETE /api/v1/posts/b":
			w.WriteHeader(http.StatusNoContent)
		case "GET /api/v1/posts":
			_ = json.NewEncoder(w).Encode([]Post{{Slug: "a", Title: "A", PublishedAt: days(1)}, {Slug: "c-8", Title: "C", PublishedAt: days(8)}})
		case "GET /api/v1/schedule":
			_ = json.NewEncoder(w).Encode(Schedule{Weekday: 6, Time: "04:00", Zone: "Asia/Kolkata"})
		default:
			http.NotFound(w, r)
		}
	})
	m := essayScreen(f)
	m.cursor = 1
	m, _ = press(t, m, "x")
	m, cmd := press(t, m, "y")
	if titles(m.essays()) != "A C Z" || titles(m.site.posts) != "A B C Z" || !strings.Contains(m.View(), "deleting") {
		t.Fatalf("pending %s", titles(m.essays()))
	}
	next, _ := m.Update(cmd())
	m = next.(model)
	if slugs(m.site.posts) != "a c-8" || m.site.sched.Zone != "Asia/Kolkata" || m.status != "deleted" {
		t.Fatalf("%s %+v %q", slugs(m.site.posts), m.site.sched, m.status)
	}
	if got := strings.Join(f.seen(), ","); got != "DELETE /api/v1/posts/b,GET /api/v1/posts,GET /api/v1/schedule" {
		t.Fatalf("requests %s", got)
	}
}

func TestTheFirstFramePaintsBeforeTheKeyIsRead(t *testing.T) {
	asked, give := make(chan struct{}), make(chan struct{})
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer rgw_late" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode([]Page{{Slug: "about", Title: "About"}})
	})
	m := newModel(f.srv.URL, "", func() (string, error) {
		close(asked)
		<-give
		return "rgw_late", nil
	})
	if view := m.View(); !strings.Contains(view, "Essays") || !strings.Contains(view, "reading the API key") {
		t.Fatalf("first frame\n%s", view)
	}
	read := m.Init()
	if read == nil {
		t.Fatal("no key read")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- read() }()
	<-asked
	m, _ = press(t, m, "jj")
	if m.cursor != 2 {
		t.Fatalf("keys waited on the key: cursor %d", m.cursor)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd != nil || m.screen != scrPages || len(m.work) != 2 {
		t.Fatalf("the page load did not wait behind the key: %v %d", cmd, len(m.work))
	}
	close(give)
	next, cmd = m.Update(<-done)
	m = next.(model)
	if m.client == nil || m.client.Token != "rgw_late" || cmd == nil {
		t.Fatalf("client %+v", m.client)
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if len(m.site.pages) != 1 || strings.Contains(m.View(), "rgw_late") {
		t.Fatalf("pages %v\n%s", m.site.pages, m.View())
	}
}

func TestTheKeyReadTimesOut(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	_, err := readKey{read: func() (string, error) { <-block; return "", nil }, wait: 20 * time.Millisecond}.send(nil, site{})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("got %v", err)
	}
	m := newModel("https://example.test", "", nil)
	m.work[0].intent = readKey{read: func() (string, error) { <-block; return "", nil }, wait: 20 * time.Millisecond}
	m.work = append(m.work, job{id: 9, intent: loadEssays{}})
	next, _ := m.Update(m.Init()())
	got := next.(model)
	if len(got.work) != 0 || !strings.Contains(got.status, "could not read the admin API key (timed out)") {
		t.Fatalf("work %v status %q", got.work, got.status)
	}
}

func TestQuitWaitsForWritesThenQuits(t *testing.T) {
	f := newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {
		writeQueue(w, queue())
	})
	m := essayScreen(f)
	m.cursor = 1
	m, move := press(t, m, "u")
	m, _ = press(t, m, "q")
	m, wait := press(t, m, "q")
	if !m.quitting || wait == nil || !strings.Contains(m.View(), "saving 1") {
		t.Fatalf("quit did not wait for the move\n%s", m.View())
	}
	next, cmd := m.Update(move())
	if cmd == nil {
		t.Fatal("did not quit after the move")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok || len(next.(model).work) != 0 {
		t.Fatal("did not quit after the move")
	}
	idle := model{screen: scrMenu}
	if _, cmd := press(t, idle, "q"); cmd == nil {
		t.Fatal("q on an idle menu did not quit")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q on an idle menu did not quit")
	}
}

func TestAWriteWithNoReplyReloadsItsSection(t *testing.T) {
	m := essayScreen(newFakeSite(t, func(w http.ResponseWriter, r *http.Request) {}))
	next, _ := m.finish(doneMsg{id: 0, err: errors.New("could not reach the site")})
	if got := next.(model); len(got.work) != 0 {
		t.Fatalf("a finish for no running job queued work: %d", len(got.work))
	}
	m.push(moveEssay{slug: "b", way: "up"})
	m.running = true
	next, _ = m.finish(doneMsg{id: m.work[0].id, err: errors.New("could not reach the site")})
	got := next.(model)
	if len(got.work) != 1 {
		t.Fatalf("work %d", len(got.work))
	}
	if _, ok := got.work[0].intent.(loadEssays); !ok {
		t.Fatalf("queued %T, want loadEssays", got.work[0].intent)
	}
	if got.status != "could not reach the site" {
		t.Fatalf("status %q", got.status)
	}
}
