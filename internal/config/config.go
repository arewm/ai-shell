package config

// Config represents the structure of ai-shell.yaml or config.yaml
type Config struct {
	EnvVars    []string   `mapstructure:"env_vars" yaml:"env_vars"`
	Mounts     []Mount    `mapstructure:"mounts" yaml:"mounts"`
	PodmanArgs []string   `mapstructure:"podman_args" yaml:"podman_args"`
	Registries []Registry `mapstructure:"registries" yaml:"registries"`
	SCMs       []SCM      `mapstructure:"scms" yaml:"scms"`
}

type Mount struct {
	Source  string `mapstructure:"source" yaml:"source"`
	Target  string `mapstructure:"target" yaml:"target"`
	Options string `mapstructure:"options" yaml:"options"`
}

type Registry struct {
	Registry    string `mapstructure:"registry" yaml:"registry"`
	UsernameEnv string `mapstructure:"username_env" yaml:"username_env"`
	TokenEnv    string `mapstructure:"token_env" yaml:"token_env"`
}

type SCM struct {
	Host        string `mapstructure:"host" yaml:"host"`
	TokenEnv    string `mapstructure:"token_env" yaml:"token_env"`
	UsernameEnv string `mapstructure:"username_env" yaml:"username_env"`
}
