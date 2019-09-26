package switch_service

import (
	"context"
	"io"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	driver "github.com/nayotta/metathings-component-switch/pkg/switch/driver"
	component "github.com/nayotta/metathings/pkg/component"
)

type SwitchService struct {
	module *component.Module
	driver driver.SwitchDriver
}

func (ss *SwitchService) logger() log.FieldLogger {
	return ss.module.Logger()
}

func (ss *SwitchService) set_state(state string) error {
	err := ss.module.PutObjects(map[string]io.Reader{
		"state": strings.NewReader(state),
	})
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to set switch state")
		return err
	}
	return nil
}

func (ss *SwitchService) reset() {
	ss.driver.Stop()
	ss.set_state("off")
}

func (ss *SwitchService) HANDLE_GRPC_Start(ctx context.Context, in *any.Any) (*any.Any, error) {
	var err error
	req := &empty.Empty{}

	if err = ptypes.UnmarshalAny(in, req); err != nil {
		return nil, err
	}

	res, err := ss.Start(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ss *SwitchService) Start(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	err := ss.driver.Start()
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to start switch")
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err = ss.set_state("on"); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	ss.logger().Infof("switch startd")

	return &empty.Empty{}, nil
}

func (ss *SwitchService) HANDLE_GRPC_Stop(ctx context.Context, in *any.Any) (*any.Any, error) {
	var err error
	req := &empty.Empty{}

	if err = ptypes.UnmarshalAny(in, req); err != nil {
		return nil, err
	}

	res, err := ss.Stop(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ss *SwitchService) Stop(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	err := ss.driver.Stop()
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to stop switch")
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err = ss.set_state("off"); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	ss.logger().Infof("switch stop")

	return &empty.Empty{}, nil
}

func (ss *SwitchService) InitModuleService(m *component.Module) error {
	var err error

	ss.module = m

	drv_opt := &driver.SwitchDriverOption{ss.module.Kernel().Config().Sub("driver").Raw()}
	ss.driver, err = driver.NewSwitchDriver(drv_opt.GetString("name"), drv_opt, "logger", ss.logger(), "module", ss.module)
	if err != nil {
		return err
	}
	ss.logger().WithField("driver", drv_opt.GetString("name")).Debugf("init switch driver")

	ss.reset()

	return nil
}
