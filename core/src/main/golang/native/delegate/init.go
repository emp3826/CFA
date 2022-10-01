package delegate

import (
	"errors"
	"syscall"

	"cfa/blob"

	"cfa/native/app"

	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/mmdb"
	"github.com/Dreamacro/clash/constant"
)

var errBlocked = errors.New("blocked")

func Init(home, versionName string, platformVersion int) {
	mmdb.LoadFromBytes(blob.GeoipDatabase)
	constant.SetHomeDir(home)

	dialer.DefaultSocketHook = func(network, address string, conn syscall.RawConn) error {
		return conn.Control(func(fd uintptr) {
			app.MarkSocket(int(fd))
		})
	}
}
