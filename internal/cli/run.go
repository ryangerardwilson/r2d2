package cli

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/r2d2/internal/menu"
	"github.com/ryangerardwilson/r2d2/internal/screen"
	"github.com/ryangerardwilson/r2d2/internal/version"
	"github.com/ryangerardwilson/r2d2/internal/web"
)

// Help is the r2d2 help. A terminal opens the menu instead of printing it.
const Help = `r2d2 is the terminal program. c3po is the agent CLI.

  r2d2
    open the screen menu
  r2d2 web
    ryangerardwilson.com admin panel
  r2d2 cyclotree [symbol] [dir]
  r2d2 l [path] [flags]
  r2d2 o [args...]
  r2d2 tunnel
  r2d2 gdrive
  r2d2 pass
  r2d2 deport
  r2d2 xl [file]
  r2d2 looptab
  r2d2 slack
  r2d2 version
  r2d2 help

screens:
  web          ryangerardwilson.com admin panel
  cyclotree    calls of one function, as a tree you move with hjkl
  l            compact colored tree of a directory
  o            vim-inspired file navigator
  tunnel       Cloudflare Tunnel route editor
  gdrive       Drive preset, then browse
  pass         1Password vaults, then copy one item
  deport       occupied external ports, freed with dd
  xl           spreadsheet for a budget workbook
  looptab      Looptab jobs, run one now
  slack        Slack workspaces, then a conversation

A pipe prints this text. A terminal opens the menu. j and k move, Enter
opens the screen, and q quits.
r2d2 web reads the admin API key with nebula crypt. Essays open in vim
with :Wrap, and the saved text is a flowing paragraph.
r2d2 <screen> help explains one screen.
`

// Run dispatches one screen.
func Run(args []string, stdout, stderr io.Writer) int {
	rest := []string{}
	if len(args) > 0 {
		rest = args[1:]
	}
	if len(rest) == 0 {
		if code, opened := menu.Run(stdout, stderr); opened {
			return code
		}
		fmt.Fprint(stdout, Help)
		return 0
	}
	if helpWord(rest[0]) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if versionWord(rest[0]) && len(rest) == 1 {
		fmt.Fprintf(stdout, "r2d2 %s\n", version.String())
		return 0
	}
	name := rest[0]
	if name == "web" {
		return web.Run(rest[1:], stdout, stderr)
	}
	if !screen.Known(name) {
		fmt.Fprintf(stderr, "unknown screen %q\n", name)
		fmt.Fprint(stderr, Help)
		return 2
	}
	return screen.Launch(name, rest[1:], stdout, stderr)
}

func helpWord(word string) bool {
	return word == "help" || word == "-h" || word == "--help"
}

func versionWord(word string) bool {
	return word == "version" || word == "--version" || word == "-v"
}
