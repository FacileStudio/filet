package filet

import "slices"

var presets = map[string]func(*Config){
	"default": func(*Config) {},
	"epitech": func(c *Config) {
		c.Limits.FuncsPerFile = 5
		c.Limits.FuncLines = 20
		c.Limits.FuncStatements = 15
		c.Limits.Params = 4
		c.Limits.Nesting = 3
		c.Limits.LineLength = 80
		c.Limits.Complexity = 10
		c.Style.BanInlineComments = true
		c.Style.BanTODO = true
		c.Style.BanGlobalMutable = true
	},
	"relaxed": func(c *Config) {
		c.Limits.FuncsPerFile = 0
		c.Limits.FuncLines = 60
		c.Limits.FuncStatements = 40
		c.Limits.FileLines = 400
		c.Limits.Nesting = 5
		c.Limits.Complexity = 15
		c.Limits.StructFields = 15
		c.Limits.InterfaceMethods = 5
		c.Architecture.MaxDepth = 6
		c.Style.BanInlineComments = false
		c.Style.BanTODO = false
		c.Style.BanGlobalMutable = false
		c.Style.BanInit = false
	},
	"facile":        facilePreset,
	"facile-strict": facileStrictPreset,
}

// facileStrictPreset is facile plus the strict Go family (style.strict). It is
// opt-in: no repository gets the go.strict.* rules without naming this preset
// or setting the key itself.
func facileStrictPreset(c *Config) {
	facilePreset(c)
	c.Style.Strict = true
}

// facilePreset is the Facile suite standard: the filet.yml that Sonde, Capsule
// and Sablier share, so a suite repository's config reduces to `preset: facile`
// plus its own exceptions. The values are pinned here rather than read from the
// defaults, so moving a default never moves the suite.
func facilePreset(c *Config) {
	c.Ignore = []string{"node_modules", "vendor", "build", "dist", "testdata", "coverage"}
	c.Extensions = []string{".go", ".ts", ".svelte", ".sh"}
	c.Limits = Limits{
		FileLines: 250, FuncsPerFile: 8, FuncLines: 30, FuncStatements: 25, LineLength: 120,
		Params: 5, Returns: 3, Nesting: 4, Complexity: 10, StructFields: 12, InterfaceMethods: 4,
	}
	c.Architecture.RequiredDirs = []string{"apps/api", "apps/client"}
	c.Architecture.RequiredFiles = map[string][]string{"apps/api/modules/*": {"router.go"}}
	c.Style = Style{
		BanInlineComments: true, BanTODO: true, RequireDocComments: true, BanGlobalMutable: true,
		BanInit: true, BanTrailingSpace: true, Format: true,
	}
	c.Disabled = []string{"gen.line.long"}
	c.FailOn = "error"
}

// Preset returns the limit-seeding function registered under name.
func Preset(name string) (func(*Config), bool) {
	apply, ok := presets[name]
	return apply, ok
}

// PresetNames returns every preset name, sorted.
func PresetNames() []string {
	names := make([]string, 0, len(presets))
	for n := range presets {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}
