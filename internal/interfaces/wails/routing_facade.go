package wails

import (
	"context"

	relayhttp "codeswitch/internal/interfaces/http/relay"
	routingapp "codeswitch/internal/routing/application"
	routingdomain "codeswitch/internal/routing/domain"
)

type RoutingFacade struct {
	service     *routingapp.Service
	relayServer *relayhttp.Server
}

func NewRoutingFacade(service *routingapp.Service, relayServer *relayhttp.Server) *RoutingFacade {
	return &RoutingFacade{service: service, relayServer: relayServer}
}

func (f *RoutingFacade) GetProfile(platform string) (routingdomain.RouteProfile, error) {
	return f.service.GetProfile(context.Background(), platform)
}

func (f *RoutingFacade) SaveProfile(profile routingdomain.RouteProfile) (routingdomain.RouteProfile, error) {
	return f.service.SaveProfile(context.Background(), profile)
}

func (f *RoutingFacade) ExportProviderConfig() (routingdomain.ProviderConfigBundle, error) {
	return f.service.ExportProviderConfig(context.Background())
}

func (f *RoutingFacade) ImportProviderConfig(bundle routingdomain.ProviderConfigBundle) (routingdomain.ProviderConfigBundle, error) {
	return f.service.ImportProviderConfig(context.Background(), bundle)
}

func (f *RoutingFacade) GetAppPreferences() (routingdomain.AppPreferences, error) {
	return f.service.GetAppPreferences(context.Background())
}

func (f *RoutingFacade) SaveAppPreferences(preferences routingdomain.AppPreferences) (routingdomain.AppPreferences, error) {
	saved, err := f.service.SaveAppPreferences(context.Background(), preferences)
	if err != nil {
		return routingdomain.AppPreferences{}, err
	}
	if f.relayServer != nil {
		f.relayServer.SetProxyConfig(saved.ProxyEnabled, saved.ProxyURL)
	}
	return saved, nil
}
