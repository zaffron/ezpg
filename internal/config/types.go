package config

type Config struct {
	Connections []Connection `yaml:"connections"`
	Settings    Settings     `yaml:"settings"`
	Path        string       `yaml:"-"`
}

type Connection struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
	URL      string `yaml:"url"`
	ReadOnly bool   `yaml:"readonly"`
}

type Settings struct {
	DefaultLimit       int    `yaml:"default_limit"`
	ConfirmDestructive bool   `yaml:"confirm_destructive"`
	EditorTabSize      int    `yaml:"editor_tab_size"`
	NullDisplay        string `yaml:"null_display"`
}

func DefaultSettings() Settings {
	return Settings{
		DefaultLimit:       100,
		ConfirmDestructive: true,
		EditorTabSize:      4,
		NullDisplay:        "NULL",
	}
}
