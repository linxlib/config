package config

// NopProvider is a no-op provider.
type NopProvider struct{}

var _ Provider = NopProvider{}

// Name implements Provider.
func (NopProvider) Name() string {
	return "no-op"
}

// Get returns a value with no configuration available.
func (n NopProvider) Get(_ string) Value {
	p, _ := NewYAML(Name(n.Name()))
	return p.Get("not_there") // Root is always present, making HasValue true
}
