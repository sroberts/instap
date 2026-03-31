package tui

import (
	"fmt"
	"os/exec"
	"runtime"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sroberts/instap/internal/api"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)
var statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Italic(true)

type listKeyMap struct {
	archive key.Binding
	delete  key.Binding
	star    key.Binding
	move    key.Binding
	refresh key.Binding
	open    key.Binding
	read    key.Binding
}

func (k listKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.read, k.archive, k.move}
}

func (k listKeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		k.read, k.open, k.archive, k.star, k.move, k.delete, k.refresh,
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
}

type item struct {
	bookmark api.Bookmark
}

func (i item) Title() string {
	starred := ""
	if i.bookmark.Starred == "1" {
		starred = " ★"
	}
	return i.bookmark.Title + starred
}
func (i item) Description() string { return i.bookmark.URL }
func (i item) FilterValue() string { return i.bookmark.Title }

type state int

const (
	stateBrowsing state = iota
	stateMoving
	stateReading
)

type model struct {
	list         list.Model
	folderList   list.Model
	viewport     viewport.Model
	client       *api.Client
	state        state
	status       string
	selectedItem *item
	width        int
	height       int
}

type statusMsg string
type errMsg error
type bookmarksMsg []api.Bookmark
type foldersMsg []api.Folder
type contentMsg string

func (m model) Init() tea.Cmd {
	return nil
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

		if m.state == stateBrowsing {
			switch {
			case key.Matches(msg, keys.open):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					openBrowser(i.bookmark.URL)
					m.status = "Opened in browser"
				}
				return m, nil
			case key.Matches(msg, keys.read):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.status = "Fetching content..."
					return m, m.fetchAndRender(i.bookmark)
				}
			case key.Matches(msg, keys.archive):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					return m, m.archiveBookmark(i.bookmark.ID)
				}
			case key.Matches(msg, keys.delete):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					return m, m.deleteBookmark(i.bookmark.ID)
				}
			case key.Matches(msg, keys.star):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					return m, m.toggleStar(i.bookmark)
				}
			case key.Matches(msg, keys.refresh):
				m.status = "Refreshing..."
				return m, m.fetchBookmarks()
			case key.Matches(msg, keys.move):
				i, ok := m.list.SelectedItem().(item)
				if ok {
					m.selectedItem = &i
					m.status = "Fetching folders..."
					return m, m.fetchFolders()
				}
			case msg.String() == "q":
				return m, tea.Quit
			}
		} else if m.state == stateMoving {
			switch msg.String() {
			case "enter":
				f, ok := m.folderList.SelectedItem().(folderItem)
				if ok && m.selectedItem != nil {
					return m, m.moveBookmark(m.selectedItem.bookmark.ID, f.folder.ID)
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
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg)
		return m, nil

	case bookmarksMsg:
		items := make([]list.Item, len(msg))
		for i, b := range msg {
			items[i] = item{bookmark: b}
		}
		m.list.SetItems(items)
		m.status = "Updated"
		return m, nil

	case foldersMsg:
		items := make([]list.Item, len(msg))
		for i, f := range msg {
			items[i] = folderItem{folder: f}
		}
		m.folderList.SetItems(items)
		m.state = stateMoving
		m.status = "Select folder"
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-2)
		m.folderList.SetSize(msg.Width-h, msg.Height-v-2)
		m.viewport.Width = msg.Width - h
		m.viewport.Height = msg.Height - v - 2
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
	var s string
	switch m.state {
	case stateBrowsing:
		s = m.list.View()
	case stateMoving:
		s = m.folderList.View()
	case stateReading:
		s = m.viewport.View() + "\n" + m.helpView()
	}

	status := ""
	if m.status != "" {
		status = "\n" + statusStyle.Render(m.status)
	}

	return docStyle.Render(s + status)
}

func (m model) helpView() string {
	return statusStyle.Render("q/esc: back • arrows/j/k: scroll")
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

		// Prepend title and URL
		markdown = fmt.Sprintf("# %s\n\n%s\n\n---\n\n%s", b.Title, b.URL, markdown)

		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.viewport.Width),
		)
		if err != nil {
			return errMsg(err)
		}

		out, err := r.Render(markdown)
		if err != nil {
			return errMsg(err)
		}

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
		err := m.client.ArchiveBookmark(id)
		if err != nil {
			return errMsg(err)
		}
		bookmarks, _ := m.client.ListBookmarks("")
		return tea.Batch(func() tea.Msg { return statusMsg("Archived") }, func() tea.Msg { return bookmarksMsg(bookmarks) })()
	}
}

func (m model) deleteBookmark(id int) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteBookmark(id)
		if err != nil {
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

func (m model) fetchFolders() tea.Cmd {
	return func() tea.Msg {
		folders, err := m.client.ListFolders()
		if err != nil {
			return errMsg(err)
		}
		return foldersMsg(folders)
	}
}

func (m model) moveBookmark(bookmarkID, folderID int) tea.Cmd {
	return func() tea.Msg {
		err := m.client.MoveBookmark(bookmarkID, folderID)
		if err != nil {
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

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	_ = err
}

func Run(client *api.Client) error {
	bookmarks, err := client.ListBookmarks("")
	if err != nil {
		return err
	}

	items := make([]list.Item, len(bookmarks))
	for i, b := range bookmarks {
		items[i] = item{bookmark: b}
	}

	m := model{
		list:       list.New(items, list.NewDefaultDelegate(), 0, 0),
		folderList: list.New(nil, list.NewDefaultDelegate(), 0, 0),
		viewport:   viewport.New(0, 0),
		client:     client,
		state:      stateBrowsing,
	}
	m.list.Title = "Instapaper Bookmarks"
	m.list.AdditionalShortHelpKeys = keys.ShortHelp
	m.list.AdditionalFullHelpKeys = keys.FullHelp
	m.folderList.Title = "Select Folder"

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
