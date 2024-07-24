package config_test

import (
	"os"
	"testing"

	"github.com/rotationalio/exchequer/pkg/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var testEnv = map[string]string{
	"EXCHEQUER_MAINTENANCE":                  "true",
	"EXCHEQUER_MODE":                         "test",
	"EXCHEQUER_LOG_LEVEL":                    "debug",
	"EXCHEQUER_CONSOLE_LOG":                  "true",
	"EXCHEQUER_BIND_ADDR":                    ":9000",
	"EXCHEQUER_ORIGIN":                       "http://localhost:9000",
	"EXCHEQUER_ADYEN_API_KEY":                "my api key",
	"EXCHEQUER_ADYEN_LIVE":                   "true",
	"EXCHEQUER_ADYEN_URL_PREFIX":             "1797a841fbb37ca7-AdyenDemo",
	"EXCHEQUER_ADYEN_WEBHOOK_USE_BASIC_AUTH": "true",
	"EXCHEQUER_ADYEN_WEBHOOK_USERNAME":       "admin",
	"EXCHEQUER_ADYEN_WEBHOOK_PASSWORD":       "supersecretpassword",
	"EXCHEQUER_ADYEN_WEBHOOK_VERIFY_HMAC":    "true",
	"EXCHEQUER_ADYEN_WEBHOOK_HMAC_SECRET":    "44782DEF547AAA06C910C43932B1EB0C71FC68D9D0C057550C48EC2ACF6BA056",
}

func TestConfig(t *testing.T) {
	// Set required environment variables and cleanup after the test is complete.
	t.Cleanup(cleanupEnv())
	setEnv()

	conf, err := config.New()
	require.NoError(t, err, "could not process configuration from the environment")
	require.False(t, conf.IsZero(), "processed config should not be zero valued")

	// Ensure configuration is correctly set from the environment
	require.True(t, conf.Maintenance)
	require.Equal(t, testEnv["EXCHEQUER_MODE"], conf.Mode)
	require.Equal(t, zerolog.DebugLevel, conf.GetLogLevel())
	require.True(t, conf.ConsoleLog)
	require.Equal(t, testEnv["EXCHEQUER_BIND_ADDR"], conf.BindAddr)
	require.Equal(t, testEnv["EXCHEQUER_ORIGIN"], conf.Origin)
	require.Equal(t, testEnv["EXCHEQUER_ADYEN_API_KEY"], conf.Adyen.APIKey)
	require.True(t, conf.Adyen.Live)
	require.Equal(t, testEnv["EXCHEQUER_ADYEN_URL_PREFIX"], conf.Adyen.URLPrefix)
	require.True(t, conf.Adyen.Webhook.UseBasicAuth)
	require.Equal(t, testEnv["EXCHEQUER_ADYEN_WEBHOOK_USERNAME"], conf.Adyen.Webhook.Username)
	require.Equal(t, testEnv["EXCHEQUER_ADYEN_WEBHOOK_PASSWORD"], conf.Adyen.Webhook.Password)
	require.True(t, conf.Adyen.Webhook.VerifyHMAC)
	require.Equal(t, testEnv["EXCHEQUER_ADYEN_WEBHOOK_HMAC_SECRET"], conf.Adyen.Webhook.HMACSecret)
}

// Returns the current environment for the specified keys, or if no keys are specified
// then it returns the current environment for all keys in the testEnv variable.
func curEnv(keys ...string) map[string]string {
	env := make(map[string]string)
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	} else {
		for key := range testEnv {
			env[key] = os.Getenv(key)
		}
	}

	return env
}

// Sets the environment variables from the testEnv variable. If no keys are specified,
// then this function sets all environment variables from the testEnv.
func setEnv(keys ...string) {
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := testEnv[key]; ok {
				os.Setenv(key, val)
			}
		}
	} else {
		for key, val := range testEnv {
			os.Setenv(key, val)
		}
	}
}

// Cleanup helper function that can be run when the tests are complete to reset the
// environment back to its previous state before the test was run.
func cleanupEnv(keys ...string) func() {
	prevEnv := curEnv(keys...)
	return func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
