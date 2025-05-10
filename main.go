package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pborman/getopt/v2"
)

type fi struct {
	filename    string
	location    string
	inTrashBox  string
	dateDeleted time.Time
	size        int64
}

var columns = []table.Column{
	{Title: "#", Width: 5},
	{Title: "Name", Width: 20},
	{Title: "Size", Width: 10},
	{Title: "Date Deleted", Width: 25},
	{Title: "Location", Width: 40},
}

type changeViewMsg struct {
	toView uint
	row    table.Row
}

const (
	tableView uint = iota
	detailView
)

// Table
type tableModel struct {
	table     table.Model
	textInput textinput.Model
	isfilter  bool
	allRows   []table.Row
	trashList []fi
}

var numTableRows int = 20

func newTableModel(rows []table.Row, trashList []fi) tableModel {
	// Create the input
	ti := textinput.New()
	ti.Placeholder = "…"
	ti.Focus()

	// Create the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(numTableRows+1), // table rows + title
	)

	// ref: https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("29")).
		Bold(false)
	t.SetStyles(s)

	return tableModel{
		table:     t,
		textInput: ti,
		allRows:   rows,
		trashList: trashList,
	}
}

func (m tableModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "esc":
			// Cancel filter
			if m.isfilter {
				m.isfilter = false
				m.textInput.Reset()
				return m, nil
			}
			return m, tea.Quit
		case msg.String() == "U":
			cursor := m.table.Cursor()
			if cursor < len(m.table.Rows()) {
				Undelete(m.trashList[cursor].inTrashBox, m.trashList[cursor].location)
			}
			m.allRows = append(m.allRows[:cursor], m.allRows[cursor+1:]...)
			m.trashList = append(m.trashList[:cursor], m.trashList[cursor+1:]...)
			m.table.SetRows(m.allRows)

		case msg.String() == "/":
			m.isfilter = true
			m.textInput.Focus()
			return m, textinput.Blink
		case msg.String() == "enter":
			if m.isfilter {
				keyword := m.textInput.Value()
				m.table.SetRows(filterRows(m.allRows, keyword))
				m.isfilter = false
				m.textInput.Reset()
				return m, nil
			}
			cursor := m.table.Cursor()
			if cursor < len(m.table.Rows()) {
				return m, func() tea.Msg {
					return changeViewMsg{
						toView: detailView,
						row:    m.table.Rows()[cursor],
					}
				}
			}
		}
	}
	if m.isfilter {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	m.table, _ = m.table.Update(msg)
	return m, nil
}

func (m tableModel) View() string {
	var sb strings.Builder
	// Header
	sb.WriteString("🗑️ TrashBox Viewer 🗑️\n\n")

	// Filter
	if m.isfilter {
		sb.WriteString("🔍 Filtering: " + m.textInput.View() + "\n\n")
	}

	// Body(Table)
	sb.WriteString(m.table.View())

	// Footer
	sb.WriteString("\n\n")
	if m.isfilter {
		sb.WriteString("[Enter]: apply filter  [Esc]:cancel filter\n")
	} else {
		sb.WriteString("[/]:start filter [U]:Undelete file [Esc]:quit\n")
	}
	return sb.String()
}

// Detail
type detailModel struct {
	row       table.Row
	trashList []fi
}

func (m detailModel) Init() tea.Cmd {
	return nil
}

func newDetailModel(row table.Row, trashList []fi, width int, height int) detailModel {
	return detailModel{
		row:       row,
		trashList: trashList,
	}
}

func (m detailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "esc":
			return m, func() tea.Msg {
				return changeViewMsg{toView: tableView}
			}
		}
	}

	return m, nil
}

func (m detailModel) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString("📋 Detail Viewer" + "\n\n")

	// Body
	for i, v := range m.row {
		sb.WriteString(fmt.Sprintf("%-18s: %s\n", columns[i].Title, v))
	}

	// Footer
	sb.WriteString("\n\n")
	sb.WriteString("[U]:Undelete file  [Esc]: Back\n")

	return sb.String()
}

// main
type mainModel struct {
	viewstate uint
	sub       tea.Model
	rows      []table.Row
	trashList []fi
	textInput textinput.Model
}

