package config

import (
	"gopkg.in/yaml.v3"
)

type Serializer interface {
	Encode(v any) (out []byte, err error)
	Decode(blob []byte, v any) (err error)
}

// Driver interface.
type Driver interface {
	Name() string
	Aliases() []string // alias format names, use for resolve format name
	Serializer
	GetSerializer() Serializer
}

// StdDriver struct
type StdDriver struct {
	name       string
	aliases    []string
	serializer Serializer
}

func (d *StdDriver) GetSerializer() Serializer {
	return d.serializer
}

// NewDriver new std driver instance.
func NewDriver(name string, serializer Serializer) *StdDriver {
	return &StdDriver{name: name, serializer: serializer}
}

// WithAliases set aliases for driver
func (d *StdDriver) WithAliases(aliases ...string) *StdDriver {
	d.aliases = aliases
	return d
}

// WithAlias add alias for driver
func (d *StdDriver) WithAlias(alias string) *StdDriver {
	d.aliases = append(d.aliases, alias)
	return d
}

// Name of driver
func (d *StdDriver) Name() string { return d.name }

// Aliases format name of driver
func (d *StdDriver) Aliases() []string {
	return d.aliases
}

// Decode of driver
func (d *StdDriver) Decode(blob []byte, v any) (err error) {
	return d.serializer.Decode(blob, v)
}

// Encode of driver
func (d *StdDriver) Encode(v any) ([]byte, error) {
	return d.serializer.Encode(v)
}

/*************************************************************
 * Yaml driver
 *************************************************************/

// YamlDriver instance fot yaml
var YamlDriver = NewDriver("yaml", &yamlSerializer{}).WithAliases("yml")

// jsonDriver for json format content
type yamlSerializer struct {
}

// Decode for the driver
func (d *yamlSerializer) Decode(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

// Encode for the driver
func (d *yamlSerializer) Encode(v any) (out []byte, err error) {
	return yaml.Marshal(v)
}
