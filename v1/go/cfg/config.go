package cfg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ConfigKey string

const (
	ConfigKeyAppConfig = "app-config"
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

func FilePath(key ConfigKey) string {
	return configLocation + string(key)
}

func Bytes(key ConfigKey) (value []byte, err error) {
	file := configLocation + string(key)
	value, err = os.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("error reading config file '%s': %w", file, err)
	}
	return
}

func BytesOrPanic(key ConfigKey) (res []byte) {
	res, err := Bytes(key)
	if err != nil {
		panic(err)
	}
	return
}

func String(key ConfigKey) (value string, err error) {
	raw, err := Bytes(key)
	if err != nil {
		return "", err
	}
	value = string(raw)
	return
}

func StringOrPanic(key ConfigKey) (res string) {
	res, err := String(key)
	if err != nil {
		panic(err)
	}
	return
}

func Object[T any](key ConfigKey) (res T, err error) {
	bytes, err := Bytes(key)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return
	}
	return
}

func ObjectOrPanic[T any](key ConfigKey) (res T) {
	bytes, err := Bytes(key)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		panic(err)
	}
	return
}
