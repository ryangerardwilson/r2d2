package skills

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[7m"
	styleReset  = "\x1b[0m"
	maxName     = 64
)

var (
	errNoVim   = errors.New("vim is not on PATH")
	namePat    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	errBadName = errors.New("a name is lowercase letters, digits, and hyphens, and does not start or end with a hyphen")
)

// screen is the terminal the list draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

// model is the skill list over the working copy.
type model struct {
	repo     *repo
	rows     []string
	cursor   int
	help     bool
	note     string // the clone's state
	broken   error  // set when the directory is not the skills repository
	writable bool
	status   string
	alarm    bool    // status is an error
	naming   *string // the name typed after n; nil when not asking
	removing string  // the skill d asked about; empty when not asking
	width    int
	height   int
	lookVim  func() (string, error)
	openVim  func(vim, path string) error
	// busy paints a frame that names the git step running, before it blocks.
	busy func(text string)
}

// loop paints, reads one key, and handles it until the operator quits.
func (m *model) loop(term screen) error {
	m.busy = func(text string) {
		m.status, m.alarm = text, false
		m.width, m.height = term.size()
		term.paint(m.view())
	}
	m.busy("syncing")
	m.apply(m.repo.open(context.Background()))
	m.status = ""
	for {
		m.width, m.height = term.size()
		term.paint(m.view())
		key, err := term.readKey()
		if err != nil {
			return err
		}
		if m.handleKey(key) {
			return nil
		}
	}
}

// apply takes the clone's state and lists the skills.
func (m *model) apply(st state) {
	m.note, m.writable, m.broken = st.note, st.writable, st.broken
	if m.broken != nil {
		m.rows, m.cursor = nil, 0
		return
	}
	m.reload("")
}

// reload lists the skills again and keeps the cursor on name when it is
// listed, or on the row it was on.
func (m *model) reload(name string) {
	rows, err := m.repo.skills()
	if err != nil {
		m.status, m.alarm = err.Error(), true
	}
	m.rows = rows
	for i, row := range rows {
		if row == name {
			m.cursor = i
		}
	}
	m.cursor = m.clamp()
}

func (m *model) handleKey(key string) bool {
	switch {
	case m.naming != nil:
		m.typeName(key)
	case m.removing != "":
		m.answer(key)
	case m.help:
		return m.helpKey(key)
	default:
		return m.listKey(key)
	}
	return false
}

func (m *model) helpKey(key string) bool {
	switch key {
	case "?", "esc":
		m.help = false
	case "ctrl+c", "q":
		return true
	}
	return false
}

func (m *model) listKey(key string) bool {
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "?":
		m.help, m.status = true, ""
	case "j", "down":
		m.cursor = cycled(m.cursor, 1, len(m.rows))
	case "k", "up":
		m.cursor = cycled(m.cursor, -1, len(m.rows))
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows) - 1
	case "enter":
		m.edit()
	case "n":
		m.startNew()
	case "d":
		m.startRemove()
	case "r":
		m.refresh()
	}
	m.cursor = m.clamp()
	return false
}

// canWrite is false, with the reason in the status, when no commit may be made.
func (m *model) canWrite() bool {
	switch {
	case m.broken != nil:
		m.status, m.alarm = m.broken.Error(), true
	case !m.writable:
		m.status, m.alarm = "writes are off: "+m.note, true
	default:
		return true
	}
	return false
}

func (m *model) selected() (string, bool) {
	if len(m.rows) == 0 {
		m.status, m.alarm = "no skills", false
		return "", false
	}
	return m.rows[m.clamp()], true
}

func (m *model) startNew() {
	if !m.canWrite() {
		return
	}
	name := ""
	m.naming, m.status, m.alarm = &name, "", false
}

// typeName edits the name after n. Enter checks it, and a rejected name
// stays in the field with the reason. Esc cancels.
func (m *model) typeName(key string) {
	name := *m.naming
	switch {
	case key == "esc" || key == "ctrl+c":
		m.naming, m.status, m.alarm = nil, "", false
	case key == "enter":
		m.create(name)
	case key == "backspace":
		if name != "" {
			name = name[:len(name)-1]
		}
	case len(key) == 1 && key[0] >= ' ' && key[0] <= '~' && len(name) < maxName+1:
		name += key
	}
	if m.naming != nil {
		*m.naming = name
	}
}

// checkName is nil for a name that may become a new skill.
func (m *model) checkName(name string) error {
	if name == "" {
		return errors.New("type a name")
	}
	if len(name) > maxName {
		return fmt.Errorf("a name is at most %d characters", maxName)
	}
	if !namePat.MatchString(name) {
		return errBadName
	}
	if _, err := os.Lstat(m.repo.skillDir(name)); err == nil {
		return fmt.Errorf("%s already exists", name)
	}
	return nil
}

// create writes a minimal SKILL.md, opens it in vim, and commits it unless
// it was emptied.
func (m *model) create(name string) {
	if err := m.checkName(name); err != nil {
		m.status, m.alarm = err.Error(), true
		return
	}
	m.naming = nil
	vim, err := m.lookVim()
	if err != nil {
		m.status, m.alarm = errNoVim.Error(), true
		return
	}
	path, err := m.repo.scaffold(name)
	if err != nil {
		m.status, m.alarm = err.Error(), true
		return
	}
	if err := m.openVim(vim, path); err != nil {
		m.status, m.alarm = "vim: "+err.Error(), true
	}
	if body, err := os.ReadFile(path); err != nil || len(bytes.TrimSpace(body)) == 0 {
		_ = os.RemoveAll(m.repo.skillDir(name))
		m.reload("")
		m.status, m.alarm = name+" was empty; nothing added", false
		return
	}
	m.save(name, "Add "+name)
}

