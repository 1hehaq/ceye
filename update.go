package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func performUpdate() {
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println()
	fmt.Println(highlightStyle.Render("checking for updates..."))

	latest, found, err := selfupdate.DetectLatest("1hehaq/ceye")
	if err != nil {
		fmt.Printf("%s %s\n", errorStyle.Render("✗"), dimStyle.Render("error checking for updates: "+err.Error()))
		fmt.Println()
		os.Exit(1)
	}

	if !found {
		fmt.Printf("%s %s\n", errorStyle.Render("✗"), dimStyle.Render("no releases found"))
		fmt.Println()
		os.Exit(1)
	}

	currentVersion := "v" + version
	v, err := semver.ParseTolerant(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		fmt.Printf("%s %s\n", errorStyle.Render("✗"), dimStyle.Render("invalid version format: "+err.Error()))
		fmt.Println()
		os.Exit(1)
	}

	if !latest.Version.GT(v) {
		fmt.Printf("%s %s\n", successStyle.Render("✓"), dimStyle.Render("already up to date ("+currentVersion+")"))
		fmt.Println()
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("%s %s\n", errorStyle.Render("✗"), dimStyle.Render("could not locate executable: "+err.Error()))
		fmt.Println()
		os.Exit(1)
	}

	fmt.Printf("  %s → %s\n", dimStyle.Render(currentVersion), highlightStyle.Render(latest.Version.String()))
	fmt.Println()
	fmt.Print(dimStyle.Render("  updating... "))

	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		fmt.Printf("%s\n", errorStyle.Render("failed"))
		fmt.Printf("  %s\n", dimStyle.Render("error: "+err.Error()))
		fmt.Println()
		os.Exit(1)
	}

	fmt.Printf("%s\n", successStyle.Render("done"))
	fmt.Println()
	fmt.Println(dimStyle.Render("  restart ceye to use the new version"))
	fmt.Println()
}
