package config

import (
	"bytes"
	"fmt"
	"github.com/linxlib/config/internal/merge"
	"github.com/linxlib/config/internal/unreachable"
	"io"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const _separator = "."

// YAML is a provider that reads from one or more YAML sources. Many aspects
// of the resulting provider's behavior can be altered by passing functional
// options.
//
// By default, the YAML provider attempts to proactively catch common mistakes
// by enabling gopkg.in/yaml.v2's strict mode. See the package-level
// documentation on strict unmarshalling for details.
//
// When populating Go structs, values produced by the YAML provider correctly
// handle all struct tags supported by gopkg.in/yaml.v2. See
// https://godoc.org/gopkg.in/yaml.v2#Marshal for details.
type YAML struct {
	name     string
	raw      [][]byte
	lookup   LookupFunc // see withDefault
	contents interface{}
	strict   bool
	empty    bool
}

// NewYAML constructs a YAML provider. See the various YAMLOptions for
// available tweaks to the default behavior.
func NewYAML(options ...YAMLOption) (*YAML, error) {
	cfg := &config{
		strict: true,
		name:   "YAML",
	}
	for _, o := range options {
		o.apply(cfg)
	}

	if cfg.err != nil {
		return nil, fmt.Errorf("error applying options: %v", cfg.err)
	}

	// Some sources shouldn't have environment variables expanded; protect those
	// sources by escaping the contents. (Expanding before merging re-exposes a
	// number of bugs, so we can't just selectively expand sources before
	// merging.)
	sourceBytes := make([][]byte, len(cfg.sources))
	for i := range cfg.sources {
		s := cfg.sources[i]
		if !s.raw {
			sourceBytes[i] = s.bytes
			continue
		}
		sourceBytes[i] = escapeVariables(s.bytes)
	}

	// On construction, go through a full merge-serialize-deserialize cycle to
	// catch any duplicated keys as early as possible (in strict mode). It also
	// strips comments, which stops us from attempting environment variable
	// expansion. (We'll expand environment variables next.)
	merged, err := merge.YAML(sourceBytes, cfg.strict)
	if err != nil {
		return nil, fmt.Errorf("couldn't merge YAML sources: %v", err)
	}

	// Expand environment variables.
	merged, err = expandVariables(cfg.lookup, merged)
	if err != nil {
		return nil, err
	}

	y := &YAML{
		name:   cfg.name,
		raw:    sourceBytes,
		lookup: cfg.lookup,
		strict: cfg.strict,
	}

	dec := yaml.NewDecoder(merged)
	dec.SetStrict(cfg.strict)
	if err := dec.Decode(&y.contents); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("couldn't decode merged YAML: %v", err)
		}
		y.empty = true
	}

	return y, nil
}

// Name returns the name of the provider. It defaults to "YAML".
func (y *YAML) Name() string {
	return y.name
}

// Get retrieves a value from the configuration. The supplied key is treated
// as a period-separated path, with each path segment used as a map key. For
// example, if the provider contains the YAML
//
//	foo:
//	  bar:
//	    baz: hello
//
// then Get("foo.bar") returns a value holding
//
//	baz: hello
//
// To get a value holding the entire configuration, use the Root constant as
// the key.
func (y *YAML) Get(key string) Value {
	return y.get(strings.Split(key, _separator))
}

func (y *YAML) get(path []string) Value {
	if len(path) == 1 && path[0] == Root {
		path = nil
	}
	return Value{
		path:     path,
		provider: y,
	}
}

// at returns the unmarshalled representation of the value at a given path,
// with a bool indicating whether the value was found.
//
// YAML mappings are unmarshalled as map[interface{}]interface{}, sequences as
// []interface{}, and scalars as interface{}.
func (y *YAML) at(path []string) (interface{}, bool) {
	if y.empty {
		return nil, false
	}

	cur := y.contents
	for _, segment := range path {
		// Cast to a mapping type. If this fails, then we ended up on a path
		// that didn't terminate on a sequence or a scalar.
		m, ok := cur.(map[interface{}]interface{})
		if !ok {
			return nil, false
		}

		// Try resolving the segment as a string and then unmarshal the path
		// segment for a comparable key. After all, YAML scalar types are more
		// than strings (boolean, integer, etc). We'll prefer a string form to
		// resolve ambiguous paths.
		if _, ok := m[segment]; !ok {
			var key interface{}
			if err := yaml.Unmarshal([]byte(segment), &key); err != nil {
				return nil, false
			}
			if !merge.IsScalar(key) {
				return nil, false
			}
			if _, ok := m[key]; !ok {
				return nil, false
			}
			cur = m[key]
		} else {
			cur = m[segment]
		}
	}
	return cur, true
}

