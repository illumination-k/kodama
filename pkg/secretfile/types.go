package secretfile

// SecretFileConfig holds configuration for secret file injection
type SecretFileConfig struct {
	Files         []FileMapping `yaml:"files,omitempty"`
	SecretName    string        `yaml:"secretName,omitempty"`
	SecretCreated bool          `yaml:"secretCreated,omitempty"`
}

// FileMapping represents a mapping from a local file to a pod destination
type FileMapping struct {
	Source      string `yaml:"source"`      // Local file path (supports ~ expansion)
	Destination string `yaml:"destination"` // Pod file path (must be absolute)
}
