package dns

import (
	"net"

	"github.com/Dreamacro/clash/common/cache"
	C "github.com/Dreamacro/clash/constant"
)

type ResolverEnhancer struct {
	mode     C.DNSMode
	mapping  *cache.LruCache
}

func (h *ResolverEnhancer) MappingEnabled() bool {
	return h.mode == C.DNSMapping
}

func (h *ResolverEnhancer) FindHostByIP(ip net.IP) (string, bool) {
	if mapping := h.mapping; mapping != nil {
		if host, existed := h.mapping.Get(ip.String()); existed {
			return host.(string), true
		}
	}

	return "", false
}

func (h *ResolverEnhancer) PatchFrom(o *ResolverEnhancer) {
	if h.mapping != nil && o.mapping != nil {
		o.mapping.CloneTo(h.mapping)
	}
}

func NewEnhancer(cfg Config) *ResolverEnhancer {
	var mapping *cache.LruCache

	if cfg.EnhancedMode != C.DNSNormal {
		mapping = cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true))
	}

	return &ResolverEnhancer{
		mode:     cfg.EnhancedMode,
		mapping:  mapping,
	}
}
