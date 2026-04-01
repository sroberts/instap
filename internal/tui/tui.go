package tui

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sroberts/instap/internal/api"
)

var (
	// Colors (Catppuccin Macchiato inspired)
	primaryColor   = lipgloss.Color("#8aadf4") // Blue
	secondaryColor = lipgloss.Color("#b7bdf8") // Lavender
	starredColor   = lipgloss.Color("#eed49f") // Yellow
	tagColor       = lipgloss.Color("#8bd5ca") // Teal
	errorColor     = lipgloss.Color("#ed8796") // Red
	surfaceColor   = lipgloss.Color("#363a4f") // Surface
	overlayColor   = lipgloss.Color("#494d64") // Overlay
	textColor      = lipgloss.Color("#cad3f5") // Text
	subtextColor   = lipgloss.Color("#8087a2") // Subtext

	// Styles
	headerStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#24273a")).
			Padding(0, 1).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(subtextColor).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(primaryColor).
				PaddingLeft(1).
				Background(overlayColor)

	normalItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	titleStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	selectedTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	urlStyle = lipgloss.NewStyle().
			Foreground(subtextColor).
			Italic(true)

	tagStyle = lipgloss.NewStyle().
			Foreground(tagColor).
			Italic(true)

	starredStyle = lipgloss.NewStyle().
			Foreground(starredColor)

	readerHeaderStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Underline(true).
				MarginBottom(1)
)

const asciiLogo = `  ___           _
 |_ _|_ __  ___| |_ __ _ _ __
  | || '_ \(_-<|  _/ _' | '_ \
 |___|_| |_/__/ \__\__,_| .__/
                        |_|`

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	title := i.bookmark.Title
	if title == "" {
		title = "Untitled"
	}

	starred := ""
	if i.bookmark.Starred == "1" {
		starred = starredStyle.Render(" ★")
	}

	tags := ""
	if i.bookmark.Tags != "" {
		tags = " " + tagStyle.Render("["+i.bookmark.Tags+"]")
	}

	var str string
	if index == m.Index() {
		t := selectedTitleStyle.Render(title)
		u := urlStyle.Render(i.bookmark.URL)
		str = selectedItemStyle.Render(fmt.Sprintf("%s%s%s\n%s", t, starred, tags, u))
	} else {
		t := titleStyle.Render(title)
		u := urlStyle.Render(i.bookmark.URL)
		str = normalItemStyle.Render(fmt.Sprintf("%s%s%s\n%s", t, starred, tags, u))
	}

	fmt.Fprint(w, str)
}

type listKeyMap struct {
	archive key.Binding
	delete  key.Binding
	star    key.Binding
	move    key.Binding
	refresh key.Binding
	open    key.Binding
	read    key.Binding
	tag     key.Binding
}

func (k listKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.read, k.archive, k.move, k.tag}
}

func (k listKeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		k.read, k.open, k.archive, k.star, k.move, k.tag, k.delete, k.refresh,
	}
}

var keys = listKeyMap{
	read: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "read"),
	),
	open: key.NewBinding(
		key.WithKeys("shift+enter", "o"),
		key.WithHelp("o", "open in browser"),
	),
	archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	star: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "star/unstar"),
	),
	move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move to folder"),
	),
	refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	tag: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "tags"),
	),
}

type item struct {
	bookmark api.Bookmark
}

func (i item) Title() string { return i.bookmark.Title }
func (i item) Description() string { return i.bookmark.URL }
func (i item) FilterValue() string { return i.bookmark.Title + " " + i.bookmark.Tags }

type state int

const (
	stateBrowsing state = iota
	stateMoving
	stateReading
	stateTagging
)

type stats struct {
	total   int
	starred int
	folders int
}

type model struct {
	list         list.Model
	folderList   list.Model
	viewport     viewport.Model
	tagInput     textinput.Model
	spinner      spinner.Model
	client       *api.Client
	stats        stats
	state        state
	status       string
	isError      bool
	isLoading    bool
	selectedItem *item
	width        int
	height       int
}

