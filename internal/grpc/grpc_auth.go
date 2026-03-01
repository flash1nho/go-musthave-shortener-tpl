package grpc

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/authenticator"
)

type grpcProvider struct {
	ctx context.Context
}

func (p *grpcProvider) GetCookie(_ string) (string, error) {
	var cookie string

	if md, ok := metadata.FromIncomingContext(p.ctx); ok {
		values := md.Get("authorization")

		if len(values) > 0 {
			cookie = values[0]
		}
	}

	return cookie, nil
}

func (p *grpcProvider) SetCookie(cookieName, cookieValue string) error {
	header := metadata.Pairs("set-cookie", cookieName+"="+cookieValue+"; Path=/")
	return grpc.SendHeader(p.ctx, header)
}

func Auth(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	newCtx, err := authenticator.Authenticate(ctx, &grpcProvider{ctx})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return handler(newCtx, req)
}
