package rest

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	v1 "github.com/GameComponent/economy-service/pkg/api/v1"
	"github.com/GameComponent/economy-service/pkg/protocol/rest/middleware"
)

// RunServer runs HTTP/REST gateway
func RunServer(ctx context.Context, logger *zap.Logger, grpcPort string, httpPort string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(
			runtime.MIMEWildcard,
			&runtime.JSONPb{OrigName: false},
		),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := v1.RegisterEconomyServiceHandlerFromEndpoint(ctx, mux, "0.0.0.0:"+grpcPort, opts); err != nil {
		logger.Fatal("failed to start HTTP gateway", zap.String("reason", err.Error()))
	}

	srv := &http.Server{
		Addr: ":" + httpPort,
		// add handler with middleware
		Handler: middleware.AddCors(
			middleware.AddRequestID(
				middleware.AddLogger(logger, mux),
			),
		),
	}

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// sig is a ^C, handle it
		}

		_, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(ctx)
	}()

	logger.Info("starting HTTP/REST gateway", zap.String("port", httpPort))
	return srv.ListenAndServe()
}
