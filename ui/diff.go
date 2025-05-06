package ui

import (
	"claude-squad/session"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	AdditionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	DeletionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	HunkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5e9"))
)

type DiffPane struct {
	viewport viewport.Model
	diff     string
	stats    string
	width    int
	height   int
}

func NewDiffPane() *DiffPane {
	return &DiffPane{
		viewport: viewport.New(0, 0),
	}
}

func (d *DiffPane) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width
	d.viewport.Height = height
	// Update viewport content if diff exists
	if d.diff != "" || d.stats != "" {
		d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, d.stats, d.diff))
	}
}

func (d *DiffPane) SetDiff(instance *session.Instance) {
	centeredFallbackMessage := lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		"No changes",
	)

	if instance == nil || !instance.Started() {
		d.viewport.SetContent(centeredFallbackMessage)
		return
	}
	
	// Special handling for simple mode (in-place) instances
	if instance.InPlace {
		// Create a more prominent warning message
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f59e0b")).
			Bold(true)
			
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60a5fa"))
			
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#f0dde4")).
			Bold(true).
			Padding(0, 1)
			
		// Get git status for current directory
		gitStatus := "Git status unavailable"
		gitCmd := exec.Command("git", "status", "-s")
		gitCmd.Dir = instance.Path
		gitStatusOutput, err := gitCmd.Output()
		if err == nil {
			if len(gitStatusOutput) == 0 {
				gitStatus = "No changes in working directory"
			} else {
				gitStatus = "Changes in working directory:\n\n" + string(gitStatusOutput)
			}
			
			// Also get branch info
			gitBranchCmd := exec.Command("git", "branch", "--show-current")
			gitBranchCmd.Dir = instance.Path
			gitBranchOutput, err := gitBranchCmd.Output()
			if err == nil {
				branch := strings.TrimSpace(string(gitBranchOutput))
				gitStatus = "Current branch: " + branch + "\n\n" + gitStatus
			}
		}
			
		header := headerStyle.Render(" SIMPLE MODE ACTIVE ")
		warningTitle := warningStyle.Render("⚠️  Working Directory Warning")
		warningMessage := "Changes are made directly to your working directory without isolation.\nThis means all file modifications will immediately affect your repository."
		
		infoTitle := infoStyle.Render("ℹ️  Git Operations")
		infoMessage := "• Changes can be committed with the Submit button (p)\n• Commits are made directly to your current branch\n• Use Submit (p) to commit and push changes\n• For branch isolation, consider using standard mode instead"
		
		gitStatusTitle := infoStyle.Render("🔍 Git Status")
		
		simpleModeMessage := lipgloss.JoinVertical(
			lipgloss.Center,
			"",
			header,
			"",
			warningTitle,
			warningMessage,
			"",
			infoTitle,
			infoMessage,
			"",
			gitStatusTitle,
			gitStatus,
			"",
		)
		
		centeredMessage := lipgloss.Place(
			d.width,
			d.height,
			lipgloss.Center,
			lipgloss.Center,
			simpleModeMessage,
		)
		d.viewport.SetContent(centeredMessage)
		return
	}

	stats := instance.GetDiffStats()
	if stats == nil {
		// Show loading message if worktree is not ready
		centeredMessage := lipgloss.Place(
			d.width,
			d.height,
			lipgloss.Center,
			lipgloss.Center,
			"Setting up worktree...",
		)
		d.viewport.SetContent(centeredMessage)
		return
	}

	if stats.Error != nil {
		// Show error message
		centeredMessage := lipgloss.Place(
			d.width,
			d.height,
			lipgloss.Center,
			lipgloss.Center,
			fmt.Sprintf("Error: %v", stats.Error),
		)
		d.viewport.SetContent(centeredMessage)
		return
	}

	if stats.IsEmpty() {
		d.stats = ""
		d.diff = ""
		d.viewport.SetContent(centeredFallbackMessage)
	} else {
		additions := AdditionStyle.Render(fmt.Sprintf("%d additions(+)", stats.Added))
		deletions := DeletionStyle.Render(fmt.Sprintf("%d deletions(-)", stats.Removed))
		d.stats = lipgloss.JoinHorizontal(lipgloss.Center, additions, " ", deletions)
		d.diff = colorizeDiff(stats.Content)
		d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, d.stats, d.diff))
	}
}

func (d *DiffPane) String() string {
	return d.viewport.View()
}

// ScrollUp scrolls the viewport up
func (d *DiffPane) ScrollUp() {
	d.viewport.LineUp(1)
}

// ScrollDown scrolls the viewport down
func (d *DiffPane) ScrollDown() {
	d.viewport.LineDown(1)
}

func colorizeDiff(diff string) string {
	var coloredOutput strings.Builder

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if len(line) > 0 {
			if strings.HasPrefix(line, "@@") {
				// Color hunk headers cyan
				coloredOutput.WriteString(HunkStyle.Render(line) + "\n")
			} else if line[0] == '+' && (len(line) == 1 || line[1] != '+') {
				// Color added lines green, excluding metadata like '+++'
				coloredOutput.WriteString(AdditionStyle.Render(line) + "\n")
			} else if line[0] == '-' && (len(line) == 1 || line[1] != '-') {
				// Color removed lines red, excluding metadata like '---'
				coloredOutput.WriteString(DeletionStyle.Render(line) + "\n")
			} else {
				// Print metadata and unchanged lines without color
				coloredOutput.WriteString(line + "\n")
			}
		} else {
			// Preserve empty lines
			coloredOutput.WriteString("\n")
		}
	}

	return coloredOutput.String()
}
