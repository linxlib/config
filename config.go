package config

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"sync"
	"time"
)

type Config struct {
	*Option
	mapKeyStruct   map[string]any
	once           sync.Once
	configModTimes map[string]time.Time
}

type Option struct {
	Environment        string
	ENVPrefix          string
	Debug              bool
	Verbose            bool
	Silent             bool
	AutoReload         bool
	AutoReloadInterval time.Duration
	AutoReloadCallback func(key string, config interface{})
	Files              []string
	// You can use embed.FS or any other fs.FS to load configs from. Default - use "os" package
	FS fs.FS
}

// New initialize a Config
func New(config *Option) *Config {
	if config == nil {
		config = &Option{}
	}

	if os.Getenv("CONFIG_DEBUG_MODE") != "" {
		config.Debug = true
	}

	if os.Getenv("CONFIG_VERBOSE_MODE") != "" {
		config.Verbose = true
	}

	if os.Getenv("CONFIG_SILENT_MODE") != "" {
		config.Silent = true
	}

	if config.AutoReload && config.AutoReloadInterval == 0 {
		config.AutoReloadInterval = time.Second
	}

	return &Config{
		Option:       config,
		mapKeyStruct: make(map[string]any),
	}
}

var testRegexp = regexp.MustCompile("_test|(\\.test$)")

// GetEnvironment get environment
func (c *Config) GetEnvironment() string {
	if c.Environment == "" {
		if env := os.Getenv("CONFIG_ENV"); env != "" {
			return env
		}

		if testRegexp.MatchString(os.Args[0]) {
			return "test"
		}

		return "development"
	}
	return c.Environment
}

func (c *Config) LoadWithKey(key string, config interface{}) (err error) {
	c.mapKeyStruct[key] = config

	defaultValue := reflect.Indirect(reflect.ValueOf(config))
	if !defaultValue.CanAddr() {
		return fmt.Errorf("config %v should be addressable", config)
	}
	err, _ = c.loadWithKey(key, config, false, c.Files...)

	if c.Option.AutoReload {
		c.once.Do(func() {
			go func() {
				timer := time.NewTimer(c.Option.AutoReloadInterval)
				for range timer.C {
					reflectPtr := reflect.New(reflect.ValueOf(config).Elem().Type())
					reflectPtr.Elem().Set(defaultValue)

					var changed bool
					if err, changed = c.loadWithKey(key, reflectPtr.Interface(), true, c.Files...); err == nil && changed {
						reflect.ValueOf(config).Elem().Set(reflectPtr.Elem())
						if c.Option.AutoReloadCallback != nil {
							c.Option.AutoReloadCallback(key, config)
						}
						for key1, config := range c.mapKeyStruct {
							if key1 != key {
								_, _ = c.loadWithKey(key1, config, false, c.Files...)

								if c.Option.AutoReloadCallback != nil {
									c.Option.AutoReloadCallback(key1, config)
								}
							}

						}

					} else if err != nil {
						fmt.Printf("Failed to reload configuration from %v, got error %v\n", c.Files, err)
					}

					timer.Reset(c.Option.AutoReloadInterval)
				}
			}()
		})

	}
	return
}

// Load will unmarshal configurations to struct from files that you provide
func (c *Config) Load(config interface{}) error {
	return c.LoadWithKey("", config)
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(config interface{}, files ...string) error {
	return New(&Option{Files: files}).Load(config)
}

func LoadWithKey(key string, config interface{}, files ...string) error {
	return New(&Option{Files: files}).LoadWithKey(key, config)
}
