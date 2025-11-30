package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func displayVersion() {
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println()
	fmt.Printf("%s %s\n", highlightStyle.Render("ceye"), dimStyle.Render("v"+version))
	fmt.Println()
}
