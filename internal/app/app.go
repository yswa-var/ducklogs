package app

import (
	"context"
	"fmt"
	"strings"

	"ducklogs/internal/config"
	"ducklogs/internal/ducklogai"
	"ducklogs/internal/logs"
	"ducklogs/internal/report"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	screenAsk = iota
	screenIngest
)

const (
	askDBPathField = iota
	askPromptField
	askReportField
)

const (
	ingestLogGroupField = iota
	ingestRegionField
	ingestDBPathField
	ingestSearchField
	ingestTimeWindowField
)

type ingestCompleteMsg struct {
	dbPath string
	err    error
}

type askCompleteMsg struct {
	result *ducklogai.Result
	err    error
}

type field struct {
	label string
	input textinput.Model
}

type model struct {
	cfg          config.Config
	screen       int
	focusIndex   int
	submitting   bool
	askFields    []field
	ingestFields []field
	status       string
	errMessage   string
	successPath  string
	sqlPreview   string
	resultText   string
	reportPath   string
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	labelStyle   = lipgloss.NewStyle().Bold(true)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	panelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("63"))
	sidebarStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(lipgloss.Color("240")).Padding(0, 2, 0, 0)
)

func Run() error {
	program := tea.NewProgram(newModel())
	_, err := program.Run()
	return err
}

func newModel() model {
	cfg := config.Load()
	askFields := []field{
		newField("DuckDB Path", cfg.Database),
		newField("Prompt", "find all occurrences of vendor not found where tracking id is af01198314485493"),
		newField("Report Path", cfg.ReportPath),
	}
	ingestFields := []field{
		newField("Log Group", "/aws/ecs/default/calculate-orchestrator-bac3-9839"),
		newField("Region", "us-east-1"),
		newField("DuckDB Path", cfg.Database),
		newField("Search (Optional)", ""),
		newField("Time Window", "30m"),
	}

	m := model{
		cfg:          cfg,
		screen:       screenAsk,
		askFields:    askFields,
		ingestFields: ingestFields,
		status:       "Ask a logs question, preview SQL, run it, and write a Markdown report.",
	}
	m.syncFocus()
	return m
}

func newField(label, value string) field {
	input := textinput.New()
	input.SetValue(value)
	input.Width = 72
	input.Prompt = "> "
	input.Cursor.Style = focusedTextStyle()
	return field{label: label, input: input}
}

func focusedPromptStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
}

func focusedTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
}

func blurredPromptStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
}

func blurredTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.submitting {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.switchScreen(screenAsk)
			return m, nil
		case "2":
			m.switchScreen(screenIngest)
			return m, nil
		case "shift+tab", "up":
			m.focusPrevious()
			return m, nil
		case "tab", "down":
			m.focusNext()
			return m, nil
		case "enter":
			if m.focusIndex == len(*m.fields())-1 {
				return m.submit()
			}
			m.focusNext()
			return m, nil
		case "ctrl+s":
			return m.submit()
		}

	case ingestCompleteMsg:
		m.submitting = false
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			m.status = "Ingest failed. Fix the form or credentials and try again."
			return m, nil
		}

		m.errMessage = ""
		m.successPath = msg.dbPath
		m.status = fmt.Sprintf("Logs ingested successfully into %s", msg.dbPath)
		return m, nil

	case askCompleteMsg:
		m.submitting = false
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			m.status = "Ask failed. Check your API key, database path, or generated SQL."
			return m, nil
		}

		m.errMessage = ""
		m.sqlPreview = msg.result.Plan.SQL
		m.reportPath = msg.result.ReportPath
		if msg.result.Plan.NeedsClarification {
			m.resultText = "Clarification needed: " + msg.result.Plan.ClarifyingQuestion
			m.status = "The question needs one more detail."
			return m, nil
		}

		if msg.result.Rows != nil {
			m.resultText = fmt.Sprintf("Rows: %d\n\n%s", msg.result.Rows.RowCount, report.MarkdownTable(msg.result.Rows, m.cfg.MaxRowsForReport))
		}
		if msg.result.ReportPath != "" {
			m.status = fmt.Sprintf("Report written to %s", msg.result.ReportPath)
		} else {
			m.status = "SQL generated and query completed."
		}
		return m, nil
	}

	fields := m.fields()
	var cmds []tea.Cmd
	for i := range *fields {
		var cmd tea.Cmd
		(*fields)[i].input, cmd = (*fields)[i].input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	header := titleStyle.Render("ducklog") + "  " + hintStyle.Render(fmt.Sprintf("DB: %s  Model: %s  Mode: Safe", m.currentDBPath(), m.cfg.OpenRouterModel))
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar(), m.mainPanel())

	footer := m.status
	if m.submitting {
		if m.screen == screenAsk {
			footer = "Generating SQL, running DuckDB, and preparing report..."
		} else {
			footer = "Ingesting logs into DuckDB..."
		}
	}

	sections := []string{header, body}
	if m.errMessage != "" {
		sections = append(sections, errorStyle.Render(m.errMessage))
	}
	if m.reportPath != "" {
		sections = append(sections, okStyle.Render("Report: "+m.reportPath))
	}
	sections = append(sections, hintStyle.Render(footer))
	sections = append(sections, hintStyle.Render("1 Ask AI  2 Ingest Logs  Tab/Shift+Tab move  Enter continue  Ctrl+S run  Q quit"))

	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(sections, "\n\n"))
}

