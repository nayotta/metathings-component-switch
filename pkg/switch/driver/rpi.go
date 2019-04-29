package switch_driver

import (
	"errors"
	"fmt"
	"sync"

	opt_helper "github.com/nayotta/metathings/pkg/common/option"
	component "github.com/nayotta/metathings/pkg/component"
	log "github.com/sirupsen/logrus"
	rpio "github.com/stianeikeland/go-rpio"
)

/*

The library use the raw BCM2835 pinouts, not the ports as they are mapped
on the output pins for the raspberry pi, and not the wiringPi convention.

            Rev 2 and 3 Raspberry Pi                        Rev 1 Raspberry Pi (legacy)
  +-----+---------+----------+---------+-----+      +-----+--------+----------+--------+-----+
  | BCM |   Name  | Physical | Name    | BCM |      | BCM | Name   | Physical | Name   | BCM |
  +-----+---------+----++----+---------+-----+      +-----+--------+----++----+--------+-----+
  |     |    3.3v |  1 || 2  | 5v      |     |      |     | 3.3v   |  1 ||  2 | 5v     |     |
  |   2 |   SDA 1 |  3 || 4  | 5v      |     |      |   0 | SDA    |  3 ||  4 | 5v     |     |
  |   3 |   SCL 1 |  5 || 6  | 0v      |     |      |   1 | SCL    |  5 ||  6 | 0v     |     |
  |   4 | GPIO  7 |  7 || 8  | TxD     | 14  |      |   4 | GPIO 7 |  7 ||  8 | TxD    |  14 |
  |     |      0v |  9 || 10 | RxD     | 15  |      |     | 0v     |  9 || 10 | RxD    |  15 |
  |  17 | GPIO  0 | 11 || 12 | GPIO  1 | 18  |      |  17 | GPIO 0 | 11 || 12 | GPIO 1 |  18 |
  |  27 | GPIO  2 | 13 || 14 | 0v      |     |      |  21 | GPIO 2 | 13 || 14 | 0v     |     |
  |  22 | GPIO  3 | 15 || 16 | GPIO  4 | 23  |      |  22 | GPIO 3 | 15 || 16 | GPIO 4 |  23 |
  |     |    3.3v | 17 || 18 | GPIO  5 | 24  |      |     | 3.3v   | 17 || 18 | GPIO 5 |  24 |
  |  10 |    MOSI | 19 || 20 | 0v      |     |      |  10 | MOSI   | 19 || 20 | 0v     |     |
  |   9 |    MISO | 21 || 22 | GPIO  6 | 25  |      |   9 | MISO   | 21 || 22 | GPIO 6 |  25 |
  |  11 |    SCLK | 23 || 24 | CE0     | 8   |      |  11 | SCLK   | 23 || 24 | CE0    |   8 |
  |     |      0v | 25 || 26 | CE1     | 7   |      |     | 0v     | 25 || 26 | CE1    |   7 |
  |   0 |   SDA 0 | 27 || 28 | SCL 0   | 1   |      +-----+--------+----++----+--------+-----+
  |   5 | GPIO 21 | 29 || 30 | 0v      |     |
  |   6 | GPIO 22 | 31 || 32 | GPIO 26 | 12  |
  |  13 | GPIO 23 | 33 || 34 | 0v      |     |
  |  19 | GPIO 24 | 35 || 36 | GPIO 27 | 16  |
  |  26 | GPIO 25 | 37 || 38 | GPIO 28 | 20  |
  |     |      0v | 39 || 40 | GPIO 29 | 21  |
  +-----+---------+----++----+---------+-----+

See the spec for full details of the BCM2835 controller:

https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2835/BCM2835-ARM-Peripherals.pdf
and https://elinux.org/BCM2835_datasheet_errata - for errors in that spec

*/

/*

Driver: rpi
  Raspberry pi gpio switch driver.
Example Config:
  driver:
    name: rpi
    version: pi3
    pin: 11
*/

