package vipero

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-toho/toho/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const (
	aliasTagName = "alias"
	envTagName   = "env"
	jsonTagName  = "json"
)

var envKeyReplacer = strings.NewReplacer(".", "_")

func NewDefault() *viper.Viper {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(envKeyReplacer)

	return v
}

func New(appName string) *viper.Viper {
	v := NewDefault()

	if appName != "" {
		v.SetDefault(app.NamedAppName, appName)
		v.SetEnvPrefix(strings.ToUpper(appName))
	}

	return v
}

// LoadConfig loads configuration into 'cfg' variable.
func LoadConfig(v *viper.Viper, cfg any, configFiles []string, opts ...viper.DecoderConfigOption) error {
	if err := SetDefaults(cfg); err != nil {
		return fmt.Errorf("could not set defaults: %w", err)
	}

	for _, file := range configFiles {
		// set the config file
		viper.SetConfigFile(file)
		// read the config file and merge it into the viper instance
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// config file not found; ignore error
			} else {
				return fmt.Errorf("unable to read config file %s: %w", file, err)
			}
		}
	}

	if err := structBindEnv(cfg, v); err != nil {
		return err
	}

	// include default decoder configuration
	opts = append([]viper.DecoderConfigOption{
		func(dc *mapstructure.DecoderConfig) {
			dc.Squash = true
			dc.TagName = jsonTagName
		},
	}, opts...)

	if err := v.Unmarshal(cfg, opts...); err != nil {
		return err
	}

	return nil
}

// structBindEnv binds environment variables into the Viper instance for
// the given struct.
// It gets the pointer of a struct that is going to holds the variables.
func structBindEnv(structure any, v *viper.Viper, prefix ...string) error {
	inputType := reflect.TypeOf(structure)
	if inputType != nil {
		if inputType.Kind() == reflect.Ptr {
			if inputType.Elem().Kind() == reflect.Struct {
				return bindStruct(reflect.ValueOf(structure).Elem(), v, prefix)
			}
		}
	}

	return errors.New("config: invalid structure")
}

// bindStruct binds a reflected struct fields path to Viper instance.
func bindStruct(s reflect.Value, v *viper.Viper, prefix []string) error {
	for i := 0; i < s.NumField(); i++ {
		fieldName := s.Type().Field(i).Name

		if t, exist := s.Type().Field(i).Tag.Lookup(jsonTagName); exist {
			fieldName = t
		}

		key := strings.Join(append(prefix, fieldName), ".")

		if t, exist := s.Type().Field(i).Tag.Lookup(aliasTagName); exist {
			v.RegisterAlias(key, t)
		}

		if t, exist := s.Type().Field(i).Tag.Lookup(envTagName); exist {
			v.BindEnv(key, t)
		} else if s.Type().Field(i).Type.Kind() == reflect.Struct {
			if s.Type().Field(i).Anonymous {
				// squash the field down if anonymous
				if err := bindStruct(s.Field(i), v, prefix); err != nil {
					return err
				}
			} else {
				if err := bindStruct(s.Field(i), v, append(prefix, fieldName)); err != nil {
					return err
				}
			}
		} else if s.Type().Field(i).Type.Kind() == reflect.Ptr {
			if !s.Field(i).IsZero() && s.Field(i).Elem().Type().Kind() == reflect.Struct {
				if err := bindStruct(s.Field(i).Elem(), v, prefix); err != nil {
					return err
				}
			}
		} else {
			v.BindEnv(key, strings.ToUpper(envKeyReplacer.Replace(key)))
		}
	}

	return nil
}