type statusMsg string
type clearStatusMsg struct{}
type errMsg error
type bookmarksMsg []api.Bookmark
type foldersMsg struct {
	folders []api.Folder
	switchState bool
}
type contentMsg string

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchBookmarks(),
		m.fetchFolders(false),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.state == stateReading {
			switch msg.String() {
			case "q", "esc":
				m.state = stateBrowsing
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		if m.state == stateTagging {
			switch msg.String() {
			case "enter":
				if m.selectedItem != nil {
					tagList := strings.Split(m.tagInput.Value(), ",")
					var tags []string
					for _, t := range tagList {
						trimmed := strings.TrimSpace(t)
						if trimmed != "" {
							tags = append(tags, trimmed)
						}
					}
					m.isLoading = true
					return m, tea.Batch(m.setTags(m.selectedItem.bookmark.ID, tags), m.spinner.Tick)
				}
			case "esc":
				m.state = stateBrowsing
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		}

		if m.state == stateBrowsing {
			switch {
			case key.Matches(msg, keys.tag):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.selectedItem = &i
					m.tagInput.SetValue(i.bookmark.Tags)
					m.tagInput.Focus()
					m.state = stateTagging
					return m, nil
				}
			case key.Matches(msg, keys.open):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					openBrowser(i.bookmark.URL)
					m.status = "Opened in browser"
					return m, m.clearStatusAfter(2 * time.Second)
				}
				return m, nil
			case key.Matches(msg, keys.read):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.selectedItem = &i
					m.status = "Fetching content..."
					m.isLoading = true
					return m, tea.Batch(m.fetchAndRender(i.bookmark), m.spinner.Tick)
				}
			case key.Matches(msg, keys.archive):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.isLoading = true
					return m, tea.Batch(m.archiveBookmark(i.bookmark.ID), m.spinner.Tick)
				}
			case key.Matches(msg, keys.delete):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.isLoading = true
					return m, tea.Batch(m.deleteBookmark(i.bookmark.ID), m.spinner.Tick)
				}
			case key.Matches(msg, keys.star):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.isLoading = true
					return m, tea.Batch(m.toggleStar(i.bookmark), m.spinner.Tick)
				}
			case key.Matches(msg, keys.refresh):
				m.status = "Refreshing..."
				m.isLoading = true
				return m, tea.Batch(m.fetchBookmarks(), m.spinner.Tick)
			case key.Matches(msg, keys.move):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.selectedItem = &i
					m.status = "Fetching folders..."
					m.isLoading = true
					return m, tea.Batch(m.fetchFolders(true), m.spinner.Tick)
				}
			case msg.String() == "q":
				return m, tea.Quit
			}
		} else if m.state == stateMoving {
			switch msg.String() {
			case "enter":
				f, ok := m.folderList.SelectedItem().(folderItem)
				if ok && m.selectedItem != nil {
					m.isLoading = true
					return m, tea.Batch(m.moveBookmark(m.selectedItem.bookmark.ID, f.folder.ID), m.spinner.Tick)
				}
			case "esc", "q":
				m.state = stateBrowsing
				return m, nil
			}
		}

	case contentMsg:
		m.state = stateReading
		m.viewport.SetContent(string(msg))
		m.viewport.GotoTop()
		m.status = ""
		m.isLoading = false
		return m, nil

	case statusMsg:
		m.status = string(msg)
		m.isError = false
		m.isLoading = false
		return m, m.clearStatusAfter(3 * time.Second)

	case clearStatusMsg:
		m.status = ""
		m.isError = false
		return m, nil

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg)
		m.isError = true
		m.isLoading = false
		return m, nil

	case bookmarksMsg:
		items := make([]list.Item, len(msg))
		starredCount := 0
		for i, b := range msg {
			items[i] = item{bookmark: b}
			if b.Starred == "1" {
				starredCount++
			}
		}
		m.stats.total = len(msg)
		m.stats.starred = starredCount
		m.list.SetItems(items)
		m.isLoading = false
		if m.status == "Fetching data..." {
			m.status = ""
		}
		return m, nil

	case foldersMsg:
		items := make([]list.Item, len(msg.folders))
		for i, f := range msg.folders {
			items[i] = folderItem{folder: f}
		}
		m.stats.folders = len(msg.folders)
		m.folderList.SetItems(items)
		m.isLoading = false
		if msg.switchState {
			m.state = stateMoving
			m.status = "Select folder"
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.state == stateBrowsing {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.state == stateMoving {
		var cmd tea.Cmd
		m.folderList, cmd = m.folderList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	header := m.headerView()
	headerHeight := lipgloss.Height(header)

	info := fmt.Sprintf(" %d/%d ", m.list.Index()+1, len(m.list.Items()))
	if m.state == stateReading {
		info = fmt.Sprintf(" %d%% ", int(m.viewport.ScrollPercent()*100))
	}
	footerInfo := footerStyle.Background(primaryColor).Foreground(lipgloss.Color("#24273a")).Bold(true).Render(info)

	statusText := ""
	if m.isLoading {
		statusText = m.spinner.View() + " " + m.status
	} else if m.status != "" {
		if m.isError {
			statusText = errorStyle.Render(m.status)
		} else {
			statusText = statusStyle.Render(m.status)
		}
	}
	footerStatus := footerStyle.Width(m.width - lipgloss.Width(footerInfo)).Render(statusText)
	footer := lipgloss.JoinHorizontal(lipgloss.Bottom, footerStatus, footerInfo)
	footerHeight := lipgloss.Height(footer)

	// Layout: header + window border(2) + windowHeight(inner) + footer = m.height
	windowHeight := m.height - headerHeight - footerHeight - 2
	if windowHeight < 4 {
		windowHeight = 4
	}

	windowStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(overlayColor).
		Padding(0, 1).
		Width(m.width - 2).
		Height(windowHeight)

	// Measure help bar so we can subtract it from content area precisely
	helpStr := ""
	helpHeight := 0
	if m.state != stateTagging && m.state != stateMoving {
		helpStr = "\n" + m.helpView()
		helpHeight = lipgloss.Height(helpStr)
	}

	// Inner content width: window width - border(2) - horizontal padding(2)
	innerWidth := m.width - 4

	var content string
	switch m.state {
	case stateBrowsing:
		m.list.SetSize(innerWidth, windowHeight-helpHeight)
		content = m.list.View()
	case stateMoving:
		m.folderList.SetSize(innerWidth, windowHeight)
		content = m.folderList.View()
	case stateReading:
		rh := ""
		rhHeight := 0
		if m.selectedItem != nil {
			rh = readerHeaderStyle.Render(m.selectedItem.bookmark.Title) + "\n"
			rhHeight = lipgloss.Height(rh)
		}
		m.viewport.Width = innerWidth
		m.viewport.Height = windowHeight - helpHeight - rhHeight
		content = rh + m.viewport.View()
	case stateTagging:
		content = fmt.Sprintf(
			"\n  Tags for: %s\n\n  %s\n\n  (enter to save, esc to cancel)",
			m.selectedItem.bookmark.Title,
			m.tagInput.View(),
		)
	}

	content += helpStr

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		windowStyle.Render(content),
		footer,
	)
}

func (m model) headerView() string {
	logo := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(asciiLogo)

	statsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(overlayColor).
		Padding(0, 1)

	statsContent := fmt.Sprintf(
		"Total:   %d\nStarred: %d\nFolders: %d",
		m.stats.total,
		m.stats.starred,
		m.stats.folders,
	)

	statsBox := statsStyle.Render(statsContent)
	gap := lipgloss.NewStyle().Width(4).Render("")

	header := lipgloss.JoinHorizontal(lipgloss.Top, logo, gap, statsBox)
	return lipgloss.NewStyle().PaddingLeft(2).Render(header)
}

func (m model) helpView() string {
	var shortcuts string
	switch m.state {
	case stateBrowsing:
		shortcuts = "enter:read • o:open • a:arch • s:star • m:move • t:tags • d:del • r:refresh"
	case stateReading:
		shortcuts = "q/esc:back • arrows/j/k:scroll"
	default:
		return ""
	}
	style := lipgloss.NewStyle().
		Background(overlayColor).
		Foreground(secondaryColor).
		Padding(0, 1).
		Bold(true).
		Width(m.width - 4).
		Align(lipgloss.Center)
	
	return style.Render(shortcuts)
}

func (m model) clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// --- Commands ---

func (m model) fetchAndRender(b api.Bookmark) tea.Cmd {
	return func() tea.Msg {
		html, err := m.client.GetBookmarkText(b.ID)
		if err != nil {
			return errMsg(err)
		}
		converter := md.NewConverter("", true, nil)
		markdown, err := converter.ConvertString(html)
		if err != nil {
			return errMsg(err)
		}
		markdown = fmt.Sprintf("# %s\n\n%s\n\n---\n\n%s", b.Title, b.URL, markdown)
		r, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(m.viewport.Width))
		out, _ := r.Render(markdown)
		return contentMsg(out)
	}
}

