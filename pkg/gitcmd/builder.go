package gitcmd

import (
	"fmt"
	"strings"
)

// CloneOptions contains options for git clone command
type CloneOptions struct {
	Branch       string // Branch to clone
	Depth        int    // Clone depth (0 for full clone)
	SingleBranch bool   // Clone only specified branch
	ExtraArgs    string // Additional git clone arguments
}

// BuildCloneCommandScript builds a bash script for git clone with token injection
// This is used by init containers and can be reused for other purposes
func BuildCloneCommandScript(repoURL string, opts *CloneOptions) string {
	var script strings.Builder

	script.WriteString("set -e\n")
	script.WriteString("echo 'Installing git...'\n")
	script.WriteString("apt-get update -qq && apt-get install -y -qq git\n\n")

	script.WriteString("echo 'Cloning repository...'\n")
	script.WriteString(fmt.Sprintf("REPO_URL='%s'\n", repoURL))

	// Inject token for HTTPS URLs
	script.WriteString(`
if [[ "$REPO_URL" == https://* ]] && [ -n "$GH_TOKEN" ]; then
    # Inject token into HTTPS URL
    CLONE_URL="${REPO_URL/https:\/\//https://${GH_TOKEN}@}"
else
    CLONE_URL="$REPO_URL"
fi
`)

	// Build clone command
	script.WriteString("git clone")
	if opts != nil && opts.Depth > 0 {
		script.WriteString(fmt.Sprintf(" --depth %d", opts.Depth))
	}
	if opts != nil && opts.SingleBranch {
		script.WriteString(" --single-branch")
	}
	if opts != nil && opts.Branch != "" {
		script.WriteString(fmt.Sprintf(" --branch '%s'", opts.Branch))
	}
	if opts != nil && opts.ExtraArgs != "" {
		script.WriteString(fmt.Sprintf(" %s", opts.ExtraArgs))
	}
	script.WriteString(" \"$CLONE_URL\" /workspace\n\n")

	script.WriteString("echo 'Repository clone complete'\n")
	return script.String()
}

// BuildBranchSetupScript builds a bash script for creating/checking out feature branches
// This protects main branches by auto-creating feature branches when needed
func BuildBranchSetupScript(targetBranch string) string {
	if targetBranch == "" {
		return ""
	}

	var script strings.Builder

	script.WriteString("cd /workspace\n")
	script.WriteString("CURRENT_BRANCH=$(git branch --show-current)\n")
	script.WriteString("echo \"Current branch: $CURRENT_BRANCH\"\n\n")

	script.WriteString("# Create feature branch if on protected branch\n")
	script.WriteString(`if [[ "$CURRENT_BRANCH" =~ ^(main|master|trunk|development)$ ]]; then
    echo "Creating feature branch: ` + targetBranch + `"
    git checkout -b "` + targetBranch + `"
else
    echo "Branch setup complete (on branch: $CURRENT_BRANCH)"
fi
`)

	return script.String()
}

// BuildGitInitScript builds a complete initialization script for git repository setup
// Combines clone and branch setup into one script for init containers
func BuildGitInitScript(repoURL, targetBranch string, opts *CloneOptions) string {
	var script strings.Builder

	// Add clone script
	script.WriteString(BuildCloneCommandScript(repoURL, opts))
	script.WriteString("\n")

	// Add branch setup script if target branch specified
	if targetBranch != "" {
		script.WriteString(BuildBranchSetupScript(targetBranch))
		script.WriteString("\n")
	}

	script.WriteString("echo 'Repository setup complete'\n")
	return script.String()
}

// ValidateCloneArgs performs basic validation on extra git clone arguments
// to prevent command injection or dangerous options
func ValidateCloneArgs(args string) error {
	if args == "" {
		return nil
	}

	// Disallow dangerous patterns
	dangerousPatterns := []string{
		"|", "&&", "||", ";", "`", "$(", // Command injection
		"--upload-pack", "--config", // Potentially dangerous git options
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(args, pattern) {
			return fmt.Errorf("git clone args contain disallowed pattern: %s", pattern)
		}
	}

	return nil
}