var (
	rpi_pin_modern_mapper = map[int]rpio.Pin{
		3:  rpio.Pin(2),
		5:  rpio.Pin(3),
		7:  rpio.Pin(4),
		11: rpio.Pin(17),
		13: rpio.Pin(27),
		15: rpio.Pin(22),
		19: rpio.Pin(10),
		21: rpio.Pin(9),
		23: rpio.Pin(11),
		27: rpio.Pin(0),
		29: rpio.Pin(5),
		31: rpio.Pin(6),
		33: rpio.Pin(13),
		35: rpio.Pin(19),
		37: rpio.Pin(26),
		8:  rpio.Pin(14),
		10: rpio.Pin(15),
		12: rpio.Pin(18),
		16: rpio.Pin(23),
		18: rpio.Pin(24),
		22: rpio.Pin(25),
		24: rpio.Pin(8),
		26: rpio.Pin(7),
		28: rpio.Pin(1),
		32: rpio.Pin(12),
		36: rpio.Pin(16),
		38: rpio.Pin(20),
		40: rpio.Pin(21),
	}
	rpi_pin_legacy_mapper = map[int]rpio.Pin{
		3:  rpio.Pin(0),
		5:  rpio.Pin(1),
		7:  rpio.Pin(4),
		11: rpio.Pin(17),
		13: rpio.Pin(21),
		15: rpio.Pin(22),
		19: rpio.Pin(10),
		21: rpio.Pin(9),
		23: rpio.Pin(11),
		8:  rpio.Pin(14),
		10: rpio.Pin(15),
		12: rpio.Pin(18),
		16: rpio.Pin(23),
		18: rpio.Pin(24),
		22: rpio.Pin(25),
		24: rpio.Pin(8),
	}
)

var rpi_pin_modes = map[string]string{
	"pi1":  "legacy",
	"pi2":  "legacy",
	"pi3":  "modern",
	"pi0":  "modern",
	"pi0w": "modern",
}

func new_invalid_config_error(key string) error {
	return errors.New(fmt.Sprintf("invalid config: %v", key))
}

func rpi_pin(version string, pin int) (rpio.Pin, error) {
	m := make(map[int]rpio.Pin)

	mode, ok := rpi_pin_modes[version]
	if !ok {
		return rpio.Pin(255), new_invalid_config_error("version")
	}

	switch mode {
	case "legacy":
		m = rpi_pin_legacy_mapper
	case "modern":
		m = rpi_pin_modern_mapper
	}
	rpin, ok := m[pin]
	if !ok {
		return rpio.Pin(255), new_invalid_config_error("pin")
	}

	return rpin, nil
}

type RpiSwitchDriver struct {
	op_mtx *sync.Mutex
	pin    rpio.Pin

	logger log.FieldLogger
	mdl    *component.Module
	opt    *SwitchDriverOption
	st     *SwitchDriverState
}

func (d *RpiSwitchDriver) Start() error {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	d.pin.High()
	d.st = SWITCH_DRIVER_STATE_ON

	return nil
}

func (d *RpiSwitchDriver) Stop() error {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	d.pin.Low()
	d.st = SWITCH_DRIVER_STATE_OFF

	return nil
}

func (d *RpiSwitchDriver) State() *SwitchDriverState {
	d.op_mtx.Lock()
	defer d.op_mtx.Unlock()

	return d.st
}

func NewRpiSwitchDriver(opt *SwitchDriverOption, args ...interface{}) (SwitchDriver, error) {
	var ok bool
	var logger log.FieldLogger
	var module *component.Module

	opt_helper.Setopt(map[string]func(key string, val interface{}) error{
		"logger": func(key string, val interface{}) error {
			if logger, ok = val.(log.FieldLogger); !ok {
				return opt_helper.ErrInvalidArguments
			}
			return nil
		},
		"module": func(key string, val interface{}) error {
			if module, ok = val.(*component.Module); !ok {
				return opt_helper.ErrInvalidArguments
			}
			return nil
		},
	})(args...)

	ver := opt.GetString("version")
	pin := opt.GetInt("pin")

	err := rpio.Open()
	if err != nil {
		return nil, err
	}

	rpin, err := rpi_pin(ver, pin)
	if err != nil {
		return nil, err
	}
	rpin.Output()
	rpin.Low()

	drv := &RpiSwitchDriver{
		op_mtx: new(sync.Mutex),
		pin:    rpin,
		logger: logger,
		mdl:    module,
		opt:    opt,
		st:     SWITCH_DRIVER_STATE_OFF,
	}

	return drv, nil
}

var register_rpi_switch_driver_once sync.Once

func init() {
	register_rpi_switch_driver_once.Do(func() {
		register_switch_driver_factory("rpi", NewRpiSwitchDriver)
	})
}
