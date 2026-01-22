package env

// EnvConfig represents environment variable configuration
type EnvConfig struct {
	DotenvFiles   []string `yaml:"dotenvFiles,omitempty"`
	ExcludeVars   []string `yaml:"excludeVars,omitempty"`
	SecretName    string   `yaml:"secretName,omitempty"`
	SecretCreated bool     `yaml:"secretCreated,omitempty"`
}

// DefaultExcludedVars contains variables that should never be overridden
// to prevent breaking the pod environment or exposing security risks
var DefaultExcludedVars = []string{
	// System variables
	"PATH",
	"HOME",
	"USER",
	"SHELL",
	"TERM",
	"PWD",
	"OLDPWD",
	"HOSTNAME",
	"LOGNAME",

	// Kubernetes variables
	"KUBERNETES_SERVICE_HOST",
	"KUBERNETES_SERVICE_PORT",
	"KUBERNETES_SERVICE_PORT_HTTPS",
	"KUBERNETES_PORT",
	"KUBERNETES_PORT_443_TCP",
	"KUBERNETES_PORT_443_TCP_PROTO",
	"KUBERNETES_PORT_443_TCP_PORT",
	"KUBERNETES_PORT_443_TCP_ADDR",

	// Claude Code variables
	"CLAUDE_CODE_AUTH_TOKEN",
	"CLAUDE_AUTH_FILE",
	"ANTHROPIC_API_KEY",
}
