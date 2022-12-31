package mmdb

import (
	"sync"

	C "github.com/Dreamacro/clash/constant"

	"github.com/oschwald/geoip2-golang"
)

var (
	mmdb *geoip2.Reader
	once sync.Once
)

func LoadFromBytes(buffer []byte) {
	once.Do(func() {
		mmdb, _ = geoip2.FromBytes(buffer)
	})
}

func DefaultInstance() *geoip2.Reader {
	once.Do(func() {
		mmdb, _ = geoip2.Open(C.Path.MMDB())
	})

	return mmdb
}
