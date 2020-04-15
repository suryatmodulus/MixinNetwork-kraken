package engine

import (
	"io/ioutil"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/pelletier/go-toml"
)

type Configuration struct {
	Engine struct {
		Interface    string `toml:"interface"`
		MaxPeerCount int    `toml:"max-peer-count"`
	} `toml:"engine"`
	RPC struct {
		Port int `toml:"port"`
	} `toml:"rpc"`
}

func Setup(path string) (*Configuration, error) {
	logger.Printf("Setup(%s)\n", path)
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Configuration
	err = toml.Unmarshal(f, &conf)
	return &conf, err
}