package config

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/rotationalio/confire"
	"github.com/rotationalio/exchequer/pkg/logger"
)

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be EXCHEQUER_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const Prefix = "exchequer"

// Config contains all of the configuration parameters for the service and is
// loaded from the environment or a configuration file with reasonable defaults for
// values that are omitted. The Config should be validated in preparation for running
// the server to ensure that all server operations work as expected.
type Config struct {
	Maintenance bool                `default:"false" desc:"if true, the node will start in maintenance mode"`
	Mode        string              `default:"release" desc:"specify the mode of the server (release, debug, testing)"`
	LogLevel    logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog  bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	BindAddr    string              `split_words:"true" default:"8204" desc:"the ip address and port to bind the web service on"`
	processed   bool
}

func New() (conf Config, err error) {
	if err = confire.Process(Prefix, &conf); err != nil {
		return Config{}, err
	}

	if err = conf.Validate(); err != nil {
		return Config{}, err
	}

	conf.processed = true
	return conf, nil
}

// Returns true if the config has not been correctly processed from the environment.
func (c Config) IsZero() bool {
	return !c.processed
}

// Custom validations are added here, particularly validations that require one or more
// fields to be processed before the validation occurs.
// NOTE: ensure that all nested config validation methods are called here.
func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		return fmt.Errorf("invalid configuration: %q is not a valid gin mode", c.Mode)
	}

	return nil
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}
