package switch_driver

import (
	"sync"

	"github.com/spf13/viper"
)

type SwitchDriverOption struct {
	*viper.Viper
}

func (o *SwitchDriverOption) Sub(key string) *SwitchDriverOption {
	sub := o.Viper.Sub(key)
	if sub == nil {
		return nil
	}

	return &SwitchDriverOption{sub}
}

type SwitchDriverState struct {
	state string
}

func (s *SwitchDriverState) String() string {
	return s.state
}

var (
	SWITCH_DRIVER_STATE_ON  = &SwitchDriverState{state: "on"}
	SWITCH_DRIVER_STATE_OFF = &SwitchDriverState{state: "off"}
)

type SwitchDriver interface {
	On() error
	Off() error
	State() *SwitchDriverState
}

type SwitchDriverFactory func(opt *SwitchDriverOption, args ...interface{}) (SwitchDriver, error)

var switch_driver_factories map[string]SwitchDriverFactory
var switch_driver_factories_once sync.Once

func register_switch_driver_factory(name string, fty SwitchDriverFactory) {
	switch_driver_factories_once.Do(func() {
		switch_driver_factories = make(map[string]SwitchDriverFactory)
	})

	switch_driver_factories[name] = fty
}

func NewSwitchDriver(name string, opt *SwitchDriverOption, args ...interface{}) (SwitchDriver, error) {
	fty, ok := switch_driver_factories[name]
	if !ok {
		return nil, ErrInvalidSwitchDriver
	}

	return fty(opt, args...)
}
