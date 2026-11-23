package web

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The worker runs the screen's writes and remote reads one at a time, in the order the keys asked for them.
// A key adds an intent and paints; the worker sends it as a tea.Cmd. Until the job ends, the screen draws the
// intent over the site's last answer. When it ends, each list in the answer replaces that cache wholesale and
// the intent is dropped. On an error the intent is dropped, so the view reverts, and the error goes to the
// status line.

// keyWait bounds the API key read. crypt.Get takes no context, so a read that outlives it finishes on its own.
const keyWait = 30 * time.Second

// drainWait bounds how long q waits for the writes still on the worker before it quits anyway.
const drainWait = 30 * time.Second

var errNoKey = errors.New("no admin API key")

// job is one intent on the worker.
type job struct {
	id     int
	intent intent
}

// intent is one thing the operator asked of the site. send runs on the worker with the client and the site's
// last answers as they are when the job starts, so each write builds on the answer to the one before.
type intent interface {
	label() string
	send(c *Client, last site) (answer, error)
}

// answer is what a job brought back. A nil list was not part of the answer and leaves its cache alone.
type answer struct {
	posts   []Post
	sched   *Schedule
	landing *Landing
	pages   []Page
	keys    []Key
	renamed []Rename
	token   string
	status  string
}

type doneMsg struct {
	id  int
	ans answer
	err error
}

type quitMsg struct{}

// enqueue adds an intent to the worker and starts it when the worker is idle. Without a client, the API key
// is read first.
func (m *model) enqueue(in intent) tea.Cmd {
	if m.client == nil && m.readKey != nil && !m.keyQueued() {
		m.push(readKey{read: m.readKey, wait: keyWait})
	}
	m.push(in)
	return m.startNext()
}

func (m *model) push(in intent) {
	m.lastID++
	m.work = append(m.work, job{id: m.lastID, intent: in})
}

func (m model) keyQueued() bool {
	for _, j := range m.work {
		if _, ok := j.intent.(readKey); ok {
			return true
		}
	}
	return false
}

func (m *model) startNext() tea.Cmd {
	if m.running || len(m.work) == 0 {
		return nil
	}
	m.running = true
	return m.command(m.work[0])
}

// command sends one job. It captures the client and the caches as they are now; the model is not touched
// off the loop.
func (m model) command(j job) tea.Cmd {
	client, last := m.client, m.site
	return func() tea.Msg {
		if _, ok := j.intent.(readKey); !ok && client == nil {
			return doneMsg{id: j.id, err: errNoKey}
		}
		ans, err := j.intent.send(client, last)
		return doneMsg{id: j.id, ans: ans, err: err}
	}
}

// finish ends the running job and starts the next one.
func (m model) finish(msg doneMsg) (tea.Model, tea.Cmd) {
	if len(m.work) == 0 || m.work[0].id != msg.id {
		return m, nil
	}
	done := m.work[0].intent
	m.work = append([]job(nil), m.work[1:]...)
	m.running = false
	if msg.err != nil {
		m.status = msg.err.Error()
		if _, ok := done.(readKey); ok {
			m.work = nil // every job behind the key read needed the key
		} else if reload := reloadAfter(done); reload != nil && !answered(msg.err) && !m.quitting {
			m.push(reload) // a write that failed without a reply may have landed
		}
	} else {
		m.take(msg.ans)
	}
	m.clamp(m.listCount())
	if m.quitting {
		if m.writes() == 0 {
			return m, tea.Quit
		}
		m.status = m.saving()
	}
	cmd := m.startNext()
	return m, cmd
}

// answered reports whether the site replied to the request that failed. A
// reply means the write did not land; no reply leaves it unknown.
func answered(err error) bool {
	var api *APIError
	return errors.As(err, &api)
}

// reloadAfter is the read that shows what a failed write left on the site,
// or nil for a read.
func reloadAfter(in intent) intent {
	switch in.(type) {
	case moveEssay, dropEssay, saveEssay, editSchedule:
		return loadEssays{}
	case savePage, dropPage:
		return loadPages{}
	case editField, setPhoto:
		return loadLanding{}
	case newKey, dropKey:
		return loadKeys{}
	}
	return nil
}

// take replaces each cache the answer carries, wholesale, and points the pending intents at any essay the
// write renamed.
func (m *model) take(ans answer) {
	if ans.token != "" {
		m.client = &Client{Origin: m.origin, Token: ans.token}
	}
	if ans.posts != nil {
		m.site.posts = orderPosts(ans.posts, time.Now())
	}
	if ans.sched != nil && ans.sched.Zone != "" {
		m.site.sched = *ans.sched
	}
	if ans.landing != nil {
		m.site.landing = ans.landing
	}
	if ans.pages != nil {
		m.site.pages = ans.pages
	}
	if ans.keys != nil {
		m.site.keys = ans.keys
	}
	m.follow(ans.renamed)
	if ans.status != "" {
		m.status = ans.status
	}
}