func (m model) fetchBookmarks() tea.Cmd {
	return func() tea.Msg {
		bookmarks, err := m.client.ListBookmarks("")
		if err != nil {
			return errMsg(err)
		}
		return bookmarksMsg(bookmarks)
	}
}

func (m model) archiveBookmark(id int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.ArchiveBookmark(id); err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return tea.Batch(func() tea.Msg { return statusMsg("Archived") }, func() tea.Msg { return bookmarksMsg(bookmarks) })()
	}
}

func (m model) deleteBookmark(id int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteBookmark(id); err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return tea.Batch(func() tea.Msg { return statusMsg("Deleted") }, func() tea.Msg { return bookmarksMsg(bookmarks) })()
	}
}

func (m model) toggleStar(b api.Bookmark) tea.Cmd {
	return func() tea.Msg {
		var err error
		if b.Starred == "1" {
			err = m.client.UnstarBookmark(b.ID)
		} else {
			err = m.client.StarBookmark(b.ID)
		}
		if err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return bookmarksMsg(bookmarks)
	}
}

func (m model) fetchFolders(switchState bool) tea.Cmd {
	return func() tea.Msg {
		folders, err := m.client.ListFolders()
		if err != nil {
			return errMsg(err)
		}
		return foldersMsg{folders: folders, switchState: switchState}
	}
}

