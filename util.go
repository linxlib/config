package config

import (
	"github.com/gookit/goutil/envutil"
	"github.com/mitchellh/mapstructure"
	"os"
	"reflect"
	"strings"
	"time"
)

// ValDecodeHookFunc returns a mapstructure.DecodeHookFunc
// that parse ENV var, and more custom parse
func ValDecodeHookFunc(parseEnv, parseTime bool) mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		var err error
		str := data.(string)
		if parseEnv {
			// https://docs.docker.com/compose/environment-variables/env-file/
			str, err = envutil.ParseOrErr(str)
			if err != nil {
				return nil, err
			}
		}
		if len(str) < 2 {
			return str, nil
		}

		// start char is number(1-9)
		if str[0] > '0' && str[0] <= '9' {
			// parse time string. eg: 10s
			if parseTime && t.Kind() == reflect.Int64 {
				dur, err := time.ParseDuration(str)
				if err == nil {
					return dur, nil
				}
			}
		}
		return str, nil
	}
}

// resolve format, check is alias
func (c *Config) resolveFormat(f string) string {
	if name, ok := c.aliasMap[f]; ok {
		return name
	}
	return f
}

/*************************************************************
 * helper methods/functions
 *************************************************************/

// LoadENVFiles load
// func LoadENVFiles(filePaths ...string) error {
// 	return dotenv.LoadFiles(filePaths...)
// }

// GetEnv get os ENV value by name
func GetEnv(name string, defVal ...string) (val string) {
	return Getenv(name, defVal...)
}

// Getenv get os ENV value by name. like os.Getenv, but support default value
//
// Notice:
// - Key is not case-sensitive when getting
func Getenv(name string, defVal ...string) (val string) {
	if val = os.Getenv(name); val != "" {
		return
	}

	if len(defVal) > 0 {
		val = defVal[0]
	}
	return
}

func parseVarNameAndType(key string) (string, string, string) {
	typ := "string"
	key = strings.Trim(key, "-")
	var desc string
	// can set var type: int, uint, bool
	if strings.IndexByte(key, ':') > 0 {
		list := strings.SplitN(key, ":", 3)
		key, typ = list[0], list[1]
		if len(list) == 3 {
			desc = list[2]
		}

		if _, ok := validTypes[typ]; !ok {
			typ = "string"
		}
	}
	return key, typ, desc
}

// format key
func formatKey(key, sep string) string {
	return strings.Trim(strings.TrimSpace(key), sep)
}
