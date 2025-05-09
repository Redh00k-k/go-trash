package main

import (
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

const (
	mainView uint = iota
	detailView
)

type model struct {
	table   table.Model
	allRows []table.Row
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
	{Title: "DateDeleted", Width: dateDeleteWidth * mul_rate},
	{Title: "FullPath", Width: fullPathWidth * mul_rate},
}

var numTableRows int = 20

func initialModel() model {
	trashfiles, err := GetTrashBoxItems()
	if err != nil {
		fmt.Println("go-trash: ", err)
		os.Exit(1)
	}

	var allRows = []table.Row{}
	for i, tf := range trashfiles {
		tmp := []string{strconv.Itoa(i + 1), tf.InFolder, strconv.FormatInt(tf.Size, 10), tf.DateDeleted.Format("2006-01-02T15:04:05Z07:00"), tf.Normal}
		allRows = append(allRows, tmp)
	}

	// Create the input
	ti := textinput.New()
	ti.Placeholder = "Filter..."
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
		table:   t,
		allRows: allRows,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// case tea.WindowSizeMsg:
	// wW := msg.Width
	// wH := msg.Height

	// reservedLines := numTableRows
	// if m.isfilter {
	// 	reservedLines += 2
	// }

	// usableHeight := msg.Height - reservedLines
	// if usableHeight < 3 {
	// 	usableHeight = 3
	// }

	// m.table.SetHeight(usableHeight)
	// m.table.SetColumns(columns)

	// m.table.SetWidth(msg.Width - mul_rate)
	// m.table.SetHeight(msg.Height - mul_rate)
	// w := m.table.Width() - 6
	// columns[0].Width = w * numWidth / 20        // #
	// columns[1].Width = w * nameWidth / 20       // Name
	// columns[2].Width = w * sizeWidth / 20       // Size
	// columns[3].Width = w * dateDeleteWidth / 20 // DateDeleted
	// columns[4].Width = w * fullPathWidth / 20   // FullPath
	// m.table.SetColumns(columns)
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c" || msg.String() == "esc":
			return m, tea.Quit
		}
	}

	m.table, _ = m.table.Update(msg)

	return m, cmd
}

func (m model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString("📋TrashBox Viewer\n\n")

	// Table
	sb.WriteString(m.table.View())

	// Footer
	sb.WriteString("\n\n")
	sb.WriteString("[Esc]:quit\n")

	return sb.String()
}

type fi struct {
	InFolder    string
	Normal      string
	ForParsing  string
	DateDeleted time.Time
	Size        int64
}

func main() {
	var (
		isList       = false
		isHelp       = false
		undeleteFile = ""
		outputPath   = ""
		tuiMode      = false
	)

	getopt.Flag(&isList, 'l', "List trashed files")
	getopt.Flag(&isHelp, 'h', "Show help")
	getopt.Flag(&undeleteFile, 'u', "Restore files to original location", "File")
	getopt.Flag(&outputPath, 'o', "Output file to location", "File")
	getopt.Flag(&tuiMode, 't', "Run TUI mode")
	getopt.Parse()
	args := getopt.Args()

	if tuiMode || len(args) == 0 {
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(undeleteFile) != 0 {
		err := RestoreItem(undeleteFile, outputPath)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
		os.Exit(0)
	}

	if isList == true {
		fmt.Println("")
		fmt.Println("# Trash Box #")
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

	for _, path := range args {
		err := MoveToTrashBox(path)
		if err != nil {
			fmt.Println("go-trash: ", err)
		}
	}
}
