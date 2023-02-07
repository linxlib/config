package config

import (
	"bytes"
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

// areSameYAML checks whether two values represent the same YAML data. It's
// only called from NewValue, where we must validate that the user-supplied
// value matches the contents of the user-supplied provider.
func areSameYAML(fromProvider, fromUser interface{}) (bool, error) {
	p, err := yaml.Marshal(fromProvider)
	if err != nil {
		// Unreachable with YAML provider, but possible if the provider is a
		// third-party implementation.
		return false, fmt.Errorf("can't represent %#v as YAML: %v", fromProvider, err)
	}
	u, err := yaml.Marshal(fromUser)
	if err != nil {
		return false, fmt.Errorf("can't represent %#v as YAML: %v", fromUser, err)
	}
	return bytes.Equal(p, u), nil
}