func (y *YAML) populate(path []string, i interface{}) error {
	val, ok := y.at(path)
	if !ok {
		return nil
	}
	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(val); err != nil {
		// Provider contents were produced by unmarshaling YAML, this isn't
		// possible.
		err := fmt.Errorf(
			"couldn't marshal config at key %s to YAML: %v",
			strings.Join(path, _separator),
			err,
		)
		return unreachable.Wrap(err)
	}
	dec := yaml.NewDecoder(buf)
	dec.SetStrict(y.strict)
	// Decoding can't ever return EOF, since encoding any value is guaranteed to
	// produce non-empty YAML.
	return dec.Decode(i)
}

func (y *YAML) withDefault(d interface{}) (*YAML, error) {
	rawDefault := &bytes.Buffer{}
	if err := yaml.NewEncoder(rawDefault).Encode(d); err != nil {
		return nil, fmt.Errorf("can't marshal default to YAML: %v", err)
	}

	// It's possible that one of the sources used when initially configuring the
	// provider was nothing but a top-level null, but that a higher-priority
	// source included some additional data. In that case, the result of merging
	// all the sources is non-null. However, the explicitly-null source should
	// override all data provided by withDefault. To handle this correctly, we
	// must use the new defaults as the lowest-priority source and re-merge the
	// original sources.
	opts := []YAMLOption{
		Name(y.name),
		Expand(y.lookup),
		Source(rawDefault),
		// y.raw contains the original sources with escaping for RawSources so
		// appendSourcs won't double-expand them.
		appendSources(y.raw),
	}
	if !y.strict {
		opts = append(opts, Permissive())
	}
	return NewYAML(opts...)
}

// A Value is a subset of a provider's configuration.
type Value struct {
	path     []string
	provider *YAML
}

// Source returns the name of the value's provider.
func (v Value) Source() string {
	return v.provider.Name()
}

// Populate unmarshals the value into the target struct, much like
// json.Unmarshal or yaml.Unmarshal. When populating a struct with some fields
// already set, data is deep-merged as described in the package-level
// documentation.
func (v Value) Populate(target interface{}) error {
	return v.provider.populate(v.path, target)
}

// Get dives further into the configuration, pulling out more deeply nested
// values. The supplied path is split on periods, and each segment is treated
// as a nested map key. For example, if the current value holds the YAML
// configuration
//
//	foo:
//	  bar:
//	    baz: quux
//
// then a call to Get("foo.bar") will hold the YAML mapping
//
//	baz: quux
func (v Value) Get(path string) Value {
	if path == Root {
		return v
	}
	extended := make([]string, len(v.path))
	copy(extended, v.path)
	extended = append(extended, strings.Split(path, _separator)...)
	return v.provider.get(extended)
}

// has checks whether any configuration is available at this key.
//
// It doesn't distinguish between configuration supplied during provider
// construction and configuration applied by WithDefault. If the value has
// explicitly been set to nil, HasValue is true.
func (v Value) has() bool {
	_, ok := v.provider.at(v.path)
	return ok
}

func (v Value) String() string {
	return fmt.Sprint(v.value())
}

// Value unmarshal the configuration into interface{}.
func (v Value) value() interface{} {
	// Simplest way to ensure that the caller can't mutate the configuration is
	// to deep-copy with Populate.
	var i interface{}
	if err := v.Populate(&i); err != nil {
		// Unreachable, since we've already ensured that the underlying YAML is
		// valid. Can't alter this signature to include an error without breaking
		// backward compatibility.
		panic(unreachable.Wrap(err).Error())
	}
	return i
}
