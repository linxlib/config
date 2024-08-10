package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (c *Config) getENVPrefix() string {
	if c.Option.ENVPrefix == "" {
		if prefix := os.Getenv("CONFIG_ENV_PREFIX"); prefix != "" {
			return prefix
		}
		return "CONFIG"
	}
	return c.Option.ENVPrefix
}

func (c *Config) getConfigurationFileWithENVPrefix(file, env string) (string, time.Time, error) {
	stat := os.Stat
	if c.FS != nil {
		stat = func(name string) (os.FileInfo, error) {
			return fs.Stat(c.FS, name)
		}
	}
	var (
		envFile string
		extname = path.Ext(file)
	)

	if extname == "" {
		envFile = fmt.Sprintf("%v.%v", file, env)
	} else {
		envFile = fmt.Sprintf("%v.%v%v", strings.TrimSuffix(file, extname), env, extname)
	}

	if fileInfo, err := stat(envFile); err == nil && fileInfo.Mode().IsRegular() {
		return envFile, fileInfo.ModTime(), nil
	}
	return "", time.Now(), fmt.Errorf("failed to find file %v", file)
}

func (c *Config) getConfigurationFiles(config *Option, watchMode bool, files ...string) ([]string, map[string]time.Time) {
	c.files = files
	stat := os.Stat
	if config.FS != nil {
		stat = func(name string) (os.FileInfo, error) {
			return fs.Stat(config.FS, name)
		}
	}

	var resultKeys []string
	var results = map[string]time.Time{}

	if !watchMode && (c.Option.Debug || c.Option.Verbose) {
		fmt.Printf("Current environment: '%v'\n", c.GetEnvironment())
	}

	for i := len(files) - 1; i >= 0; i-- {
		foundFile := false
		file := files[i]

		// check configuration
		if fileInfo, err := stat(file); err == nil && fileInfo.Mode().IsRegular() {
			foundFile = true
			resultKeys = append(resultKeys, file)
			results[file] = fileInfo.ModTime()
		}

		// check configuration with env
		if file, modTime, err := c.getConfigurationFileWithENVPrefix(file, c.GetEnvironment()); err == nil {
			foundFile = true
			resultKeys = append(resultKeys, file)
			results[file] = modTime
		}

		// check example configuration
		if !foundFile {
			if example, modTime, err := c.getConfigurationFileWithENVPrefix(file, "example"); err == nil {
				if !watchMode && !c.Silent {
					fmt.Printf("Failed to find configuration %v, using example file %v\n", file, example)
				}
				resultKeys = append(resultKeys, example)
				results[example] = modTime
			} else if !c.Silent {
				fmt.Printf("Failed to find configuration %v\n", file)
			}
		}
	}
	return resultKeys, results
}

type RawNode struct {
	*yaml.Node
}

func (n *RawNode) UnmarshalYAML(node *yaml.Node) error {
	n.Node = node
	return nil
}

func decodeYaml(data []byte, key string, config any) error {
	if key != "" {
		var target map[string]*RawNode
		err := yaml.Unmarshal(data, &target)
		if err != nil {
			return err
		}
		if v, ok := target[key]; ok {
			return v.Decode(config)
		}
		return nil
	} else {
		err := yaml.Unmarshal(data, config)
		if err != nil {
			return err
		}
		return nil
	}
}
func decodeJson(data []byte, key string, config any) error {
	if key != "" {
		var target map[string]json.RawMessage
		err := unmarshalJSON(data, &target)
		if err != nil {
			return err
		}
		if v, ok := target[key]; ok {
			return unmarshalJSON(v, config)
		}
		return nil
	} else {
		err := unmarshalJSON(data, config)
		if err != nil {
			return err
		}
		return nil
	}
}

func (c *Config) processFileWithKey(key string, config interface{}, file string) error {
	readFile := os.ReadFile
	if c.FS != nil {
		readFile = func(filename string) ([]byte, error) {
			return fs.ReadFile(c.FS, filename)
		}
	}
	data, err := readFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		return decodeYaml(data, key, config)
	case strings.HasSuffix(file, ".json"):
		var target map[string]json.RawMessage
		err := unmarshalJSON(data, &target)
		if err != nil {
			return err
		}
		if v, ok := target[key]; ok {
			return unmarshalJSON(v, config)
		} else {
			return nil
		}
	default:
		err := decodeJson(data, key, config)
		if err != nil {
			return err
		}
		yamlError := decodeYaml(data, key, config)
		if yamlError == nil {
			return nil
		} else {
			var yErr *yaml.TypeError
			if errors.As(yamlError, &yErr) {
				return yErr
			}
		}
		return errors.New("failed to decode config")
	}
}
func (c *Config) processFile(config interface{}, file string) error {
	return c.processFileWithKey("", config, file)
}