// follow points the waiting intents, and an essay picked for delete or open in vim, at the new slugs.
func (m *model) follow(renamed []Rename) {
	if len(renamed) == 0 {
		return
	}
	to := make(map[string]string, len(renamed))
	for _, r := range renamed {
		to[r.From] = r.To
	}
	for i := range m.work {
		m.work[i].intent = followed(m.work[i].intent, to)
	}
	if next, ok := to[m.editSlug]; ok && m.kind == "essay" {
		m.editSlug = next
	}
}

func followed(in intent, to map[string]string) intent {
	switch in := in.(type) {
	case moveEssay:
		in.slug = renamedSlug(in.slug, to)
		return in
	case dropEssay:
		in.slug = renamedSlug(in.slug, to)
		return in
	case saveEssay:
		in.slug = renamedSlug(in.slug, to)
		return in
	default:
		return in
	}
}

func renamedSlug(slug string, to map[string]string) string {
	if next, ok := to[slug]; ok {
		return next
	}
	return slug
}

// leave quits. With writes still on the worker it waits for them, up to drainWait, while the frame says
// "saving N"; a second q quits at once.
func (m model) leave() (tea.Model, tea.Cmd) {
	if m.quitting || m.writes() == 0 {
		return m, tea.Quit
	}
	m.quitting = true
	m.dropReads()
	m.status = m.saving()
	return m, tea.Tick(drainWait, func(time.Time) tea.Msg { return quitMsg{} })
}

func (m model) saving() string {
	return fmt.Sprintf("saving %d, q quits now", m.writes())
}

func (m model) writes() int {
	n := 0
	for _, j := range m.work {
		if !isRead(j.intent) {
			n++
		}
	}
	return n
}

// dropReads drops the reads that have not started; the key read stays for the writes behind it.
func (m *model) dropReads() {
	var kept []job
	for i, j := range m.work {
		_, key := j.intent.(readKey)
		if (i == 0 && m.running) || key || !isRead(j.intent) {
			kept = append(kept, j)
		}
	}
	m.work = kept
}

func isRead(in intent) bool {
	switch in.(type) {
	case readKey, loadEssays, loadLanding, loadPages, loadKeys:
		return true
	default:
		return false
	}
}

// workLine is what the frame says while the worker is busy.
func (m model) workLine() string {
	if len(m.work) == 0 {
		return ""
	}
	line := m.work[0].intent.label()
	if len(m.work) > 1 {
		line += fmt.Sprintf(", %d more", len(m.work)-1)
	}
	return line
}

// essays is the essay list the screen draws: the site's last list with each pending intent applied in order.
// posts is the view's own copy, so the cache is never edited.
func (m model) essays() []Post {
	posts := append([]Post(nil), m.site.posts...)
	now := time.Now()
	for _, j := range m.work {
		switch in := j.intent.(type) {
		case moveEssay:
			i := postIndex(posts, in.slug)
			if k := neighbour(posts, i, in.way, now); k >= 0 {
				posts[i], posts[k] = posts[k], posts[i]
				posts[i].PublishedAt, posts[k].PublishedAt = posts[k].PublishedAt, posts[i].PublishedAt
			}
		case dropEssay:
			posts = withoutPost(posts, in.slug)
		case saveEssay:
			if i := postIndex(posts, in.slug); in.slug != "" && i >= 0 {
				posts[i].Title, posts[i].Body = in.title, in.body
			}
		}
	}
	return posts
}

func postIndex(posts []Post, slug string) int {
	for i, post := range posts {
		if post.Slug == slug {
			return i
		}
	}
	return -1
}

// neighbour is where the essay at i goes on a move, or -1 when it cannot move that way: only a future essay
// moves, and only past another future essay.
func neighbour(posts []Post, i int, way string, now time.Time) int {
	if i < 0 || i >= len(posts) || !futurePost(posts[i], now) {
		return -1
	}
	k := i - 1
	if way == "down" {
		k = i + 1
	}
	if k < 0 || k >= len(posts) || !futurePost(posts[k], now) {
		return -1
	}
	return k
}

func withoutPost(posts []Post, slug string) []Post {
	out := posts[:0:0]
	for _, post := range posts {
		if post.Slug != slug {
			out = append(out, post)
		}
	}
	return out
}

// pageList is the site's last page list with the pending deletes and saves applied.
func (m model) pageList() []Page {
	pages := append([]Page(nil), m.site.pages...)
	for _, j := range m.work {
		switch in := j.intent.(type) {
		case dropPage:
			kept := pages[:0:0]
			for _, page := range pages {
				if page.Slug != in.slug {
					kept = append(kept, page)
				}
			}
			pages = kept
		case savePage:
			pages = savedPage(pages, Page{Slug: in.slug, Title: in.title, Body: in.body})
		}
	}
	return pages
}

