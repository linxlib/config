package main

import (
	"fmt"
	"github.com/linxlib/config"
	"time"
)

type MyTestConfig struct {
	A string `yaml:"a" default:"1" env:"TEST_A"`
	B int    `yaml:"b" default:"2" env:"TEST_B"`
}

func main() {
	conf := config.New(&config.Option{
		ENVPrefix:          "FW",
		Silent:             true,
		Debug:              false,
		AutoReload:         true,
		AutoReloadInterval: time.Second * 1,
		AutoReloadCallback: func(key string, config1 interface{}) {
			fmt.Println("key=", key, "config=", config1)
		},
		Files: []string{"config.yaml"},
	})
	var mainConfig MyTestConfig
	conf.LoadWithKey("main", &mainConfig)
	var secondConfig MyTestConfig
	conf.LoadWithKey("main", &secondConfig)
	var thirdConfig MyTestConfig
	conf.LoadWithKey("main2", &thirdConfig)

	timer := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-timer.C:
			fmt.Printf("mainconfig: %+v\n", mainConfig)

			fmt.Printf("secondConfig: %+v\n", secondConfig)
			fmt.Printf("thirdConfig: %+v\n", thirdConfig)
		}
	}
}
