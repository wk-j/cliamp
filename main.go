// Package main is the entry point for the CLIAMP terminal music player.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/v2"

	"winamp-cli/player"
	"winamp-cli/playlist"
	"winamp-cli/ui"
)

func run() error {
	autoPlay := flag.Bool("autoplay", false, "start playing the first track immediately")
	mini := flag.Bool("mini", false, "compact minimal UI with less width")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: cliamp [flags] <file.mp3> [file2.mp3 ...]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return errors.New("usage: cliamp [--autoplay] <file.mp3> [file2.mp3 ...]")
	}

	// Expand shell globs that may not have been expanded by the shell
	var files []string
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err != nil || len(matches) == 0 {
			files = append(files, arg)
		} else {
			files = append(files, matches...)
		}
	}

	// Build playlist from file arguments
	pl := playlist.New()
	for _, f := range files {
		pl.Add(playlist.TrackFromPath(f))
	}

	// Initialize audio engine at CD-quality sample rate
	sr := beep.SampleRate(44100)
	p := player.New(sr)
	defer p.Close()

	// Launch the TUI
	m := ui.NewModel(p, pl, *autoPlay, *mini)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
