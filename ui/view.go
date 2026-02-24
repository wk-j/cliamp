package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	panelWidth        = 60 // usable inner width (66 frame - 2 border - 4 padding)
	miniPanelMinW     = 28 // minimum usable inner width for mini mode
	miniFrameOverhead = 4  // border (2) + padding (2×1) for mini frame
)

// Pre-built styles for elements created per-render to avoid repeated allocation.
var (
	seekFillStyle = lipgloss.NewStyle().Foreground(colorSeekBar)
	seekDimStyle  = lipgloss.NewStyle().Foreground(colorDim)
	volBarStyle   = lipgloss.NewStyle().Foreground(colorVolume)
	activeToggle  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
)

// pw returns the usable inner panel width for the current mode.
func (m Model) pw() int {
	if m.mini {
		w := m.width - miniFrameOverhead
		if w < miniPanelMinW {
			w = miniPanelMinW
		}
		return w
	}
	return panelWidth
}

// miniFrameW returns the outer frame width for mini mode.
func (m Model) miniFrameW() int {
	w := m.width
	if w < miniPanelMinW+miniFrameOverhead {
		w = miniPanelMinW + miniFrameOverhead
	}
	return w
}

// View renders the full TUI frame.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sections []string
	if m.mini {
		sections = []string{
			m.renderTitle(),
			m.renderTrackInfo(),
			m.renderTimeStatus(),
			m.renderSpectrum(),
			m.renderSeekBar(),
			m.renderVolume(),
			m.renderPlaylistHeader(),
			m.renderPlaylist(),
			m.renderHelp(),
		}
	} else {
		sections = []string{
			m.renderTitle(),
			m.renderTrackInfo(),
			m.renderTimeStatus(),
			"",
			m.renderSpectrum(),
			m.renderSeekBar(),
			"",
			m.renderVolume(),
			m.renderEQ(),
			"",
			m.renderPlaylistHeader(),
			m.renderPlaylist(),
			"",
			m.renderHelp(),
		}
	}

	if m.err != nil {
		sections = append(sections, errorStyle.Render(fmt.Sprintf("ERR: %s", m.err)))
	}

	content := strings.Join(sections, "\n")
	if m.mini {
		return miniFrameStyle.Width(m.miniFrameW()).Render(content)
	}
	return frameStyle.Render(content)
}

func (m Model) renderTitle() string {
	return titleStyle.Render("C L I A M P")
}

func (m Model) renderTrackInfo() string {
	track, _ := m.playlist.Current()
	name := track.DisplayName()
	if name == "" {
		name = "No track loaded"
	}

	pw := m.pw()
	prefix := "\U000f0e1e "
	if m.mini {
		prefix = "♫ "
	}
	maxW := pw - len([]rune(prefix))
	runes := []rune(name)

	if len(runes) <= maxW {
		return trackStyle.Render(prefix + name)
	}

	// Cyclic scrolling for long titles
	sep := []rune("   \U000f0e1e   ")
	if m.mini {
		sep = []rune("  ♫  ")
	}
	padded := append(runes, sep...)
	total := len(padded)
	off := m.titleOff % total

	display := make([]rune, maxW)
	for i := range maxW {
		display[i] = padded[(off+i)%total]
	}
	return trackStyle.Render(prefix + string(display))
}

func (m Model) renderTimeStatus() string {
	pos := m.player.Position()
	dur := m.player.Duration()

	posMin := int(pos.Minutes())
	posSec := int(pos.Seconds()) % 60
	durMin := int(dur.Minutes())
	durSec := int(dur.Seconds()) % 60

	timeStr := fmt.Sprintf("%02d:%02d / %02d:%02d", posMin, posSec, durMin, durSec)

	var status string
	if m.mini {
		switch {
		case m.player.IsPlaying() && m.player.IsPaused():
			status = statusStyle.Render("\uf04c")
		case m.player.IsPlaying():
			status = statusStyle.Render("\uf04b")
		default:
			status = dimStyle.Render("\uf04d")
		}
	} else {
		switch {
		case m.player.IsPlaying() && m.player.IsPaused():
			status = statusStyle.Render("\uf04c Paused")
		case m.player.IsPlaying():
			status = statusStyle.Render("\uf04b Playing")
		default:
			status = dimStyle.Render("\uf04d Stopped")
		}
	}

	left := timeStyle.Render(timeStr)
	gap := m.pw() - lipgloss.Width(left) - lipgloss.Width(status)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + status
}

func (m Model) renderSpectrum() string {
	bands := m.vis.Analyze(m.player.Samples())
	if m.mini {
		return m.vis.RenderDynamic(bands, m.pw())
	}
	return m.vis.Render(bands)
}

