package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/ryangerardwilson/musica/player"
)

// runMusica is r2d2 musica. With no arguments it shows the Tesseract Musica
// center through the launcher's open.sh musica, as bare musica does. It does
// not call player.Run for that: the pinned musica module opens the terminal
// player with no arguments, and a terminal holding the musica session is what
// the center has to take over. end, play, open, config, help, and
// download-worker pass to player.Run unchanged.
func runMusica(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return openMusicaCenter(stderr)
	}
	return player.Run(args, os.Stdin, stdout, stderr)
}

// startMusicaCenter starts the launcher script in its own session and does
// not wait for it, so the launcher stays up after r2d2 returns. Tests replace
// it.
var startMusicaCenter = func(script string) error {
	cmd := exec.Command(script, "musica")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// musicaCenterScript is <config>/quickshell/omarchy-launcher/open.sh,
// <config> being XDG_CONFIG_HOME when it is absolute and ~/.config otherwise.
func musicaCenterScript() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "quickshell", "omarchy-launcher", "open.sh"), nil
}

func openMusicaCenter(stderr io.Writer) int {
	script, err := musicaCenterScript()
	if err == nil {
		err = startMusicaCenter(script)
	}
	if err != nil {
		fmt.Fprintf(stderr, "The Tesseract Musica center opens with %s musica: %v\n", script, err)
		return 1
	}
	return 0
}
