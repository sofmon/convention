package convention

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ConfigKey string

const (
	ConfigKeyAppConfig ConfigKey = "app-config"

	configKeyCommSecret ConfigKey = "communication_secret"
	configKeyDatabase   ConfigKey = "database"
)

var configLocation = "/etc/app/"

func SetConfigLocation(folder string) error {
	fi, err := os.Stat(folder)
	if os.IsNotExist(err) {
		return fmt.Errorf("folder '%s' does not exists", folder)
	}
	if err != nil {
		return fmt.Errorf("error reading folder '%s': %w", folder, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("config location '%s' must be a folder", folder)
	}
	configLocation = folder
	if !strings.HasSuffix(configLocation, "/") {
		configLocation += "/"
	}
	return nil
}

func ConfigFilePath(key ConfigKey) string {
	return configLocation + string(key)
}

func ConfigBytes(key ConfigKey) (value []byte, err error) {
	file := configLocation + string(key)
	value, err = os.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("error reading config file '%s': %w", file, err)
	}
	return
}

func ConfigBytesOrPanic(key ConfigKey) (res []byte) {
	res, err := ConfigBytes(key)
	if err != nil {
		panic(err)
	}
	return
}

func ConfigString(key ConfigKey) (value string, err error) {
	raw, err := ConfigBytes(key)
	if err != nil {
		return "", err
	}
	value = string(raw)
	return
}

func ConfigStringOrPanic(key ConfigKey) (res string) {
	res, err := ConfigString(key)
	if err != nil {
		panic(err)
	}
	return
}

func ConfigObject[T any](key ConfigKey) (res T, err error) {
	bytes, err := ConfigBytes(key)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return
	}
	return
}

func ConfigObjectOrPanic[T any](key ConfigKey) (res T) {
	bytes, err := ConfigBytes(key)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		panic(err)
	}
	return
}
