package config

import (
	"github.com/Dreamacro/clash/config"
)

var processors = []processor{
	patchOverride,
	patchDns,
	patchProviders,
}

type processor func(cfg *config.RawConfig, profileDir string) error

func patchOverride(cfg *config.RawConfig, _ string) error {
	return nil
}

func patchDns(cfg *config.RawConfig, _ string) error {
	if !cfg.DNS.Enable {
		cfg.DNS = config.RawDNS{
			Enable:            true,
			UseHosts:          true,
			DefaultNameserver: defaultNameServers,
			NameServer:        defaultNameServers,
		}

		cfg.ClashForAndroid.AppendSystemDNS = true
	}
	return nil
}

func patchProviders(cfg *config.RawConfig, profileDir string) error {
	forEachProviders(cfg, func(index int, total int, key string, provider map[string]any) {
		if _, ok := provider["path"].(string); ok {
			provider["path"] = profileDir + "/providers/"
		}
	})
	return nil
}

func process(cfg *config.RawConfig, profileDir string) error {
	for _, p := range processors {
		if err := p(cfg, profileDir); err != nil {
			return err
		}
	}
	return nil
}
