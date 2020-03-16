package conf

import (
	"flag"
	"time"
	"github.com/BurntSushi/toml"
)

// Conf config
var (
	confPath string
	Conf     *Redis
)

func init() {
	flag.StringVar(&confPath, "conf", "redis-example.toml", "default config path")
}

// Init init config.
func Init() (err error) {
	_, err = toml.DecodeFile(confPath, &Conf)
	return
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Redis .
type Redis struct {
	Network      string
	Addr         string
	Auth         string
	Active       int
	Idle         int
	DialTimeout  duration
	ReadTimeout  duration
	WriteTimeout duration
	IdleTimeout  duration
	Expire       duration
}
