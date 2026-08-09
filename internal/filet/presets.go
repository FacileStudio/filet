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
