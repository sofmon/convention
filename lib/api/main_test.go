package api_test

import (
	"fmt"
	"sync"
	"testing"

	convCfg "github.com/sofmon/convention/lib/cfg"
)

var (
	apiPorts = map[string]int{}
	mutex    sync.Mutex
)

func portForAPITest(t *testing.T) int {
	mutex.Lock()
	defer mutex.Unlock()
	port, ok := apiPorts[t.Name()]
	if !ok {
		port = 12345 + len(apiPorts)
		apiPorts[t.Name()] = port
	}
	return port
}

func TestMain(m *testing.M) {

	err := convCfg.SetConfigLocation("../../.secret")
	if err != nil {
		err = fmt.Errorf("SetConfigLocation failed: %w", err)
		panic(err)
	}

	m.Run()
}
