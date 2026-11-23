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
// paint draws frame. A positive cursor row places a steady block cursor
// at that 1-based cell. A zero row hides the cursor.
type screen interface {
	size() (width, height int)
	paint(frame string, cursorRow, cursorCol int)
	readKey() (string, error)
}

// model is the skill list over the working copy.
type model struct {
	repo       *repo
	rows       []string
	cursor     int
	help       bool
	note       string // the clone's state
	broken     error  // set when the directory is not the skills repository
	writable   bool
	status     string
	alarm      bool    // status is an error
	naming     *string // the name typed after n or rn; nil when not asking
	renameFrom string  // the skill rn is renaming; empty when creating
	removing   string  // the skill d asked about; empty when not asking
	leader     string  // r while waiting for the second key of rn or rr
	width      int
	height     int
	lookVim    func() (string, error)
	openVim    func(vim, path string) error
	// busy paints a frame that names the git step running, before it blocks.
	busy func(text string)
	// hosts install the skills plugin from main; run runs their commands.
	hosts []host
	run   runFunc
}

// loop paints, reads one key, and handles it until the operator quits.
func (m *model) loop(term screen) error {
	m.busy = func(text string) {
		m.status, m.alarm = text, false
		m.width, m.height = term.size()
		row, col := m.blockCursor()
		term.paint(m.view(), row, col)
	}
	m.busy("syncing")
	m.apply(m.repo.open(context.Background()))
	m.status = ""
	for {
		m.width, m.height = term.size()
		row, col := m.blockCursor()
		term.paint(m.view(), row, col)
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
	case m.leader != "":
		m.leaderKey(key)
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
		m.leader, m.status, m.alarm = "r", "", false
	}
	m.cursor = m.clamp()
	return false
}

// leaderKey takes the key after r. n renames, r refreshes, and any other
// key cancels.
func (m *model) leaderKey(key string) {
	m.leader = ""
	switch key {
	case "n":
		m.startRename()
	case "r":
		m.refresh()
	}
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
	m.naming, m.renameFrom, m.status, m.alarm = &name, "", "", false
}

// startRename asks for a new name of the selected skill. The field starts
// as the current name.
func (m *model) startRename() {
	if !m.canWrite() {
		return
	}
	name, ok := m.selected()
	if !ok {
		return
	}
	typed := name
	m.renameFrom, m.naming, m.status, m.alarm = name, &typed, "", false
}

// typeName edits the name after n or rn. Enter checks it, and a rejected
// name stays in the field with the reason. Esc cancels.
func (m *model) typeName(key string) {
	name := *m.naming
	switch {
	case key == "esc" || key == "ctrl+c":
		m.naming, m.renameFrom, m.status, m.alarm = nil, "", "", false
	case key == "enter":
		if m.renameFrom != "" {
			m.rename(name)
		} else {
			m.create(name)
		}
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

// create writes an empty SKILL.md, opens it in vim, and commits whatever
// vim saved unless the file was left empty.
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

// rename moves the selected skill to name. The same name commits nothing.
// A rejected name stays in the field.
func (m *model) rename(name string) {
	from := m.renameFrom
	if name == from {
		m.naming, m.renameFrom = nil, ""
		m.status, m.alarm = "no change to "+from, false
		return
	}
	if err := m.checkName(name); err != nil {
		m.status, m.alarm = err.Error(), true
		return
	}
	m.naming, m.renameFrom = nil, ""
	message := "Rename " + from + " to " + name
	m.commitAndPush(name, message, func(ctx context.Context) error {
		return m.repo.move(ctx, from, name, message)
	})
}

// save commits the skill's paths and pushes main. A failed push keeps the
// commit and says so.
func (m *model) save(name, message string) {
	m.commitAndPush(name, message, func(ctx context.Context) error {
		return m.repo.commit(ctx, name, message)
	})
}

// commitAndPush runs commit, reloads onto stay, pushes main, and installs
// the pushed commit for every host. Only then does the status say pushed.
func (m *model) commitAndPush(stay, message string, commit func(context.Context) error) {
	ctx := context.Background()
	m.busy("committing " + message)
	err := commit(ctx)
	m.reload(stay)
	if errors.Is(err, errNothing) {
		m.status, m.alarm = "nothing to commit for "+stay, false
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
	m.install(ctx, message)
}

// install refreshes every host's installed plugin and checks it holds the
// skills of the commit just pushed. A failure names the host and leaves
// the commit on main.
func (m *model) install(ctx context.Context, message string) {
	want, err := m.repo.tree(ctx)
	if err != nil {
		m.status, m.alarm = message+" is on main, not installed: "+err.Error(), true
		return
	}
	var failed, names []string
	for _, h := range m.hosts {
		names = append(names, h.name)
		m.busy("installing " + message + " for " + h.name)
		if err := refreshHost(ctx, m.run, h, want); err != nil {
			failed = append(failed, h.name+": "+err.Error())
		}
	}
	if len(failed) > 0 {
		m.status, m.alarm = message+" is on main, not installed for "+strings.Join(failed, "; "), true
		return
	}
	m.status, m.alarm = "pushed "+message+"; installed for "+strings.Join(names, ", ")+"; restart them to load it", false
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
		line := m.namePrefix() + *m.naming
		if m.status != "" {
			line += "  " + m.status
		}
		return line, ""
	case m.leader == "r":
		return "n rename, r refresh", styleActive
	case m.removing != "":
		return "remove " + m.removing + "? y/n", styleActive
	case m.status == "":
		return "", ""
	case m.alarm:
		return m.status, styleError
	}
	return m.status, styleMuted
}

// blockCursor is the steady block at the end of the name being typed.
// A zero row hides the cursor.
func (m *model) blockCursor() (row, col int) {
	if m.naming == nil || m.help {
		return 0, 0
	}
	lines := strings.Split(strings.TrimSuffix(m.view(), "\n"), "\n")
	row = len(lines)
	col = len(m.namePrefix()) + len(*m.naming) + 1
	if m.width > 0 && col > m.width {
		col = m.width
	}
	if col < 1 {
		col = 1
	}
	return row, col
}

// namePrefix is the label in front of the name being typed.
func (m *model) namePrefix() string {
	if m.renameFrom != "" {
		return "rename " + m.renameFrom + ": "
	}
	return "new skill: "
}

func helpView() string {
	lines := []string{
		paintStyle(styleTitle, "skills"),
		"",
		"j/k    move",
		"g/G    first, last",
		"enter  edit in vim",
		"n      new skill",
		"rn     rename skill",
		"d      remove skill",
		"rr     fetch and fast-forward",
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
