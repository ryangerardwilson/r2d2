package web

import (
	"bytes"
	"encoding/json"
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
	PublishedAt *time.Time `json:"published_at"`
	Minutes     int        `json:"minutes"`
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

type Home struct {
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
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Prefix  string `json:"prefix"`
	Created string `json:"created_at"`
	Token   string `json:"token,omitempty"`
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

func (c *Client) GetLanding() (Home, error) {
	var home Home
	err := c.do(http.MethodGet, "/api/v1/landing", nil, &home)
	return home, err
}

func (c *Client) PutLanding(home Home) (Home, error) {
	if home.Quotes == nil {
		home.Quotes = []Quote{}
	}
	if home.Socials == nil {
		home.Socials = []Social{}
	}
	var out Home
	err := c.do(http.MethodPut, "/api/v1/landing", home, &out)
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

func (c *Client) CreatePost(title, body string) (Post, error) {
	var post Post
	err := c.do(http.MethodPost, "/api/v1/posts", map[string]string{"title": title, "body": body}, &post)
	return post, err
}

func (c *Client) PutPost(slug, title, body string) (Post, error) {
	var post Post
	err := c.do(http.MethodPut, "/api/v1/posts/"+slug, map[string]string{"title": title, "body": body}, &post)
	return post, err
}

func (c *Client) DeletePost(slug string) error {
	return c.do(http.MethodDelete, "/api/v1/posts/"+slug, nil, nil)
}

func (c *Client) MovePost(slug, way string) error {
	return c.do(http.MethodPost, "/api/v1/posts/"+slug+"/"+way, map[string]string{}, nil)
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

func (c *Client) PutPage(page Page) (Page, error) {
	var out Page
	err := c.do(http.MethodPut, "/api/v1/pages/"+page.Slug, page, &out)
	return out, err
}

func (c *Client) DeletePage(slug string) error {
	return c.do(http.MethodDelete, "/api/v1/pages/"+slug, nil, nil)
}

func (c *Client) GetSchedule() (Schedule, error) {
	var sched Schedule
	err := c.do(http.MethodGet, "/api/v1/schedule", nil, &sched)
	return sched, err
}

func (c *Client) PutSchedule(sched Schedule) (Schedule, error) {
	var out Schedule
	err := c.do(http.MethodPut, "/api/v1/schedule", sched, &out)
	return out, err
}

func (c *Client) PutMedia(name string, raw []byte) error {
	return c.doRaw(http.MethodPut, "/api/v1/media/"+name, "application/octet-stream", bytes.NewReader(raw), nil)
}

func (c *Client) ListKeys() ([]Key, error) {
	var keys []Key
	err := c.do(http.MethodGet, "/api/v1/keys", nil, &keys)
	if keys == nil {
		keys = []Key{}
	}
	return keys, err
}

func (c *Client) CreateKey(name string) (Key, error) {
	var key Key
	err := c.do(http.MethodPost, "/api/v1/keys", map[string]string{"name": name}, &key)
	return key, err
}

func (c *Client) DeleteKey(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/v1/keys/%d", id), nil, nil)
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

func apiErr(status int, raw []byte) error {
	var body struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && strings.TrimSpace(body.Error) != "" {
		return &APIError{Status: status, Detail: body.Error}
	}
	return &APIError{Status: status}
}
