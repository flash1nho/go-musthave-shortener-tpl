package grpc

import (
	"context"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/facade"
	"github.com/golang/protobuf/ptypes/empty"
)

type GrpcHandler struct {
	UnimplementedShortenerServiceServer

	facade *facade.Facade
}

func NewHandler(facade *facade.Facade) *GrpcHandler {
	return &GrpcHandler{
		facade: facade,
	}
}

func (g *GrpcHandler) ShortenURL(ctx context.Context, req *URLShortenRequest) (*URLShortenResponse, error) {
	var response URLShortenResponse

	userID := g.facade.GetUserFromContext(ctx)
	result, err := g.facade.PostURLFacade(userID, req.URL)

	if err != nil {
		return nil, err
	}

	response.Result = result

	return &response, nil
}

func (g *GrpcHandler) ExpandURL(ctx context.Context, req *URLExpandRequest) (*URLExpandResponse, error) {
	var response URLExpandResponse

	URLDetails, err := g.facade.GetURLFacade(req.ID)

	if err != nil {
		return nil, err
	}

	response.Result = URLDetails.OriginalURL

	return &response, nil
}

func (g *GrpcHandler) ListUserURLs(ctx context.Context, _ *empty.Empty) (*UserURLsResponse, error) {
	var response UserURLsResponse

	userID := g.facade.GetUserFromContext(ctx)
	result, err := g.facade.APIUserURLFacade(userID)

	if err != nil {
		return nil, err
	}

	grpcURLs := make([]*URLData, 0, len(result))

	for _, v := range result {
		grpcURLs = append(grpcURLs, &URLData{
			ShortURL:    v.ShortURL,
			OriginalURL: v.OriginalURL,
		})
	}

	response.URL = &grpcURLs

	return &response, nil
}
