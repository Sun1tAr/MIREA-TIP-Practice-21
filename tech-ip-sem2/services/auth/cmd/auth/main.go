package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grp "github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/auth/internal/grpc"
	pb "github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/proto/auth"
	"github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/shared/logger"
)

func main() {
	// Инициализация структурированного логгера
	logrusLogger := logger.Init("auth")

	grpcPort := os.Getenv("AUTH_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logrusLogger.WithError(err).Fatal("failed to listen")
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &grp.Server{Logger: logrusLogger})
	reflection.Register(s)

	go func() {
		logrusLogger.WithField("port", grpcPort).Info("Auth gRPC server starting")
		if err := s.Serve(lis); err != nil {
			logrusLogger.WithError(err).Fatal("failed to serve")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrusLogger.Info("Shutting down Auth gRPC server...")
	s.GracefulStop()
}
