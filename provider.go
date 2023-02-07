package config

// Root is a virtual key that accesses the entire configuration. Using it as
// the key when calling Provider.Get or Value.Get returns the whole
// configuration.
const Root = ""

// Provider is an abstraction over a configuration store, such as a collection
// of merged YAML, JSON, or TOML files.
type Provider interface {
	Name() string         // name of the configuration store
	Get(key string) Value // retrieves a portion of the configuration, see Value for details
}
