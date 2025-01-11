package config

import (
	"os"
	"sync"

	"github.com/goravel/framework/support/file"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config interface {
	// Env get configs from env.
	Env(envName string, defaultValue ...any) any
	// Add configs to application.
	Add(name string, configuration any)
	// Get configs from application.
	Get(path string, defaultValue ...any) any
	// GetString get string type configs from application.
	GetString(path string, defaultValue ...string) string
	// GetInt get int type configs from application.
	GetInt(path string, defaultValue ...int) int
	// GetBool get bool type configs from application.
	GetBool(path string, defaultValue ...bool) bool
}

type Impl struct {
	vip *viper.Viper
}

var (
	instance Config
	once     sync.Once
)

// InitEnv initializes and returns the configuration singleton.
func InitEnv(envPath string) {
	if instance != nil {
		log.Info().Msg("Env already initialized!")
		return
	}

	once.Do(func() {
		vip := viper.New()
		vip.AutomaticEnv()

		if file.Exists(envPath) {
			vip.SetConfigType("env")
			vip.SetConfigFile(envPath)

			if err := vip.ReadInConfig(); err != nil {
				log.Panic().Msgf("Invalid ConfigImpl error: " + err.Error())
				os.Exit(1)
			}
		}

		instance = &Impl{vip: vip}
	})
}

// GetInstance returns the initialized ConfigImpl instance.
// Panics if InitEnv has not been called.
func GetInstance() Config {
	if instance == nil {
		log.Fatal().Msg("configs not loaded. Call InitEnv(envPath) first.")
	}
	return instance
}

// Env retrieves a configuration value or a default value if not set.
func (c *Impl) Env(envName string, defaultValue ...any) any {
	return c.Get(envName, defaultValue...)
}

// Add adds a new configuration value.
func (c *Impl) Add(name string, value any) {
	c.vip.Set(name, value)
}

// Get retrieves a configuration value or a default value if not set.
func (c *Impl) Get(path string, defaultValue ...any) any {
	if !c.vip.IsSet(path) && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return c.vip.Get(path)
}

// GetString retrieves a string configuration value or a default value if not set.
func (c *Impl) GetString(path string, defaultValue ...string) string {
	if !c.vip.IsSet(path) && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return c.vip.GetString(path)
}

// GetInt retrieves an int configuration value or a default value if not set.
func (c *Impl) GetInt(path string, defaultValue ...int) int {
	if !c.vip.IsSet(path) && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return c.vip.GetInt(path)
}

// GetBool retrieves a bool configuration value or a default value if not set.
func (c *Impl) GetBool(path string, defaultValue ...bool) bool {
	if !c.vip.IsSet(path) && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return c.vip.GetBool(path)
}