func savedPage(pages []Page, page Page) []Page {
	for i := range pages {
		if pages[i].Slug == page.Slug {
			pages[i] = page
			return pages
		}
	}
	return append(pages, page)
}

// keyList is the site's last key list without the keys being revoked.
func (m model) keyList() []Key {
	keys := append([]Key(nil), m.site.keys...)
	for _, j := range m.work {
		if in, ok := j.intent.(dropKey); ok {
			kept := keys[:0:0]
			for _, key := range keys {
				if key.ID != in.id {
					kept = append(kept, key)
				}
			}
			keys = kept
		}
	}
	return keys
}

// landingView is the site's last landing page with the pending field edits applied, or nil until it answers.
func (m model) landingView() *Landing {
	if m.site.landing == nil {
		return nil
	}
	landing := *m.site.landing
	for _, j := range m.work {
		if in, ok := j.intent.(editField); ok {
			landing = withField(landing, in.key, in.value)
		}
	}
	return &landing
}

// schedView is the site's last weekly slot with the pending field edits applied.
func (m model) schedView() Schedule {
	sched := m.site.sched
	for _, j := range m.work {
		if in, ok := j.intent.(editSchedule); ok {
			sched = in.apply(sched)
		}
	}
	return sched
}

// readKey reads the admin API key with nebula crypt, bounded by wait.
type readKey struct {
	read func() (string, error)
	wait time.Duration
}

func (readKey) label() string { return "reading the API key" }

func (in readKey) send(*Client, site) (answer, error) {
	token, err := within(in.read, in.wait)
	if err != nil {
		return answer{}, fmt.Errorf("could not read the admin API key (%s)", err.Error())
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return answer{}, errors.New("the admin API key is empty")
	}
	return answer{token: token}, nil
}

func within(read func() (string, error), wait time.Duration) (string, error) {
	type result struct {
		token string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		token, err := read()
		done <- result{token, err}
	}()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case r := <-done:
		return r.token, r.err
	case <-timer.C:
		return "", errors.New("timed out")
	}
}

type loadEssays struct{}

func (loadEssays) label() string { return "loading" }

func (loadEssays) send(c *Client, _ site) (answer, error) {
	return readQueue(c, QueueAnswer{}, "")
}

// readQueue is the answer to an essay write: the queue the site answered with or, from a site that answered
// without it, the essays and the schedule read again.
func readQueue(c *Client, qa QueueAnswer, status string) (answer, error) {
	ans := answer{posts: qa.Posts, sched: qa.Schedule, renamed: qa.Renamed, status: status}
	if ans.posts != nil {
		return ans, nil
	}
	posts, err := c.ListPosts()
	if err != nil {
		return answer{}, err
	}
	ans.posts = posts
	sched, err := c.GetSchedule()
	if err != nil && !missingAction(err) {
		return answer{}, err
	}
	if err == nil {
		ans.sched = &sched
	}
	return ans, nil
}

type loadLanding struct{}

func (loadLanding) label() string { return "loading" }

func (loadLanding) send(c *Client, _ site) (answer, error) {
	landing, err := c.GetLanding()
	if err != nil {
		return answer{}, err
	}
	return answer{landing: &landing}, nil
}

type loadPages struct{}

func (loadPages) label() string { return "loading" }

func (loadPages) send(c *Client, _ site) (answer, error) {
	return readPages(c, nil, "")
}

func readPages(c *Client, pages []Page, status string) (answer, error) {
	if pages == nil {
		var err error
		if pages, err = c.ListPages(); err != nil {
			return answer{}, err
		}
	}
	return answer{pages: pages, status: status}, nil
}

type loadKeys struct{}

func (loadKeys) label() string { return "loading" }

func (loadKeys) send(c *Client, _ site) (answer, error) {
	return readKeys(c, nil, "")
}

func readKeys(c *Client, keys []Key, status string) (answer, error) {
	if keys == nil {
		var err error
		if keys, err = c.ListKeys(); err != nil {
			return answer{}, err
		}
	}
	return answer{keys: keys, status: status}, nil
}

// moveEssay trades a future essay's slot with its neighbour, up (sooner) or down (later).
type moveEssay struct{ slug, way string }

func (moveEssay) label() string { return "moving" }

func (in moveEssay) send(c *Client, _ site) (answer, error) {
	qa, err := c.MovePost(in.slug, in.way)
	if err != nil {
		return answer{}, err
	}
	return readQueue(c, qa, "moved")
}

type dropEssay struct{ slug string }

