package mangrove

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Style definitions using lipgloss.
var (
	// SuccessStyle is used for success messages (green).
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	// WarningStyle is used for warning messages (yellow).
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	// ErrorStyle is used for error messages (red).
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	// InfoStyle is used for informational messages (blue).
	InfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	// RepoNameStyle is used for rendering repository names (bold).
	RepoNameStyle = lipgloss.NewStyle().Bold(true)

	// BranchNameStyle is used for rendering branch names (cyan).
	BranchNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// ProfileNameStyle is used for rendering profile names (magenta bold).
	ProfileNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)

	// DimStyle is used for less important text.
	DimStyle = lipgloss.NewStyle().Faint(true)

	// CleanBadge shows a clean status indicator.
	CleanBadge = SuccessStyle.Render("\u2713 clean")

	// HeaderStyle is used for section headers.
	HeaderStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)

// PrintSuccess prints a success message with a green checkmark.
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "  %s %s\n", SuccessStyle.Render("\u2713"), msg)
}

// PrintWarning prints a warning message with a yellow warning sign.
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "  %s %s\n", WarningStyle.Render("\u26a0"), msg)
}

// PrintError prints an error message with a red cross.
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "  %s %s\n", ErrorStyle.Render("\u2717"), msg)
}

// PrintInfo prints an informational message.
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "  %s\n", InfoStyle.Render(msg))
}

// PrintHeader prints a section header.
func PrintHeader(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "\n%s\n", HeaderStyle.Render(msg))
}

// ChangedBadge returns a styled changed files indicator.
func ChangedBadge(count int) string {
	return WarningStyle.Render(fmt.Sprintf("\u25cf %d changed", count))
}

// PrintRepoStatus prints a formatted status line for a repo within a workspace.
func PrintRepoStatus(repoName, branchName string, changedCount int, ahead, behind int, defaultBase string) {
	name := RepoNameStyle.Render(fmt.Sprintf("%-16s", repoName))
	branch := BranchNameStyle.Render(branchName)

	var status string
	if changedCount == 0 {
		status = CleanBadge
	} else {
		status = ChangedBadge(changedCount)
	}

	var aheadBehind string
	if ahead > 0 || behind > 0 {
		parts := []string{}
		if ahead > 0 {
			parts = append(parts, fmt.Sprintf("%d commits ahead", ahead))
		}
		if behind > 0 {
			parts = append(parts, fmt.Sprintf("%d commits behind", behind))
		}
		aheadBehind = DimStyle.Render(fmt.Sprintf("(%s of %s)", joinParts(parts), defaultBase))
	}

	line := fmt.Sprintf("  %s  %s  %s", name, branch, status)
	if aheadBehind != "" {
		line += "  " + aheadBehind
	}
	fmt.Fprintln(os.Stderr, line)
}

// FormatRepoStatusCompact returns a compact repo status string for list view.
func FormatRepoStatusCompact(repoName string, changedCount int) string {
	if changedCount == 0 {
		return fmt.Sprintf("[%s: %s]", repoName, CleanBadge)
	}
	return fmt.Sprintf("[%s: %s]", repoName, ChangedBadge(changedCount))
}

// joinParts joins string parts with " and ".
func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + " and " + parts[1]
}
