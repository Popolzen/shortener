package grpc

import (
	pb "github.com/Popolzen/shortener/api/proto"
	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/grpc/interceptors"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const userIDKey model.ContextKey = "user_id"

// NewServer создает и настраивает gRPC сервер
func NewServer(service shortener.URLService, cfg *config.Config, pub *audit.Publisher) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryInterceptor(cfg)),
	)

	shortenerServer := NewShortenerServer(service, cfg, pub)
	pb.RegisterShortenerServiceServer(srv, shortenerServer)

	reflection.Register(srv)

	return srv
}