func (m Model) renderSeekBar() string {
	pos := m.player.Position()
	dur := m.player.Duration()

	var progress float64
	if dur > 0 {
		progress = float64(pos) / float64(dur)
	}
	progress = max(0, min(1, progress))

	pw := m.pw()
	filled := int(progress * float64(pw-1))

	return seekFillStyle.Render(strings.Repeat("━", filled)) +
		seekFillStyle.Render("●") +
		seekDimStyle.Render(strings.Repeat("━", max(0, pw-filled-1)))
}

func (m Model) renderVolume() string {
	vol := m.player.Volume()
	frac := max(0, min(1, (vol+30)/36))

	if m.mini {
		// "V " (2) + bar + " -30" (4) = 6 overhead
		barW := m.pw() - 6
		if barW < 4 {
			barW = 4
		}
		filled := int(frac * float64(barW))
		bar := volBarStyle.Render(strings.Repeat("█", filled)) +
			dimStyle.Render(strings.Repeat("░", barW-filled))
		return labelStyle.Render("V ") + bar + dimStyle.Render(fmt.Sprintf(" %+.0f", vol))
	}

	barW := 22
	filled := int(frac * float64(barW))
	bar := volBarStyle.Render(strings.Repeat("█", filled)) +
		dimStyle.Render(strings.Repeat("░", barW-filled))
	return labelStyle.Render("VOL ") + bar + dimStyle.Render(fmt.Sprintf(" %+.1fdB", vol))
}

func (m Model) renderEQ() string {
	bands := m.player.EQBands()
	labels := [10]string{"70", "180", "320", "600", "1k", "3k", "6k", "12k", "14k", "16k"}

	parts := make([]string, len(labels))
	for i, label := range labels {
		style := eqInactiveStyle
		if m.focus == focusEQ && i == m.eqCursor {
			style = eqActiveStyle
			label = fmt.Sprintf("%+.0f", bands[i])
		}
		parts[i] = style.Render(label)
	}

	return labelStyle.Render("EQ  ") + strings.Join(parts, " ")
}

func (m Model) renderPlaylistHeader() string {
	var shuffle string
	if m.playlist.Shuffled() {
		shuffle = activeToggle.Render("[S]")
	} else {
		shuffle = dimStyle.Render("[S]")
	}

	if m.mini {
		var repeat string
		if m.playlist.Repeat() != 0 {
			repeat = activeToggle.Render(fmt.Sprintf("[R:%s]", m.playlist.Repeat()))
		} else {
			repeat = dimStyle.Render("[R]")
		}
		return dimStyle.Render("─ Playlist ─ ") + shuffle + " " + repeat
	}

	if m.playlist.Shuffled() {
		shuffle = activeToggle.Render("[Shuffle]")
	} else {
		shuffle = dimStyle.Render("[Shuffle]")
	}

	repeatStr := fmt.Sprintf("[Repeat: %s]", m.playlist.Repeat())
	if m.playlist.Repeat() != 0 {
		repeatStr = activeToggle.Render(repeatStr)
	} else {
		repeatStr = dimStyle.Render(repeatStr)
	}

	return dimStyle.Render("── Playlist ── ") + shuffle + " " + repeatStr + " " + dimStyle.Render("──")
}

func (m Model) renderPlaylist() string {
	tracks := m.playlist.Tracks()
	if len(tracks) == 0 {
		return dimStyle.Render("  No tracks loaded")
	}

	currentIdx := m.playlist.Index()
	visible := min(m.plVisible, len(tracks))

	scroll := m.plScroll
	if scroll+visible > len(tracks) {
		scroll = len(tracks) - visible
	}
	scroll = max(0, scroll)

	lines := make([]string, 0, visible)
	for i := scroll; i < scroll+visible && i < len(tracks); i++ {
		prefix := "  "
		style := playlistItemStyle

		if i == currentIdx && m.player.IsPlaying() {
			prefix = "\uf04b "
			style = playlistActiveStyle
		}

		if m.focus == focusPlaylist && i == m.plCursor {
			style = playlistSelectedStyle
		}

		name := tracks[i].DisplayName()
		maxW := m.pw() - 6
		nameRunes := []rune(name)
		if len(nameRunes) > maxW {
			name = string(nameRunes[:maxW-1]) + "…"
		}

		lines = append(lines, style.Render(fmt.Sprintf("%s%d. %s", prefix, i+1, name)))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderHelp() string {
	if m.mini {
		return helpStyle.Render("[Spc]Play [<>]Trk [Q]Quit")
	}
	return helpStyle.Render("[Spc]\U000f040e  [<>]Trk [\uf060\uf061]Seek [+-]Vol [Tab]Focus [Q]Quit")
}
