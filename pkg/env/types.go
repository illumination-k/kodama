package env

// EnvConfig represents environment variable configuration
type EnvConfig struct {
	DotenvFiles   []string `yaml:"dotenvFiles,omitempty"`
	ExcludeVars   []string `yaml:"excludeVars,omitempty"`
	SecretName    string   `yaml:"secretName,omitempty"`
	SecretCreated bool     `yaml:"secretCreated,omitempty"`
}

// DefaultExcludedVars contains system-critical variables that should never be overridden
// These variables are essential for pod operation and are always excluded
var DefaultExcludedVars = []string{
	// System variables (critical for shell and process operation)
	"PATH",
	"HOME",
	"USER",
	"SHELL",
	"TERM",
	"PWD",
	"OLDPWD",
	"HOSTNAME",
	"LOGNAME",

	// Kubernetes variables (injected by K8s, should never be overridden)
	"KUBERNETES_SERVICE_HOST",
	"KUBERNETES_SERVICE_PORT",
	"KUBERNETES_SERVICE_PORT_HTTPS",
	"KUBERNETES_PORT",
	"KUBERNETES_PORT_443_TCP",
	"KUBERNETES_PORT_443_TCP_PROTO",
	"KUBERNETES_PORT_443_TCP_PORT",
	"KUBERNETES_PORT_443_TCP_ADDR",
}
