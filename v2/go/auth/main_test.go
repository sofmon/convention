package auth_test

import (
	"fmt"
	"testing"

	cfg "github.com/sofmon/convention/v2/go/cfg"
)

func TestMain(m *testing.M) {

	err := cfg.SetConfigLocation("../../../.secret")
	if err != nil {
		err = fmt.Errorf("SetConfigLocation failed: %w", err)
		panic(err)
	}

	m.Run()
}
