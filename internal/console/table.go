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
		border:      lipgloss.RoundedBorder(),
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

// Row adds a row. The cells go in as they come: what they end up looking like is
// decided when the table is printed, so that nothing here has to know about
// colour and a caller's strings are left alone.
func (m *TableModel) Row(row ...string) *TableModel {
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
		StyleFunc(func(_, col int) lipgloss.Style {
			cell := lipgloss.NewStyle().Padding(0, m.padding)

			if !m.horizontal {
				return cell
			}

			// A horizontal table is a list of label and value, so the first
			// column names what the second one says. Inherit keeps the padding
			// set above and takes the colour from the style named.
			switch col {
			case 0:
				return cell.Inherit(MutedStyle)
			case 1:
				return cell.Inherit(ValueStyle)
			}

			return cell
		}).
		Headers(m.headers...).
		Rows(m.rows...)

	fmt.Println(lipgloss.NewStyle().MarginLeft(m.indent).Render(t.Render()))
}
