package ui

import (
	"darker/internal/db"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			Background(lipgloss.Color("#1A1A1A")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF"))

	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#FFD700")).Bold(true)
	descStyle         = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#00FF7F"))
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	activeTagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	offlineTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500"))
)

type item struct {
	site db.Site
}

func (i item) Title() string {
	status := offlineTagStyle.Render("[OFFLINE]")
	if i.site.IsActive {
		status = activeTagStyle.Render("[ONLINE]")
	}
	return status + " " + i.site.Title
}
func (i item) Description() string { return i.site.URL }
func (i item) FilterValue() string { return i.site.Title + " " + i.site.URL }

type Model struct {
	store     *db.Store
	textInput textinput.Model
	list      list.Model
	searching bool
	err       error
	width     int
	height    int
}

func InitialModel(store *db.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter keywords (e.g. 'wiki', 'index')..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 60
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SEARCH RESULTS"
	l.Styles.Title = titleStyle

	return Model{
		store:     store,
		textInput: ti,
		list:      l,
		searching: true,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.searching {
				sites, err := m.store.SearchSites(m.textInput.Value())
				if err != nil {
					m.err = err
					return m, nil
				}
				items := make([]list.Item, len(sites))
				for i, s := range sites {
					items[i] = item{site: s}
				}
				m.list.SetItems(items)
				m.searching = false
				return m, nil
			}
		case "esc":
			if !m.searching {
				m.searching = true
				return m, nil
			}
		case "c":
			if !m.searching {
				if i, ok := m.list.SelectedItem().(item); ok {
					clipboard.WriteAll(i.site.URL)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-12)
	}

	if m.searching {
		m.textInput, cmd = m.textInput.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

const logo = `
 ██████   █████  ██████  ██   ██ ███████ ██████  
 ██   ██ ██   ██ ██   ██ ██  ██  ██      ██   ██ 
 ██   ██ ███████ ██████  █████   █████   ██████  
 ██   ██ ██   ██ ██   ██ ██  ██  ██      ██   ██ 
 ██████  ██   ██ ██   ██ ██   ██ ███████ ██   ██ 
`

func (m Model) View() string {
	var s string
	
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Render(logo)
	s += header + "\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render(" DARK WEB SEARCH ENGINE ") + "\n\n"

	if m.searching {
		s += " 🔍 " + m.textInput.View() + "\n\n"
		s += statusStyle.Render(" [ENTER] Search • [Q] Quit")
	} else {
		s += m.list.View()
		s += "\n\n" + statusStyle.Render(" [ESC] New Search • [C] Copy Link • [Q] Quit")
	}

	if m.err != nil {
		s += "\n\n" + errorStyle.Render(" ERROR: ") + m.err.Error()
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(s)
}
