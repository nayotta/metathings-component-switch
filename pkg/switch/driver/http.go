package switch_driver

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	opt_helper "github.com/nayotta/metathings/pkg/common/option"
	component "github.com/nayotta/metathings/pkg/component"
	log "github.com/sirupsen/logrus"
)

type HttpSwitchDriver struct {
	op_mtx *sync.Mutex

	logger log.FieldLogger
	mdl    *component.Module
	opt    *SwitchDriverOption
}

func (d *HttpSwitchDriver) request(in map[string]interface{}) (map[string]interface{}, error) {
	target := d.opt.GetString("target")
	buf, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(target, "application/json", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatusCode
	}
	buf, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	err = json.Unmarshal(buf, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

/*
 * send turn on action to target url
 * request.body: {"action": "on"}
 */
func (d *HttpSwitchDriver) On() error {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	if d.state() == SWITCH_DRIVER_STATE_ON {
		return ErrNotTurnable
	}

	_, err := d.request(map[string]interface{}{"action": "on"})
	if err != nil {
		return err
	}

	return nil
}

/*
 * send turn off action to target url
 * request.body: {"action": "off"}
 */
func (d *HttpSwitchDriver) Off() error {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	if d.state() == SWITCH_DRIVER_STATE_OFF {
		return ErrNotTurnable
	}

	_, err := d.request(map[string]interface{}{"action": "off"})
	if err != nil {
		return err
	}

	return nil
}

func (d *HttpSwitchDriver) state() *SwitchDriverState {
	out, err := d.request(map[string]interface{}{"action": "get_state"})
	if err != nil {
		return SWITCH_DRIVER_STATE_OFF
	}

	p, ok := out["state"]
	if !ok {
		return SWITCH_DRIVER_STATE_OFF
	}

	s, ok := p.(string)
	if !ok {
		return SWITCH_DRIVER_STATE_OFF
	}

	st := SWITCH_DRIVER_STATE_OFF
	switch s {
	case "on":
		st = SWITCH_DRIVER_STATE_ON
	case "off":
		st = SWITCH_DRIVER_STATE_OFF
	default:
		st = SWITCH_DRIVER_STATE_OFF
	}

	return st
}

/*
 * send get_state action to target url, response state in "on" or "off"
 * request.body: {"action": "get_state"}
 * response.body: {"state": <state>}
 */
func (d *HttpSwitchDriver) State() *SwitchDriverState {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	return d.state()
}

func NewHttpSwitchDriver(opt *SwitchDriverOption, args ...interface{}) (SwitchDriver, error) {
	var logger log.FieldLogger
	var module *component.Module

	opt_helper.Setopt(map[string]func(key string, val interface{}) error{
		"logger": opt_helper.ToLogger(&logger),
		"module": component.ToModule(&module),
	})

	drv := &HttpSwitchDriver{
		op_mtx: new(sync.Mutex),
		logger: logger,
		mdl:    module,
		opt:    opt,
	}

	return drv, nil
}

var register_http_switch_driver_once sync.Once

func init() {
	register_http_switch_driver_once.Do(func() {
		register_switch_driver_factory("http", NewHttpSwitchDriver)
	})
}
