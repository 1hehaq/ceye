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
	fmt.Printf("    cat targets.txt | %s -target -\n", cmdStyle.Render("ceye"))
	fmt.Printf("    %s -target %s -config %s -notify=%s\n", cmdStyle.Render("ceye"), argStyle.Render("example.com"), argStyle.Render("custom.yaml"), argStyle.Render("discord"))
	fmt.Printf("    %s -target %s\n", cmdStyle.Render("ceye"), argStyle.Render("targets.txt"))
	fmt.Printf("    echo \"@reboot %s %s -target %s > /tmp/ceye.log 2>&1 &\" | %s -\n\n", cmdStyle.Render("nohup"), cmdStyle.Render("ceye"), argStyle.Render("example.com"), cmdStyle.Render("crontab"))

	fmt.Println(successStyle.Render(" options:"))
	fmt.Printf("    %s      target domain to monitor (file path, single domain, or '-' for stdin)\n", flagStyle.Render("-target"))
	fmt.Printf("    %s      path to configuration file (default: ~/.config/ceye/provider.yaml)\n", flagStyle.Render("-config"))
	fmt.Printf("    %s      notification provider: discord, telegram, both\n", flagStyle.Render("-notify"))
	fmt.Printf("    %s     show version\n", flagStyle.Render("-version"))
	fmt.Printf("    %s      update to latest version\n", flagStyle.Render("-update"))
	fmt.Printf("    %s    show this help message\n\n", flagStyle.Render("-h, -help"))

	fmt.Println(successStyle.Render(" configuration:"))
	fmt.Printf("    %s config file location: ~/.config/ceye/provider.yaml\n", argStyle.Render("•"))
	fmt.Printf("    %s supports multiple targets and notification providers\n\n", argStyle.Render("•"))

	fmt.Println(successStyle.Render(" requirements:"))
	fmt.Printf("    %s\n\n", argStyle.Render("docker must be installed and running"))

	fmt.Println(argStyle.Render(" monitor your targets real time via certificate transparency logs"))
	fmt.Println(argStyle.Render(" powered by github.com/d-Rickyy-b/certstream-server-go"))
	fmt.Println()
}
