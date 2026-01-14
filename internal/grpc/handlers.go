package grpc

import (
	"context"
	"errors"

	pb "github.com/Popolzen/shortener/api/proto"
	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ShortenerServer реализует gRPC сервис сокращения URL
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	service   shortener.URLService
	config    *config.Config
	publisher *audit.Publisher
}

// NewShortenerServer создает новый gRPC сервер
func NewShortenerServer(service shortener.URLService, cfg *config.Config, pub *audit.Publisher) *ShortenerServer {
	return &ShortenerServer{
		service:   service,
		config:    cfg,
		publisher: pub,
	}
}

// ShortenURL создает короткую ссылку (POST /api/shorten)
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL cannot be empty")
	}

	// Получаем userID из контекста (установлен interceptor'ом)
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Вызываем бизнес-логику
	shortURL, err := s.service.Shorten(req.Url, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to shorten URL")
	}

	fullShortURL := s.config.BaseURL + "/" + shortURL

	// Аудит
	s.publisher.Publish(audit.NewEvent(audit.ActionShorten, userID, req.Url))

	return &pb.URLShortenResponse{
		Result: fullShortURL,
	}, nil
}

// ExpandURL получает оригинальный URL (GET /:id)
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ID cannot be empty")
	}

	longURL, err := s.service.GetLongURL(req.Id)
	if err != nil {
		if errors.Is(err, model.ErrURLDeleted) {
			return nil, status.Error(codes.NotFound, "URL has been deleted")
		}
		return nil, status.Error(codes.NotFound, "URL not found")
	}

	// Аудит
	userID, _ := ctx.Value(userIDKey).(string)
	s.publisher.Publish(audit.NewEvent(audit.ActionFollow, userID, longURL))

	return &pb.URLExpandResponse{
		Result: longURL,
	}, nil
}

// ListUserURLs возвращает все URL пользователя (GET /api/user/urls)
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	urls, err := s.service.GetFormattedUserURLs(userID, s.config.BaseURL)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user URLs")
	}

	// Конвертируем в protobuf формат
	var pbURLs []*pb.URLData
	for _, u := range urls {
		pbURLs = append(pbURLs, &pb.URLData{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{
		Urls: pbURLs,
	}, nil
}
