package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls the ryangerardwilson.com admin API. The token is not printed.
type Client struct {
	Origin string
	Token  string
	HTTP   *http.Client
}

type Post struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	PublishedAt *time.Time `json:"published_at"` // nil is an unscheduled draft
}

type Page struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Quote struct {
	Quote string `json:"quote"`
	Name  string `json:"name"`
	Work  string `json:"work"`
}

type Social struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Resource struct {
	Heading string `json:"heading"`
	Title   string `json:"title"`
	Sub     string `json:"sub"`
	CTA     string `json:"cta"`
	Href    string `json:"href"`
}

type Landing struct {
	BrandFull       string   `json:"brand_full"`
	BrandShort      string   `json:"brand_short"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	Hero            string   `json:"hero"`
	Lede            string   `json:"lede"`
	Praise          string   `json:"praise"`
	Quotes          []Quote  `json:"quotes"`
	Resource        Resource `json:"resource"`
	AboutHeading    string   `json:"about_heading"`
	AboutText       string   `json:"about_text"`
	CTATitle        string   `json:"cta_title"`
	CTALede         string   `json:"cta_lede"`
	CTAFine         string   `json:"cta_fine"`
	FooterName      string   `json:"footer_name"`
	Portrait        string   `json:"portrait"`
	Cover           string   `json:"cover"`
	Socials         []Social `json:"socials"`
}

type Schedule struct {
	Weekday int    `json:"weekday"`
	Time    string `json:"time"`
	Zone    string `json:"zone"`
}

type Key struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
	Token  string `json:"token,omitempty"`
}

// Rename is an essay a write renamed. A future essay's slug carries its publish date, so a write that moves
// essays renames them.
type Rename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// QueueAnswer is the site's answer to an essay write: the essay it wrote, when there is one, and the queue,
// the weekly slot, and the renames the write left. Posts is nil when an older site answered without them.
type QueueAnswer struct {
	Post
	Posts    []Post    `json:"posts"`
	Schedule *Schedule `json:"schedule"`
	Renamed  []Rename  `json:"renamed"`
}

// ScheduleAnswer is the weekly slot the site saved, and the queue it laid out on it.
type ScheduleAnswer struct {
	Schedule
	Posts   []Post   `json:"posts"`
	Renamed []Rename `json:"renamed"`
}

// PageAnswer is the page a write saved, when there is one, and every page after the write. Pages is nil when
// the site answered without them.
type PageAnswer struct {
	Page
	Pages []Page `json:"pages"`
}

// KeyAnswer is the key a write created, with its token, and every key after the write. Keys is nil when the
// site answered without them.
type KeyAnswer struct {
	Key
	Keys []Key `json:"keys"`
}

type APIError struct {
	Status int
	Detail string
}

func (e *APIError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("the site returned %d", e.Status)
	}
	return e.Detail
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 100 * time.Second}
}

func (c *Client) GetLanding() (Landing, error) {
	var landing Landing
	err := c.do(http.MethodGet, "/api/v1/landing", nil, &landing)
	return landing, err
}

func (c *Client) PutLanding(landing Landing) (Landing, error) {
	if landing.Quotes == nil {
		landing.Quotes = []Quote{}
	}
	if landing.Socials == nil {
		landing.Socials = []Social{}
	}
	var out Landing
	err := c.do(http.MethodPut, "/api/v1/landing", landing, &out)
	return out, err
}

func (c *Client) ListPosts() ([]Post, error) {
	var posts []Post
	err := c.do(http.MethodGet, "/api/v1/posts", nil, &posts)
	if posts == nil {
		posts = []Post{}
	}
	return posts, err
}

func (c *Client) CreatePost(title, body string) (QueueAnswer, error) {
	var out QueueAnswer
	err := c.do(http.MethodPost, "/api/v1/posts", map[string]string{"title": title, "body": body}, &out)
	return out, err
}

func (c *Client) PutPost(slug, title, body string) (QueueAnswer, error) {
	var out QueueAnswer
	err := c.do(http.MethodPut, "/api/v1/posts/"+slug, map[string]string{"title": title, "body": body}, &out)
	return out, err
}

func (c *Client) DeletePost(slug string) (QueueAnswer, error) {
	var out QueueAnswer
	err := c.do(http.MethodDelete, "/api/v1/posts/"+slug, nil, &out)
	return out, err
}

func (c *Client) MovePost(slug, way string) (QueueAnswer, error) {
	var out QueueAnswer
	err := c.do(http.MethodPost, "/api/v1/posts/"+slug+"/"+way, map[string]string{}, &out)
	return out, err
}

func (c *Client) Proof(title, body string) (string, string, error) {
	var out struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	err := c.do(http.MethodPost, "/api/v1/proof", map[string]string{"title": title, "body": body}, &out)
	return out.Title, out.Body, err
}

func (c *Client) ListPages() ([]Page, error) {
	var pages []Page
	err := c.do(http.MethodGet, "/api/v1/pages", nil, &pages)
	if pages == nil {
		pages = []Page{}
	}
	return pages, err
}

func (c *Client) PutPage(page Page) (PageAnswer, error) {
	var out PageAnswer
	err := c.do(http.MethodPut, "/api/v1/pages/"+page.Slug, page, &out)
	return out, err
}

func (c *Client) DeletePage(slug string) (PageAnswer, error) {
	var out PageAnswer
	err := c.do(http.MethodDelete, "/api/v1/pages/"+slug, nil, &out)
	return out, err
}

func (c *Client) GetSchedule() (Schedule, error) {
	var sched Schedule
	err := c.do(http.MethodGet, "/api/v1/schedule", nil, &sched)
	return sched, err
}

func (c *Client) PutSchedule(sched Schedule) (ScheduleAnswer, error) {
	var out ScheduleAnswer
	err := c.do(http.MethodPut, "/api/v1/schedule", sched, &out)
	return out, err
}

// PutMedia answers the landing page the photo was saved into, or nil when the site answered without it.
func (c *Client) PutMedia(name string, raw []byte) (*Landing, error) {
	var out *Landing
	err := c.doRaw(http.MethodPut, "/api/v1/media/"+name, "application/octet-stream", bytes.NewReader(raw), &out)
	return out, err
}

func (c *Client) ListKeys() ([]Key, error) {
	var keys []Key
	err := c.do(http.MethodGet, "/api/v1/keys", nil, &keys)
	if keys == nil {
		keys = []Key{}
	}
	return keys, err
}

func (c *Client) CreateKey(name string) (KeyAnswer, error) {
	var out KeyAnswer
	err := c.do(http.MethodPost, "/api/v1/keys", map[string]string{"name": name}, &out)
	return out, err
}

func (c *Client) DeleteKey(id int64) (KeyAnswer, error) {
	var out KeyAnswer
	err := c.do(http.MethodDelete, fmt.Sprintf("/api/v1/keys/%d", id), nil, &out)
	return out, err
}

func (c *Client) do(method, path string, in any, out any) error {
	var body io.Reader
	kind := ""
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
		kind = "application/json"
	}
	return c.doRaw(method, path, kind, body, out)
}

func (c *Client) doRaw(method, path, kind string, body io.Reader, out any) error {
	if closer, ok := body.(io.ReadCloser); ok {
		defer closer.Close()
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.Origin, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if kind != "" {
		req.Header.Set("Content-Type", kind)
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return fmt.Errorf("could not reach the site")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("could not read the site")
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiErr(resp.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if json.Unmarshal(raw, out) != nil {
		return fmt.Errorf("the site returned unexpected JSON")
	}
	return nil
}

// missingAction is true when the site has no such route, as an older site
// answers the proof action.
func missingAction(err error) bool {
	var api *APIError
	return errors.As(err, &api) && (api.Status == http.StatusNotFound || api.Status == http.StatusMethodNotAllowed)
}

func apiErr(status int, raw []byte) error {
	var body struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && strings.TrimSpace(body.Error) != "" {
		return &APIError{Status: status, Detail: body.Error}
	}
	return &APIError{Status: status}
}
