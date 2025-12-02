package auth_test

import (
	"testing"
	// convCfg "github.com/sofmon/convention/lib/cfg"
)

func TestMain(m *testing.M) {

	// err := convCfg.SetConfigLocation("../../../.secret")
	// if err != nil {
	// 	err = fmt.Errorf("SetConfigLocation failed: %w", err)
	// 	panic(err)
	// }

	m.Run()
}
