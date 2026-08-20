package wails

import (
	"context"

	routingapp "codeswitch/internal/routing/application"
	routingdomain "codeswitch/internal/routing/domain"
)

type RoutingFacade struct {
	service *routingapp.Service
}

func NewRoutingFacade(service *routingapp.Service) *RoutingFacade {
	return &RoutingFacade{service: service}
}

func (f *RoutingFacade) GetProfile(platform string) (routingdomain.RouteProfile, error) {
	return f.service.GetProfile(context.Background(), platform)
}

func (f *RoutingFacade) SaveProfile(profile routingdomain.RouteProfile) (routingdomain.RouteProfile, error) {
	return f.service.SaveProfile(context.Background(), profile)
}

func (f *RoutingFacade) GetAppPreferences() (routingdomain.AppPreferences, error) {
	return f.service.GetAppPreferences(context.Background())
}

func (f *RoutingFacade) SaveAppPreferences(preferences routingdomain.AppPreferences) (routingdomain.AppPreferences, error) {
	return f.service.SaveAppPreferences(context.Background(), preferences)
}