func (dropEssay) label() string { return "deleting" }

func (in dropEssay) send(c *Client, _ site) (answer, error) {
	qa, err := c.DeletePost(in.slug)
	if err != nil {
		return answer{}, err
	}
	return readQueue(c, qa, "deleted")
}

// saveEssay proofreads and saves an essay. An empty slug is a new essay.
type saveEssay struct{ slug, title, body string }

func (saveEssay) label() string { return "saving" }

func (in saveEssay) send(c *Client, _ site) (answer, error) {
	title, body, err := proofOrKeep(c, in.title, in.body)
	if err != nil {
		return answer{}, err
	}
	var qa QueueAnswer
	if in.slug == "" {
		qa, err = c.CreatePost(title, body)
	} else {
		qa, err = c.PutPost(in.slug, title, body)
	}
	if err != nil {
		return answer{}, err
	}
	return readQueue(c, qa, "saved")
}

func proofOrKeep(client *Client, title, body string) (string, string, error) {
	nextTitle, nextBody, err := client.Proof(title, body)
	if missingAction(err) {
		return title, body, nil
	}
	if err != nil {
		return "", "", err
	}
	return nextTitle, nextBody, nil
}

type savePage struct{ slug, title, body string }

func (savePage) label() string { return "saving" }

func (in savePage) send(c *Client, _ site) (answer, error) {
	pa, err := c.PutPage(Page{Slug: in.slug, Title: in.title, Body: in.body})
	if err != nil {
		return answer{}, err
	}
	return readPages(c, pa.Pages, "saved")
}

type dropPage struct{ slug string }

func (dropPage) label() string { return "deleting" }

func (in dropPage) send(c *Client, _ site) (answer, error) {
	pa, err := c.DeletePage(in.slug)
	if err != nil {
		return answer{}, err
	}
	return readPages(c, pa.Pages, "deleted")
}

// editField saves one landing field. It reads the landing page from the
// site inside the same job and changes only that field, so an edit made
// elsewhere since the section opened is not written back over.
type editField struct{ key, value string }

func (editField) label() string { return "saving" }

func (in editField) send(c *Client, _ site) (answer, error) {
	current, err := c.GetLanding()
	if err != nil {
		return answer{}, err
	}
	saved, err := c.PutLanding(withField(current, in.key, in.value))
	if err != nil {
		return answer{}, err
	}
	return answer{landing: &saved, status: "saved"}, nil
}

// setPhoto uploads the portrait or the cover from a file path.
type setPhoto struct{ name, path string }

func (in setPhoto) label() string { return "saving the " + in.name }

func (in setPhoto) send(c *Client, _ site) (answer, error) {
	raw, err := os.ReadFile(in.path)
	if err != nil {
		return answer{}, errors.New("could not read the photo")
	}
	landing, err := c.PutMedia(in.name, raw)
	if err != nil {
		return answer{}, err
	}
	if landing == nil {
		fresh, err := c.GetLanding()
		if err != nil {
			return answer{}, err
		}
		landing = &fresh
	}
	return answer{landing: landing, status: in.name + " saved"}, nil
}

// editSchedule saves one field of the weekly slot. Like editField it reads
// the slot from the site inside the job and changes only that field.
type editSchedule struct {
	field string
	value string
	day   int
}

func (editSchedule) label() string { return "saving" }

func (in editSchedule) apply(sched Schedule) Schedule {
	switch in.field {
	case "weekday":
		sched.Weekday = in.day
	case "time":
		sched.Time = in.value
	case "zone":
		sched.Zone = in.value
	}
	return sched
}

func (in editSchedule) send(c *Client, _ site) (answer, error) {
	current, err := c.GetSchedule()
	if err != nil {
		return answer{}, err
	}
	sa, err := c.PutSchedule(in.apply(current))
	if err != nil {
		return answer{}, err
	}
	ans := answer{sched: &sa.Schedule, posts: sa.Posts, renamed: sa.Renamed, status: "schedule saved"}
	if ans.posts == nil {
		if ans.posts, err = c.ListPosts(); err != nil {
			return answer{}, err
		}
	}
	return ans, nil
}

type newKey struct{ name string }

func (newKey) label() string { return "saving" }

func (in newKey) send(c *Client, _ site) (answer, error) {
	ka, err := c.CreateKey(in.name)
	if err != nil {
		return answer{}, err
	}
	return readKeys(c, ka.Keys, "new key "+ka.Token)
}

type dropKey struct{ id int64 }

func (dropKey) label() string { return "revoking" }

func (in dropKey) send(c *Client, _ site) (answer, error) {
	ka, err := c.DeleteKey(in.id)
	if err != nil {
		return answer{}, err
	}
	return readKeys(c, ka.Keys, "revoked")
}
