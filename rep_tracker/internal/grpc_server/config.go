package grpc_server

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type GrpcServerConfig struct {
	Addr                    string
	Transport               string
	ConcurrentStreamsNumber int
	MaxRcvSize              int
	MaxSendSize             int
	EnableHealthService     bool
	EnableReflection        bool
	MaxConnectionIdle       time.Duration
	MaxConnectionAge        time.Duration
	MaxConnectionAgeGrace   time.Duration
	KeepAliveTime           time.Duration
	KeepAliveTimeout        time.Duration
	KeepAliveMinTime        time.Duration
	KeepAliveWithoutStream  bool
	GracefulStopTimeout     time.Duration
}

func ConfigureGrpcServerAndServer(ctx context.Context, cfg *GrpcServerConfig, registerFunc func(s grpc.ServiceRegistrar)) error {
	listener, err := net.Listen(cfg.Transport, cfg.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	grpcServer := grpc.NewServer(cfg.buildServerOptions()...)
	defer grpcServer.Stop()

	if cfg.EnableHealthService {
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)
	}
	if cfg.EnableReflection {
		reflection.Register(grpcServer)
	}
	if registerFunc != nil {
		registerFunc(grpcServer)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		if cfg.GracefulStopTimeout > 0 {
			done := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(cfg.GracefulStopTimeout):
				grpcServer.Stop()
			}
		} else {
			grpcServer.GracefulStop()
		}
		return nil
	case err = <-errCh:
		return err
	}
}

func (cfg GrpcServerConfig) buildServerOptions() []grpc.ServerOption {
	opts := make([]grpc.ServerOption, 0, 8)
	if cfg.ConcurrentStreamsNumber > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(uint32(cfg.ConcurrentStreamsNumber)))
	}
	if cfg.MaxRcvSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(cfg.MaxRcvSize))
	}
	if cfg.MaxSendSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(cfg.MaxSendSize))
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     cfg.MaxConnectionIdle,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
		Time:                  cfg.KeepAliveTime,
		Timeout:               cfg.KeepAliveTimeout,
	}
	if kaParams.MaxConnectionIdle > 0 || kaParams.MaxConnectionAge > 0 || kaParams.MaxConnectionAgeGrace > 0 || kaParams.Time > 0 || kaParams.Timeout > 0 {
		opts = append(opts, grpc.KeepaliveParams(kaParams))
	}

	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             cfg.KeepAliveMinTime,
		PermitWithoutStream: cfg.KeepAliveWithoutStream,
	}
	if kaPolicy.MinTime > 0 || kaPolicy.PermitWithoutStream {
		opts = append(opts, grpc.KeepaliveEnforcementPolicy(kaPolicy))
	}

	return opts
}