// edit opens the selected SKILL.md in vim and commits it when it changed.
func (m *model) edit() {
	name, ok := m.selected()
	if !ok || !m.canWrite() {
		return
	}
	vim, err := m.lookVim()
	if err != nil {
		m.status, m.alarm = errNoVim.Error(), true
		return
	}
	path := m.repo.skillFile(name)
	before, _ := os.ReadFile(path)
	if err := m.openVim(vim, path); err != nil {
		m.status, m.alarm = "vim: "+err.Error(), true
	}
	after, _ := os.ReadFile(path)
	if bytes.Equal(before, after) {
		m.status, m.alarm = "no change to "+name, false
		return
	}
	m.save(name, "Update "+name)
}

func (m *model) startRemove() {
	name, ok := m.selected()
	if !ok || !m.canWrite() {
		return
	}
	m.removing, m.status, m.alarm = name, "", false
}

// answer takes the key after d. Only y removes the skill.
func (m *model) answer(key string) {
	name := m.removing
	m.removing = ""
	if key != "y" {
		m.status, m.alarm = "kept "+name, false
		return
	}
	if err := os.RemoveAll(m.repo.skillDir(name)); err != nil {
		m.status, m.alarm = err.Error(), true
		m.reload(name)
		return
	}
	m.save(name, "Remove "+name)
}

// save commits the skill's paths and pushes main. A failed push keeps the
// commit and says so.
func (m *model) save(name, message string) {
	ctx := context.Background()
	m.busy("committing " + message)
	err := m.repo.commit(ctx, name, message)
	m.reload(name)
	if errors.Is(err, errNothing) {
		m.status, m.alarm = "nothing to commit for "+name, false
		return
	}
	if err != nil {
		m.status, m.alarm = message+" failed: "+err.Error(), true
		return
	}
	m.busy("pushing " + message)
	if err := m.repo.push(ctx); err != nil {
		m.status, m.alarm = message+" is committed, push failed: "+err.Error(), true
		m.note = "unpushed commits; push failed"
		return
	}
	m.status, m.alarm = "pushed "+message, false
}

// refresh fetches and fast-forwards main, or says why it did not.
func (m *model) refresh() {
	if m.broken != nil {
		m.status, m.alarm = m.broken.Error(), true
		return
	}
	name, _ := m.current()
	m.busy("fetching")
	st := m.repo.sync(context.Background())
	m.note, m.writable = st.note, st.writable
	m.reload(name)
	m.status, m.alarm = "", false
}

func (m *model) current() (string, bool) {
	if len(m.rows) == 0 {
		return "", false
	}
	return m.rows[m.clamp()], true
}

func (m *model) clamp() int {
	if len(m.rows) == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= len(m.rows) {
		return len(m.rows) - 1
	}
	return m.cursor
}

func cycled(cursor, delta, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	} else if cursor >= n {
		cursor = n - 1
	}
	return (cursor + delta%n + n) % n
}

func (m *model) view() string {
	if m.help {
		return helpView()
	}
	var b strings.Builder
	title := paintStyle(styleTitle, "skills")
	if m.note != "" {
		title += "  " + paintStyle(styleMuted, truncate(m.note, m.width-8))
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	switch {
	case m.broken != nil:
		b.WriteString(paintStyle(styleError, truncate(m.broken.Error(), m.width)) + "\n")
	case len(m.rows) == 0:
		b.WriteString("no skills\n")
	default:
		m.writeRows(&b)
	}
	if line, style := m.footer(); line != "" {
		fmt.Fprintf(&b, "\n%s\n", paintStyle(style, truncate(line, m.width)))
	}
	return b.String()
}

func (m *model) writeRows(b *strings.Builder) {
	start, end := m.window()
	for i := start; i < end; i++ {
		prefix, style := "  ", ""
		if i == m.clamp() {
			prefix, style = "> ", styleActive
		}
		b.WriteString(paintStyle(style, truncate(prefix+m.rows[i], m.width)))
		b.WriteByte('\n')
	}
}

// footer is the prompt when one is open, or else the status.
func (m *model) footer() (string, string) {
	switch {
	case m.naming != nil:
		line := "new skill: " + *m.naming + "_"
		if m.status != "" {
			line += "  " + m.status
		}
		return line, ""
	case m.removing != "":
		return "remove " + m.removing + "? y/n", styleActive
	case m.status == "":
		return "", ""
	case m.alarm:
		return m.status, styleError
	}
	return m.status, styleMuted
}

func helpView() string {
	lines := []string{
		paintStyle(styleTitle, "skills"),
		"",
		"j/k    move",
		"g/G    first, last",
		"enter  edit in vim",
		"n      new skill",
		"d      remove skill",
		"r      fetch and fast-forward",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window() (int, int) {
	rows := m.height - 4
	if m.height <= 0 || rows < 1 || len(m.rows) <= rows {
		return 0, len(m.rows)
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := start + rows
	if end > len(m.rows) {
		end = len(m.rows)
	}
	return start, end
}

func paintStyle(style, text string) string {
	if style == "" {
		return text
	}
	return style + text + styleReset
}

func truncate(value string, width int) string {
	if width <= 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}
