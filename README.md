# Config

Golang Configuration tool that support YAML, JSON, Shell Environment

is same as [configor](https://github.com/jinzhu/configor) , but add LoadWithKey support and remove toml

## Usage

config.example.yaml in `./config` directory

```yaml
server:
  host: 10.10.0.178
  port: 8585

redis:
  host: 10.1.0.16
  port: 6379
  db: 5
```

```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/linxlib/config"
	"os"
)

type ServerConfig struct {
	Host string `yaml:"host" default:"0.0.0.0" env:"CONFIG_HOST"`
	Port string `yaml:"port" default:"8080"`
}

type RedisConfig struct {
	Host string `yaml:"host" default:"0.0.0.0"`
	Port string `yaml:"port" default:"8080"`
	DB   int    `yaml:"db" default:"0"`
}

func main() {
	//os.Setenv("CONFIG_HOST", "1.1.1.1")
	os.Setenv("CONFIG_REDIS_HOST", "1.2.1.1")
	var sc = new(ServerConfig)
	var rc = new(RedisConfig)
	err := config.LoadWithKey("server", sc, "config/config.yaml")
	if err != nil {
		return
	}
	err = config.LoadWithKey("redis", rc, "config/config.yaml")
	if err != nil {
		return
	}
	var sc1 = new(ServerConfig)
	err = config.LoadWithKey("", sc1, "config/config.yaml")
	if err != nil {
		return
	}
	bs, _ := json.MarshalIndent(sc, "", "  ")
	fmt.Println(string(bs))
	bs, _ = json.MarshalIndent(rc, "", "  ")
	fmt.Println(string(bs))
	bs, _ = json.MarshalIndent(sc1, "", "  ")
	fmt.Println(string(bs))

}
```

With configuration file *config.yml*:

```yaml
appname: test

db:
    name:     test
    user:     test
    password: test
    port:     1234

contacts:
- name: i test
  email: test@test.com
```

## Debug Mode & Verbose Mode

Debug/Verbose mode is helpful when debuging your application, `debug mode` will let you know how `config` loaded your configurations, like from which file, shell env, `verbose mode` will tell you even more, like those shell environments `config` tried to load.

```go
// Enable debug mode or set env `CONFIG_DEBUG_MODE` to true when running your application
config.New(&config.Config{Debug: true}).Load(&Config, "config.json")

// Enable verbose mode or set env `CONFIG_VERBOSE_MODE` to true when running your application
config.New(&config.Config{Verbose: true}).Load(&Config, "config.json")
```

## Auto Reload Mode

Config can auto reload configuration based on time

```go
// auto reload configuration every second
config.New(&config.Config{AutoReload: true}).Load(&Config, "config.json")

// auto reload configuration every minute
config.New(&config.Config{AutoReload: true, AutoReloadInterval: time.Minute}).Load(&Config, "config.json")
```

Auto Reload Callback

```go
config.New(&config.Config{AutoReload: true, AutoReloadCallback: func(config interface{}) {
    fmt.Printf("%v changed", config)
}}).Load(&Config, "config.json")
```

# Advanced Usage

* Load multiple configurations

```go
// Earlier configurations have higher priority
config.Load(&Config, "application.yml", "database.json")
```

* Load configuration by environment

Use `CONFIG_ENV` to set environment, if `CONFIG_ENV` not set, environment will be `development` by default, and it will be `test` when running tests with `go test`

```go
// config.go
config.Load(&Config, "config.json")

$ go run config.go
// Will load `config.json`, `config.development.json` if it exists
// `config.development.json` will overwrite `config.json`'s configuration
// You could use this to share same configuration across different environments

$ CONFIG_ENV=production go run config.go
// Will load `config.json`, `config.production.json` if it exists
// `config.production.json` will overwrite `config.json`'s configuration

$ go test
// Will load `config.json`, `config.test.json` if it exists
// `config.test.json` will overwrite `config.json`'s configuration

$ CONFIG_ENV=production go test
// Will load `config.json`, `config.production.json` if it exists
// `config.production.json` will overwrite `config.json`'s configuration
```

```go
// Set environment by config
config.New(&config.Config{Environment: "production"}).Load(&Config, "config.json")
```

* Example Configuration

```go
// config.go
config.Load(&Config, "config.yml")

$ go run config.go
// Will load `config.example.yml` automatically if `config.yml` not found and print warning message
```

* Load From Shell Environment

```go
$ CONFIG_APPNAME="hello world" CONFIG_DB_NAME="hello world" go run config.go
// Load configuration from shell environment, it's name is {{prefix}}_FieldName
```

```go
// You could overwrite the prefix with environment CONFIG_ENV_PREFIX, for example:
$ CONFIG_ENV_PREFIX="WEB" WEB_APPNAME="hello world" WEB_DB_NAME="hello world" go run config.go

// Set prefix by config
config.New(&config.Config{ENVPrefix: "WEB"}).Load(&Config, "config.json")
```

* Anonymous Struct

Add the `anonymous:"true"` tag to an anonymous, embedded struct to NOT include the struct name in the environment
variable of any contained fields.  For example:

```go
type Details struct {
	Description string
}

type Config struct {
	Details `anonymous:"true"`
}
```

With the `anonymous:"true"` tag specified, the environment variable for the `Description` field is `CONFIG_DESCRIPTION`.
Without the `anonymous:"true"`tag specified, then environment variable would include the embedded struct name and be `CONFIG_DETAILS_DESCRIPTION`.

* With flags

```go
func main() {
	conf := flag.String("file", "config.yml", "configuration file")
	flag.StringVar(&Config.APPName, "name", "", "app name")
	flag.StringVar(&Config.DB.Name, "db-name", "", "database name")
	flag.StringVar(&Config.DB.User, "db-user", "root", "database user")
	flag.Parse()

	os.Setenv("CONFIG_ENV_PREFIX", "-")
	config.Load(&Config, *conf)
	// config.Load(&Config) // only load configurations from shell env & flag
}
```



## License

Released under the MIT License