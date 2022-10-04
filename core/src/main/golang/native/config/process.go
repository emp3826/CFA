package config

import (
	"errors"

	"cfa/native/common"

	"github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/dns"
)

var processors = []processor{
	patchOverride,
	patchGeneral,
	patchProfile,
	patchDns,
	patchProviders,
	validConfig,
}

type processor func(cfg *config.RawConfig, profileDir string) error

func patchOverride(cfg *config.RawConfig, _ string) error {
	return nil
}

func patchGeneral(cfg *config.RawConfig, _ string) error {
	cfg.Interface = ""
	cfg.ExternalUI = ""
	cfg.ExternalController = ""

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

	if cfg.ClashForAndroid.AppendSystemDNS {
		cfg.DNS.NameServer = append(cfg.DNS.NameServer, "dhcp://"+dns.SystemDNSPlaceholder)
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

func validConfig(cfg *config.RawConfig, _ string) error {
	if len(cfg.Proxy) == 0 && len(cfg.ProxyProvider) == 0 {
		return errors.New("profile does not contain `proxies` or `proxy-providers`")
	}

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
