package config

import (
	"encoding/hex"
	"errors"
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
	Origin      string              `default:"http://localhost:8204" desc:"origin (url) of the user interface for CORS access"`
	Adyen       AdyenConfig
	processed   bool
}

type AdyenConfig struct {
	APIKey    string `split_words:"true" required:"true" desc:"api key for adyen payments api access"`
	Live      bool   `default:"false" desc:"set to true to enable live payments and access to the live environment"`
	URLPrefix string `split_words:"true" desc:"the live endpoint url prefix used to access the live environment"`
	Webhook   AdyenWebhookConfig
}

type AdyenWebhookConfig struct {
	UseBasicAuth bool   `split_words:"true" default:"false" desc:"verify adyen webhooks with basic authentication"`
	Username     string `default:"" desc:"if basic auth is enabled, provide the configured username"`
	Password     string `default:"" desc:"if basic auth is enabled, provide the configured password in plaintext"`
	VerifyHMAC   bool   `split_words:"true" default:"false" desc:"if true, verify the hmac in the additional details of the webhook"`
	HMACSecret   string `split_words:"true" desc:"specify the configured hmac secret for message verification"`
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

	if err = c.Adyen.Validate(); err != nil {
		return err
	}

	return nil
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

func (c AdyenConfig) Validate() error {
	if c.Live {
		if c.URLPrefix == "" {
			return errors.New("invalid configuration: url prefix is required when in live mode")
		}
	}

	if err := c.Webhook.Validate(); err != nil {
		return err
	}

	return nil
}

func (c AdyenWebhookConfig) Validate() error {
	if c.UseBasicAuth {
		if c.Username == "" || c.Password == "" {
			return errors.New("invalid configuration: username and password required when basic auth is enabled")
		}
	}

	if c.VerifyHMAC {
		if c.HMACSecret == "" {
			return errors.New("invalid configuration: hmac secret is required when verify hmac is enabled")
		}

		if _, err := hex.DecodeString(c.HMACSecret); err != nil {
			return errors.New("invalid configuration:  hmac secret must be a hex encoded string")
		}
	}

	return nil
}
