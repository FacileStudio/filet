package filet

import (
	"regexp"
	"slices"
)

// ConfigNames returns the filenames filet looks for, in priority order. The
// first is also what `filet init` writes.
//
// The undotted name comes first: this file is a guidelines document meant to be
// read, edited and argued with, not machinery to be hidden. The dotted spellings
// are still read, so a repository that has not been renamed keeps working.
func ConfigNames() []string {
	return []string{"filet.yml", "filet.yaml", ".filet.yml", ".filet.yaml"}
}

// Limits holds the numeric thresholds every size and complexity rule compares against.
type Limits struct {
	FileLines        int `yaml:"fileLines"`
	FuncsPerFile     int `yaml:"funcsPerFile"`
	FuncLines        int `yaml:"funcLines"`
	FuncStatements   int `yaml:"funcStatements"`
	LineLength       int `yaml:"lineLength"`
	Params           int `yaml:"params"`
	Returns          int `yaml:"returns"`
	Nesting          int `yaml:"nesting"`
	Complexity       int `yaml:"complexity"`
	StructFields     int `yaml:"structFields"`
	InterfaceMethods int `yaml:"interfaceMethods"`
}

// Architecture describes the file layout the project is supposed to respect.
type Architecture struct {
	RequiredDirs     []string            `yaml:"requiredDirs"`
	ForbiddenDirs    []string            `yaml:"forbiddenDirs"`
	MaxDepth         int                 `yaml:"maxDepth"`
	FileNamePattern  string              `yaml:"fileNamePattern"`
	RequiredFiles    map[string][]string `yaml:"requiredFiles"`
	ForbiddenImports map[string][]string `yaml:"forbiddenImports"`
}

// Treesitter configures the parse-tree analysis tier. Off by default so filet
// stays deterministic and offline until a project opts into vendored grammars.
type Treesitter struct {
	Enabled bool `yaml:"enabled"`
}

// Style holds the opinionated toggles that are a matter of team taste.
type Style struct {
	BanInlineComments  bool `yaml:"banInlineComments"`
	BanTODO            bool `yaml:"banTODO"`
	RequireDocComments bool `yaml:"requireDocComments"`
	BanGlobalMutable   bool `yaml:"banGlobalMutable"`
	BanInit            bool `yaml:"banInit"`
	BanTrailingSpace   bool `yaml:"banTrailingSpace"`
}

// Config is the full contents of filet.yml.
type Config struct {
	Preset       string       `yaml:"preset"`
	Ignore       []string     `yaml:"ignore"`
	Extensions   []string     `yaml:"extensions"`
	Limits       Limits       `yaml:"limits"`
	Architecture Architecture `yaml:"architecture"`
	Style        Style        `yaml:"style"`
	Treesitter   Treesitter   `yaml:"treesitter"`
	Disabled     []string     `yaml:"disabled"`
	FailOn       string       `yaml:"failOn"`

	root     string
	fileName *regexp.Regexp
}

// DefaultConfig returns the configuration used when a project has no config file.
func DefaultConfig() *Config {
	return &Config{
		Ignore: []string{
			".git", "node_modules", "vendor", "dist", "build", "target",
			"testdata", ".venv", "__pycache__", ".next", ".output",
			"_app", "coverage",
		},
		Extensions: []string{
			".go", ".ts", ".tsx", ".js", ".jsx", ".svelte", ".rs",
			".py", ".c", ".h", ".cpp", ".java", ".rb", ".sh",
		},
		Limits:       defaultLimits(),
		Architecture: Architecture{},
		Style:        defaultStyle(),
		Treesitter:   Treesitter{},
		FailOn:       "error",
	}
}

func defaultLimits() Limits {
	return Limits{
		FileLines:        250,
		FuncsPerFile:     8,
		FuncLines:        30,
		FuncStatements:   25,
		LineLength:       120,
		Params:           5,
		Returns:          3,
		Nesting:          4,
		Complexity:       10,
		StructFields:     12,
		InterfaceMethods: 4,
	}
}

func defaultStyle() Style {
	return Style{
		RequireDocComments: true,
		BanTrailingSpace:   true,
		BanInlineComments:  true,
		BanTODO:            true,
		BanGlobalMutable:   true,
		BanInit:            true,
	}
}

func (c *Config) compile() error {
	if c.Architecture.FileNamePattern == "" {
		return nil
	}
	re, err := regexp.Compile(c.Architecture.FileNamePattern)
	if err != nil {
		return err
	}
	c.fileName = re
	return nil
}

// Enabled reports whether a rule should run, honouring the disabled list.
func (c *Config) Enabled(rule string) bool {
	return !slices.Contains(c.Disabled, rule)
}

// Root returns the absolute directory the config was found in.
func (c *Config) Root() string { return c.root }

// UsesTreesitter reports whether the parse-tree tier owns ext's shape rules:
// the tier is enabled and a grammar is registered for the extension.
func (c *Config) UsesTreesitter(ext string) bool {
	return c.Treesitter.Enabled && HasGrammar(ext)
}