// unmarshalJSON unmarshals the given data into the config interface.
// If the errorOnUnmatchedKeys boolean is true, an error will be returned if there
// are keys in the data that do not match fields in the config interface.
func unmarshalJSON(data []byte, config interface{}) error {
	reader := strings.NewReader(string(data))
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(config)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func getPrefixForStruct(prefixes []string, fieldStruct *reflect.StructField) []string {
	if fieldStruct.Anonymous && fieldStruct.Tag.Get("anonymous") == "true" {
		return prefixes
	}
	return append(prefixes, fieldStruct.Name)
}

func (c *Config) processDefaults(config interface{}) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	if configValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	configType := configValue.Type()
	for i := 0; i < configType.NumField(); i++ {
		var (
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
		)

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank {
			// Set default configuration if blank
			if value := fieldStruct.Tag.Get("default"); value != "" {
				if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
					return err
				}
			}
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			if err := c.processDefaults(field.Addr().Interface()); err != nil {
				return err
			}
		case reflect.Slice:
			for i := 0; i < field.Len(); i++ {
				if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
					if err := c.processDefaults(field.Index(i).Addr().Interface()); err != nil {
						return err
					}
				}
			}
		default:

		}
	}

	return nil
}

func (c *Config) processTags(config interface{}, prefixes ...string) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	if configValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	configType := configValue.Type()
	for i := 0; i < configType.NumField(); i++ {
		var (
			envNames    []string
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
			envName     = fieldStruct.Tag.Get("env") // read configuration from shell env
		)

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		if envName == "" {
			envNames = append(envNames, strings.Join(append(prefixes, fieldStruct.Name), "_"))                  // Configor_DB_Name
			envNames = append(envNames, strings.ToUpper(strings.Join(append(prefixes, fieldStruct.Name), "_"))) // CONFIGOR_DB_NAME
		} else {
			envNames = []string{envName}
		}

		if c.Option.Verbose {
			fmt.Printf("Trying to load struct `%v`'s field `%v` from env %v\n", configType.Name(), fieldStruct.Name, strings.Join(envNames, ", "))
		}

		// Load From Shell ENV
		for _, env := range envNames {
			if value := os.Getenv(env); value != "" {
				if c.Option.Debug || c.Option.Verbose {
					fmt.Printf("Loading configuration for struct `%v`'s field `%v` from env %v...\n", configType.Name(), fieldStruct.Name, env)
				}

				switch reflect.Indirect(field).Kind() {
				case reflect.Bool:
					switch strings.ToLower(value) {
					case "", "0", "f", "false":
						field.Set(reflect.ValueOf(false))
					default:
						field.Set(reflect.ValueOf(true))
					}
				case reflect.String:
					field.Set(reflect.ValueOf(value))
				default:
					if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
						return err
					}
				}
				break
			}
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank && fieldStruct.Tag.Get("required") == "true" {
			// return error if it is required but blank
			return errors.New(fieldStruct.Name + " is required, but blank")
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		if field.Kind() == reflect.Struct {
			if err := c.processTags(field.Addr().Interface(), getPrefixForStruct(prefixes, &fieldStruct)...); err != nil {
				return err
			}
		}

		if field.Kind() == reflect.Slice {
			if arrLen := field.Len(); arrLen > 0 {
				for i := 0; i < arrLen; i++ {
					if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
						if err := c.processTags(field.Index(i).Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(i))...); err != nil {
							return err
						}
					}
				}
			} else {
				defer func(field reflect.Value, fieldStruct reflect.StructField) {
					if !configValue.IsZero() {
						// load slice from env
						newVal := reflect.New(field.Type().Elem()).Elem()
						if newVal.Kind() == reflect.Struct {
							idx := 0
							for {
								newVal = reflect.New(field.Type().Elem()).Elem()
								if err := c.processTags(newVal.Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(idx))...); err != nil {
									return // err
								} else if reflect.DeepEqual(newVal.Interface(), reflect.New(field.Type().Elem()).Elem().Interface()) {
									break
								} else {
									idx++
									field.Set(reflect.Append(field, newVal))
								}
							}
						}
					}
				}(field, fieldStruct)
			}
		}
	}
	return nil
}

func (c *Config) loadWithKey(key string, config interface{}, watchMode bool, files ...string) (err error, changed bool) {
	defer func() {
		if c.Option.Debug || c.Option.Verbose {
			if err != nil {
				fmt.Printf("Failed to load configuration from %v, got %v\n", files, err)
			}

			fmt.Printf("Configuration:\n  %#v\n", config)
		}
	}()

	configFiles, configModTimeMap := c.getConfigurationFiles(c.Option, watchMode, files...)

	if watchMode {
		if len(configModTimeMap) == len(c.configModTimes) {
			var changed bool
			for f, t := range configModTimeMap {
				if v, ok := c.configModTimes[f]; !ok || t.After(v) {
					changed = true
				}
			}

			if !changed {
				return nil, false
			}
		}
	}

	// process defaults
	_ = c.processDefaults(config)

	for _, file := range configFiles {
		if c.Option.Debug || c.Option.Verbose {
			fmt.Printf("Loading configurations from file '%v'...\n", file)
		}
		if err = c.processFileWithKey(key, config, file); err != nil {
			return err, true
		}
	}
	c.configModTimes = configModTimeMap

	if prefix := c.getENVPrefix(); prefix == "-" {
		if key != "" {
			err = c.processTags(config, strings.ToUpper(key))
		} else {
			err = c.processTags(config)
		}

	} else {
		if key != "" {
			err = c.processTags(config, prefix, strings.ToUpper(key))
		} else {
			err = c.processTags(config, prefix)
		}

	}

	return err, true
}

func (c *Config) load(config interface{}, watchMode bool, files ...string) (err error, changed bool) {
	return c.loadWithKey("", config, watchMode, files...)
}
