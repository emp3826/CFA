package all

import (
	_ "cfa/app"
	_ "cfa/common"
	_ "cfa/config"
	_ "cfa/delegate"
	_ "cfa/proxy"
	_ "cfa/tun"
	_ "cfa/tunnel"
	_ "golang.org/x/sync/semaphore"
)