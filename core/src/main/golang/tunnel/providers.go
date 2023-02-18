package tunnel

import (
	"fmt"

	"github.com/Dreamacro/clash/constant/provider"
	"github.com/Dreamacro/clash/tunnel"
)

type Provider struct {
	Name        string `json:"name"`
	VehicleType string `json:"vehicleType"`
	Type        string `json:"type"`
}

func QueryProviders() []*Provider {
	p := tunnel.Providers()

	providers := make([]provider.Provider, 0, len(p))

	for _, proxy := range p {
		if proxy.VehicleType() == provider.Compatible {
			continue
		}

		providers = append(providers, proxy)
	}

	result := make([]*Provider, 0, len(providers))

	for _, p := range providers {
		result = append(result, &Provider{
			Name:        p.Name(),
			VehicleType: p.VehicleType().String(),
			Type:        p.Type().String(),
		})
	}

	return result
}

func UpdateProvider(_ string, name string) error {
	p, ok := tunnel.Providers()[name]
	if !ok {
		return fmt.Errorf("%s not found", name)
	}

	return p.Update()
}
