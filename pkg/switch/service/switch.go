package switch_service

import (
	"context"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	driver "github.com/nayotta/metathings-component-switch/pkg/switch/driver"
	pb "github.com/nayotta/metathings-protocol-switch/go/proto"
	component "github.com/nayotta/metathings/pkg/component"
)

type SwitchService struct {
	module *component.Module
	driver driver.SwitchDriver
}

func (ss *SwitchService) logger() log.FieldLogger {
	return ss.module.Logger()
}

func (ss *SwitchService) update_state() error {
	err := ss.module.PutObject("state", strings.NewReader(ss.driver.State().String()))
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to set switch state")
		return err
	}

	return nil
}

func (ss *SwitchService) reset() {
	ss.driver.Off()
	ss.update_state()
}

func (ss *SwitchService) HANDLE_GRPC_On(ctx context.Context, in *any.Any) (*any.Any, error) {
	var err error
	req := &empty.Empty{}

	if err = ptypes.UnmarshalAny(in, req); err != nil {
		return nil, err
	}

	res, err := ss.On(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ss *SwitchService) On(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	err := ss.driver.On()
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to turn switch on")
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err = ss.update_state(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	ss.logger().Infof("switch on")

	return &empty.Empty{}, nil
}

func (ss *SwitchService) HANDLE_GRPC_Off(ctx context.Context, in *any.Any) (*any.Any, error) {
	var err error
	req := &empty.Empty{}

	if err = ptypes.UnmarshalAny(in, req); err != nil {
		return nil, err
	}

	res, err := ss.Off(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ss *SwitchService) Off(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	err := ss.driver.Off()
	if err != nil {
		ss.logger().WithError(err).Errorf("failed to turn switch off")
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err = ss.update_state(); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	ss.logger().Infof("switch off")

	return &empty.Empty{}, nil
}

func (ss *SwitchService) HANDLE_GRPC_State(ctx context.Context, in *any.Any) (*any.Any, error) {
	var err error
	req := &empty.Empty{}

	if err = ptypes.UnmarshalAny(in, req); err != nil {
		return nil, err
	}

	res, err := ss.State(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := ptypes.MarshalAny(res)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (ss *SwitchService) State(ctx context.Context, _ *empty.Empty) (*pb.StateResponse, error) {
	state := ss.driver.State().String()

	ss.logger().WithField("state", state).Debugf("get state")

	return &pb.StateResponse{State: state}, nil
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