func (m mainModel) Init() tea.Cmd {
	return m.sub.Init()
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case changeViewMsg:
		if msg.toView == tableView {
			tm := newTableModel(m.rows, m.trashList)
			m.viewstate = tableView
			m.sub = tm
			return m, tm.Init()
		} else if msg.toView == detailView {
			dm := newDetailModel(msg.row, m.trashList, 80, 20)
			m.viewstate = detailView
			m.sub = dm
			return m, dm.Init()
		}
	}
	subModel, cmd := m.sub.Update(msg)
	m.sub = subModel
	return m, cmd
}

func (m mainModel) View() string {
	return m.sub.View()
}

// fileter Helper
func filterRows(rows []table.Row, keyword string) []table.Row {
	if keyword == "" {
		return rows
	}
	var filtered []table.Row
	for _, row := range rows {
		for _, col := range row {
			if strings.Contains(strings.ToLower(col), strings.ToLower(keyword)) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered
}

func initialModel() mainModel {
	trashList, err := GetTrashBoxItems()
	if err != nil {
		fmt.Println("go-trash: ", err)
		os.Exit(1)
	}

	var allRows = []table.Row{}
	for i, tf := range trashList {
		tmp := []string{strconv.Itoa(i), tf.filename, strconv.FormatInt(tf.size, 10), tf.dateDeleted.Format("2006-01-02T15:04:05Z07:00"), tf.location}
		allRows = append(allRows, tmp)
	}

	// Create the input
	ti := textinput.New()
	ti.Placeholder = "…"
	ti.Focus()

	// Create the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(allRows),
		table.WithFocused(true),
		table.WithHeight(numTableRows+1), // table rows + title
	)

	// ref: https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("29")).
		Bold(false)
	t.SetStyles(s)

	start := newTableModel(allRows, trashList)
	return mainModel{
		viewstate: tableView,
		sub:       start,
		rows:      allRows,
		textInput: ti,
		trashList: trashList,
	}
}

func main() {
	var (
		isList       = false
		isHelp       = false
		undeleteFile = ""
		outputPath   = ""
		isTuiMode    = false
	)

	getopt.Flag(&isList, 'l', "List trashed files")
	getopt.Flag(&isHelp, 'h', "Show help")
	getopt.Flag(&undeleteFile, 'u', "Restore files to original location", "File")
	getopt.Flag(&outputPath, 'o', "Output file to location", "File")
	getopt.Flag(&isTuiMode, 't', "Run TUI mode")
	getopt.Parse()
	args := getopt.Args()

	if len(undeleteFile) != 0 {
		trashfiles, err := GetTrashBoxItems()
		if err != nil {
			fmt.Println("go-trash: ", err)
			os.Exit(1)
		}

		var udFileList []fi
		for _, file := range trashfiles {
			if strings.Contains(file.filename, undeleteFile) {
				udFileList = append(udFileList, file)
			}
		}

		if len(udFileList) > 1 {
			fmt.Printf("Found %d files that matched.\n\n", len(udFileList))
			for _, file := range udFileList {
				fmt.Printf("Filename: %s\n", file.filename)
				fmt.Printf("Location: %s\n\n", file.location)
			}
			fmt.Printf("Do you want to undelete them? [Y/n]: ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if scanner.Text() != "Y" {
				os.Exit(0)
			}
		}

		for _, file := range udFileList {
			err := Undelete(file.inTrashBox, file.location)
			if err != nil {
				fmt.Println("go-trash: ", err)
				os.Exit(1)
			}
			fmt.Printf("UnDelete %s → %s\n", file.filename, file.location)
		}

		os.Exit(0)
	}

	if isList {
		fmt.Println("")
		fmt.Println("🗑️ TrashBox 🗑️")
		err := PrintTrashBoxItems()
		if err != nil {
			fmt.Println("go-trash: ", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if isHelp {
		getopt.Usage()
		os.Exit(1)
	}

	if isTuiMode || len(args) == 0 {
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Move to trash
	for _, path := range args {
		err := MoveToTrashBox(path)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
	}
}
