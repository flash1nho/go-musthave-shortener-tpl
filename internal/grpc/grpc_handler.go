package grpc

import (
	"context"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/facade"
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

	userID, err := g.facade.GetUserFromContext(ctx)

	if err != nil {
		return nil, err
	}

	result, err := g.facade.PostURLFacade(ctx, userID, req.URL)

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

func (g *GrpcHandler) ListUserURLs(ctx context.Context, req *UserURLsRequest) (*UserURLsResponse, error) {
	var response UserURLsResponse

	userID, err := g.facade.GetUserFromContext(ctx)

	if err != nil {
		return nil, err
	}

	result, err := g.facade.APIUserURLFacade(ctx, userID)

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

	response.URLs = &grpcURLs

	return &response, nil
}