func (m model) sidebar() string {
	items := []string{"NAV"}
	ask := "  Ask AI"
	ingest := "  Ingest Logs"
	if m.screen == screenAsk {
		ask = "> Ask AI"
	}
	if m.screen == screenIngest {
		ingest = "> Ingest Logs"
	}
	items = append(items, ask, ingest, "  Saved SQL", "  Reports", "  Settings")
	return sidebarStyle.Width(18).Render(strings.Join(items, "\n"))
}

func (m model) mainPanel() string {
	if m.screen == screenIngest {
		return panelStyle.Width(86).Render(m.renderFields("INGEST", *m.fields()))
	}

	var sections []string
	sections = append(sections, m.renderFields("ASK", *m.fields()))
	if m.sqlPreview != "" {
		sections = append(sections, labelStyle.Render("Generated SQL")+"\n"+m.sqlPreview)
	}
	if m.resultText != "" {
		sections = append(sections, labelStyle.Render("Results")+"\n"+m.resultText)
	}
	return panelStyle.Width(86).Render(strings.Join(sections, "\n\n"))
}

func (m model) renderFields(title string, fields []field) string {
	var fieldViews []string
	fieldViews = append(fieldViews, titleStyle.Render(title))
	for i, field := range fields {
		label := labelStyle.Render(field.label)
		if i == m.focusIndex && !m.submitting {
			label = titleStyle.Render(field.label)
		}
		fieldViews = append(fieldViews, fmt.Sprintf("%s\n%s", label, field.input.View()))
	}
	return strings.Join(fieldViews, "\n\n")
}

func (m *model) fields() *[]field {
	if m.screen == screenIngest {
		return &m.ingestFields
	}
	return &m.askFields
}

func (m *model) switchScreen(screen int) {
	m.screen = screen
	m.focusIndex = 0
	m.errMessage = ""
	m.syncFocus()
}

func (m *model) focusNext() {
	m.focusIndex = (m.focusIndex + 1) % len(*m.fields())
	m.syncFocus()
}

func (m *model) focusPrevious() {
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(*m.fields()) - 1
	}
	m.syncFocus()
}

func (m *model) syncFocus() {
	for i := range m.askFields {
		m.askFields[i].input.Blur()
		m.askFields[i].input.PromptStyle = blurredPromptStyle()
		m.askFields[i].input.TextStyle = blurredTextStyle()
	}
	for i := range m.ingestFields {
		m.ingestFields[i].input.Blur()
		m.ingestFields[i].input.PromptStyle = blurredPromptStyle()
		m.ingestFields[i].input.TextStyle = blurredTextStyle()
	}

	fields := m.fields()
	if len(*fields) == 0 {
		return
	}
	(*fields)[m.focusIndex].input.Focus()
	(*fields)[m.focusIndex].input.PromptStyle = focusedPromptStyle()
	(*fields)[m.focusIndex].input.TextStyle = focusedTextStyle()
}

func (m model) submit() (tea.Model, tea.Cmd) {
	if m.screen == screenAsk {
		return m.submitAsk()
	}
	return m.submitIngest()
}

func (m model) submitAsk() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.askFields[askPromptField].input.Value())
	reportPath := strings.TrimSpace(m.askFields[askReportField].input.Value())
	if prompt == "" {
		m.errMessage = "prompt is required"
		m.status = "The Ask form has validation issues."
		return m, nil
	}

	cfg := m.cfg
	cfg.Database = strings.TrimSpace(m.askFields[askDBPathField].input.Value())

	m.submitting = true
	m.errMessage = ""
	m.sqlPreview = ""
	m.resultText = ""
	m.reportPath = ""
	m.status = "Generating SQL, running DuckDB, and preparing report..."

	return m, askCmd(cfg, ducklogai.Options{
		Prompt: prompt,
		Run:    true,
		Report: reportPath,
	})
}

func (m model) submitIngest() (tea.Model, tea.Cmd) {
	query, err := m.buildIngestQuery()
	if err != nil {
		m.errMessage = err.Error()
		m.successPath = ""
		m.status = "The Ingest form has validation issues."
		return m, nil
	}

	m.submitting = true
	m.errMessage = ""
	m.successPath = ""
	m.status = "Ingesting logs into DuckDB..."

	return m, ingestCmd(query)
}

func (m model) buildIngestQuery() (logs.Query, error) {
	lookback, err := logs.ParseLookback(m.ingestFields[ingestTimeWindowField].input.Value())
	if err != nil {
		return logs.Query{}, err
	}

	query := logs.Query{
		LogGroup: strings.TrimSpace(m.ingestFields[ingestLogGroupField].input.Value()),
		Region:   strings.TrimSpace(m.ingestFields[ingestRegionField].input.Value()),
		DBPath:   strings.TrimSpace(m.ingestFields[ingestDBPathField].input.Value()),
		Search:   strings.TrimSpace(m.ingestFields[ingestSearchField].input.Value()),
		Lookback: lookback,
	}

	if err := query.Validate(); err != nil {
		return logs.Query{}, err
	}

	return query, nil
}

func (m model) currentDBPath() string {
	if m.screen == screenIngest {
		return strings.TrimSpace(m.ingestFields[ingestDBPathField].input.Value())
	}
	return strings.TrimSpace(m.askFields[askDBPathField].input.Value())
}

func askCmd(cfg config.Config, opts ducklogai.Options) tea.Cmd {
	return func() tea.Msg {
		result, err := ducklogai.Ask(context.Background(), cfg, opts)
		return askCompleteMsg{result: result, err: err}
	}
}

func ingestCmd(query logs.Query) tea.Cmd {
	return func() tea.Msg {
		err := logs.Download(context.Background(), query)
		return ingestCompleteMsg{
			dbPath: query.DBPath,
			err:    err,
		}
	}
}
