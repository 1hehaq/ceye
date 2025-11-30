package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func displayHelp() {
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	argStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	printBanner()
	fmt.Println(successStyle.Render(" usage:"))
	fmt.Printf("    %s -target %s -webhook %s\n", cmdStyle.Render("ceye"), argStyle.Render("hackerone.com"), argStyle.Render("URL"))
	fmt.Printf("    %s -target %s\n\n", cmdStyle.Render("ceye"), argStyle.Render("bugcrowd.com"))

	fmt.Println(successStyle.Render(" options:"))
	fmt.Printf("    %s      target domain to monitor\n", flagStyle.Render("-target"))
	fmt.Printf("    %s     discord webhook URL\n", flagStyle.Render("-webhook"))
	fmt.Printf("    %s     show version\n", flagStyle.Render("-version"))
	fmt.Printf("    %s      update to latest version\n", flagStyle.Render("-update"))
	fmt.Printf("    %s    show this help message\n\n", flagStyle.Render("-h, -help"))

	fmt.Println(successStyle.Render(" requirements:"))
	fmt.Printf("    %s\n\n", argStyle.Render("docker must be installed and running"))

	fmt.Println(argStyle.Render(" monitor your targets real time via certificate transparency logs"))
	fmt.Println(argStyle.Render(" powered by github.com/d-Rickyy-b/certstream-server-go"))
	fmt.Println()
}