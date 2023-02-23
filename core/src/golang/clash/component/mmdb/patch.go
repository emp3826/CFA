package mmdb

import (
	"github.com/oschwald/geoip2-golang"
)

func Instance() *geoip2.Reader {
	return DefaultInstance()
}
