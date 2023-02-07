package config

import (
	"fmt"
)

type scopedProvider struct {
	Provider

	prefix string
}

var _ Provider = (*scopedProvider)(nil)

func (s *scopedProvider) Get(key string) Value {
	return s.Provider.Get(fmt.Sprintf("%s.%s", s.prefix, key))
}

// NewScopedProvider wraps a provider and adds a prefix to all Get calls.
func NewScopedProvider(prefix string, provider Provider) Provider {
	if prefix == "" {
		return provider
	}
	return &scopedProvider{provider, prefix}
}

// NewProviderGroup composes multiple providers, with later providers
// overriding earlier ones. The merge logic is described in the package-level
// documentation. To preserve backward compatibility, the resulting provider
// disables strict unmarshalling.
//
// Prefer using NewYAML instead of this where possible. NewYAML gives you
// strict unmarshalling by default and allows use of other options at the same
// time.
func NewProviderGroup(name string, providers ...Provider) (Provider, error) {
	opts := make([]YAMLOption, 0, len(providers)+2)
	opts = append(opts, Name(name), Permissive())
	for _, p := range providers {
		if v := p.Get(Root); v.has() {
			opts = append(opts, Static(v.value()))
		}
	}
	return NewYAML(opts...)
}
