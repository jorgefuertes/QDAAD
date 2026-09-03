package console

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type TableModel struct {
	headers     []string
	rows        [][]string
	border      lipgloss.Border
	borderStyle lipgloss.Style
	horizontal  bool
	padding     int
	indent      int
}

func NewTable() *TableModel {
	return &TableModel{
		headers:     []string{},
		rows:        [][]string{},
		border:      lipgloss.DoubleBorder(),
		borderStyle: SubtitleStyle,
		padding:     1,
		indent:      2,
	}
}

func (m *TableModel) Indent(indent int) *TableModel {
	m.indent = indent

	return m
}

func (m *TableModel) Padding(padding int) *TableModel {
	m.padding = padding

	return m
}

func (m *TableModel) Horizontal(isHorizontal bool) *TableModel {
	m.horizontal = isHorizontal

	return m
}

func (m *TableModel) Header(headers ...string) *TableModel {
	m.headers = append(m.headers, headers...)

	return m
}

func (m *TableModel) Row(row ...string) *TableModel {
	if m.horizontal && len(row) >= 2 {
		row[0] = MutedStyle.Render(row[0])
		row[1] = ValueStyle.Render(row[1])
	}

	m.rows = append(m.rows, row)

	return m
}

func (m *TableModel) Border(border lipgloss.Border) *TableModel {
	m.border = border

	return m
}

func (m *TableModel) BorderStyle(style lipgloss.Style) *TableModel {
	m.borderStyle = style

	return m
}

func (m *TableModel) Print() {
	t := table.New().
		Border(m.border).BorderStyle(m.borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, m.padding)
		}).
		Headers(m.headers...).
		Rows(m.rows...)

	fmt.Println(lipgloss.NewStyle().MarginLeft(m.indent).Render(t.Render()))
}
