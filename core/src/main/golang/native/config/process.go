package config

import (
	"cfa/native/common"

	"github.com/Dreamacro/clash/config"
)

var processors = []processor{
	patchOverride,
	patchProfile,
	patchDns,
	patchProviders,
}

type processor func(cfg *config.RawConfig, profileDir string) error

func patchOverride(cfg *config.RawConfig, _ string) error {
	return nil
}

func patchProfile(cfg *config.RawConfig, _ string) error {
	cfg.Profile.StoreSelected = false
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
		if path, ok := provider["path"].(string); ok {
			provider["path"] = profileDir + "/providers/" + common.ResolveAsRoot(path)
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
