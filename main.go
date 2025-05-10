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

const (
	tableView uint = iota
	detailView
)

type model struct {
	table         table.Model
	allRows       []table.Row
	textInput     textinput.Model
	filteredInput string
	viewstate     uint
	isfilter      bool
	selectedRow   table.Row
	trashList     []fi
}

var mul_rate int = 5

const (
	numWidth        int = 1
	nameWidth       int = 4
	sizeWidth       int = 2
	dateDeleteWidth int = 6
	fullPathWidth   int = 10
)

var columns = []table.Column{
	{Title: "#", Width: numWidth * mul_rate},
	{Title: "Name", Width: nameWidth * mul_rate},
	{Title: "Size", Width: sizeWidth * mul_rate},
	{Title: "Date Deleted", Width: dateDeleteWidth * mul_rate},
	{Title: "Location", Width: fullPathWidth * mul_rate},
}

var numTableRows int = 20

func initialModel() model {
	trash, err := GetTrashBoxItems()
	if err != nil {
		fmt.Println("go-trash: ", err)
		os.Exit(1)
	}

	var allRows = []table.Row{}
	for i, tf := range trash {
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
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		table:     t,
		allRows:   allRows,
		textInput: ti,
		trashList: trash,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "esc":
			if m.viewstate == detailView {
				m.viewstate = tableView
				return m, nil
			}

			// Cancel filter
			if m.isfilter {
				m.isfilter = false
				m.textInput.Reset()
				return m, nil
			}
			return m, tea.Quit

		case msg.String() == "/":
			// Filter
			if !m.isfilter {
				m.isfilter = true
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case msg.String() == "U":
			cursor := m.table.Cursor()
			if cursor < len(m.table.Rows()) {
				Undelete(m.trashList[cursor].inTrashBox, m.trashList[cursor].location)
			}
			m.allRows = append(m.allRows[:cursor], m.allRows[cursor+1:]...)
			m.trashList = append(m.trashList[:cursor], m.trashList[cursor+1:]...)
			m.table.SetRows(m.allRows)

			if m.viewstate == detailView {
				m.viewstate = tableView
			}

		case msg.Type == tea.KeyEnter:
			if m.isfilter {
				m.filteredInput = m.textInput.Value()
				filteredRows := filterRows(m.allRows, m.filteredInput)
				m.table.SetRows(filteredRows)
				m.isfilter = false
				m.textInput.Reset()
				return m, nil
			}

			if m.viewstate == tableView {
				cursor := m.table.Cursor()
				if cursor < len(m.table.Rows()) {
					m.selectedRow = m.table.Rows()[cursor]
					m.viewstate = detailView
				}
			} else if m.viewstate == detailView {
				m.viewstate = tableView
			}
		}
	}

	// Update input only if filtering
	if m.isfilter {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	m.table, _ = m.table.Update(msg)

	return m, cmd
}

func (m model) View() string {
	var sb strings.Builder
	switch m.viewstate {
	case tableView:
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

	case detailView:
		// Header
		sb.WriteString("📋 Detail Viewer 📋\n\n")

		// TODO: Show contents
		// Body
		for i, v := range m.selectedRow {
			sb.WriteString(fmt.Sprintf("%-18s: %s\n", columns[i].Title, v))
		}

		// Footer
		sb.WriteString("\n\n")
		sb.WriteString("[U]:Undelete file [Esc]:quit\n")
	}
	return sb.String()
}

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

	for _, path := range args {
		err := MoveToTrashBox(path)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
	}
}
