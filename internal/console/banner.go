package console

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func Banner(text string, style lipgloss.Style) {
	fig := figure.NewFigure(text, "rectangles", false)
	banner := fig.String()

	println(style.Render(banner))
}