func (m model) setTags(bookmarkID int, tags []string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.SetTags(bookmarkID, tags); err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return tea.Batch(
			func() tea.Msg { return statusMsg("Tags updated") },
			func() tea.Msg { return bookmarksMsg(bookmarks) },
			func() tea.Msg { return stateMsg(stateBrowsing) },
		)()
	}
}

func (m model) moveBookmark(bookmarkID, folderID int) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.MoveBookmark(bookmarkID, folderID); err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return tea.Batch(
			func() tea.Msg { return statusMsg("Moved") },
			func() tea.Msg { return bookmarksMsg(bookmarks) },
			func() tea.Msg { return stateMsg(stateBrowsing) },
		)()
	}
}

type stateMsg state

type folderItem struct {
	folder api.Folder
}

func (i folderItem) Title() string       { return i.folder.Title }
func (i folderItem) Description() string { return fmt.Sprintf("ID: %d", i.folder.ID) }
func (i folderItem) FilterValue() string { return i.folder.Title }

func openBrowser(u string) {
	if !api.IsValidURL(u) {
		return
	}
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", u).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		err = exec.Command("open", u).Start()
	}
	_ = err
}

func Run(client *api.Client) error {
	ti := textinput.New()
	ti.Placeholder = "tag1, tag2, ..."
	ti.CharLimit = 250
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.PaginationStyle = footerStyle
	l.Styles.HelpStyle = footerStyle

	fl := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	fl.SetShowTitle(false)
	fl.SetShowStatusBar(false)
	fl.SetShowHelp(false)

	m := model{
		list:       l,
		folderList: fl,
		viewport:   viewport.New(0, 0),
		tagInput:   ti,
		spinner:    s,
		client:     client,
		state:      stateBrowsing,
		isLoading:  true,
		status:     "Fetching data...",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
