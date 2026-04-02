package restore

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/InfantAjayVenus/trash-rm/internal/log"
)

// BubbleteaSelectFunc returns a SelectFunc backed by a bubbletea interactive list.
// It presents entries in reverse-chronological order and returns the index of
// the chosen entry within the original entries slice (not the reversed display order).
func BubbleteaSelectFunc(in io.Reader, out io.Writer) SelectFunc {
	return func(entries []log.LogEntry) (int, error) {
		return runTUI(entries, in, out)
	}
}

// entryItem wraps a log.LogEntry for the bubbles list component.
type entryItem struct {
	entry log.LogEntry
	index int // index in the original (non-reversed) entries slice
}

func (e entryItem) Title() string {
	t, err := time.Parse(time.RFC3339, e.entry.Timestamp)
	if err != nil {
		return fmt.Sprintf("[%s] %s  (cwd: %s)", e.entry.Timestamp, e.entry.Command, e.entry.CWD)
	}
	return fmt.Sprintf("[%s] %s  (cwd: %s)", t.Format("2006-01-02 15:04:05"), e.entry.Command, e.entry.CWD)
}

func (e entryItem) Description() string { return "" }
func (e entryItem) FilterValue() string  { return e.entry.Command }

// tuiModel is the bubbletea application model.
type tuiModel struct {
	list     list.Model
	chosen   int
	quitting bool
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			m.chosen = -1
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(entryItem); ok {
				m.chosen = item.index
			}
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func runTUI(entries []log.LogEntry, in io.Reader, out io.Writer) (int, error) {
	items := buildListItems(entries)

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 80, 20)
	l.Title = "Trash History — select an entry to restore (q/Esc to quit)"

	m := tuiModel{list: l, chosen: -1}

	opts := []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(out)}
	p := tea.NewProgram(m, opts...)
	final, err := p.Run()
	if err != nil {
		return -1, fmt.Errorf("TUI error: %w", err)
	}

	result := final.(tuiModel)
	return result.chosen, nil
}

// buildListItems converts entries to list items in reverse-chronological order.
func buildListItems(entries []log.LogEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i := range entries {
		originalIndex := len(entries) - 1 - i
		items[i] = entryItem{entry: entries[originalIndex], index: originalIndex}
	}
	return items
}
