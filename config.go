package config

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"time"
)

type Config struct {
	*Option
	files          []string
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

	return &Config{Option: config}
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

func (c *Config) LoadWithKey(key string, config interface{}, files ...string) (err error) {
	if len(files) == 0 {
		if len(c.files) != 0 {
			files = c.files
		}
	}
	defaultValue := reflect.Indirect(reflect.ValueOf(config))
	if !defaultValue.CanAddr() {
		return fmt.Errorf("config %v should be addressable", config)
	}
	err, _ = c.loadWithKey(key, config, false, files...)

	if c.Option.AutoReload {
		go func() {
			timer := time.NewTimer(c.Option.AutoReloadInterval)
			for range timer.C {
				reflectPtr := reflect.New(reflect.ValueOf(config).Elem().Type())
				reflectPtr.Elem().Set(defaultValue)

				var changed bool
				if err, changed = c.loadWithKey(key, reflectPtr.Interface(), true, files...); err == nil && changed {
					reflect.ValueOf(config).Elem().Set(reflectPtr.Elem())
					if c.Option.AutoReloadCallback != nil {
						c.Option.AutoReloadCallback(key, config)
					}
				} else if err != nil {
					fmt.Printf("Failed to reload configuration from %v, got error %v\n", files, err)
				}
				timer.Reset(c.Option.AutoReloadInterval)
			}
		}()
	}
	return
}

// Load will unmarshal configurations to struct from files that you provide
func (c *Config) Load(config interface{}, files ...string) error {
	return c.LoadWithKey("", config, files...)
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(config interface{}, files ...string) error {
	return New(nil).LoadWithKey("", config, files...)
}

func LoadWithKey(key string, config interface{}, files ...string) error {
	return New(nil).LoadWithKey(key, config, files...)
}
