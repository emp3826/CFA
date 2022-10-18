package delegate

import (
	"syscall"

	"cfa/blob"

	"cfa/native/app"

	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/mmdb"
	"github.com/Dreamacro/clash/constant"
)

func Init(home string) {
	mmdb.LoadFromBytes(blob.GeoipDatabase)
	constant.SetHomeDir(home)

	dialer.DefaultSocketHook = func(network, address string, conn syscall.RawConn) error {
		return conn.Control(func(fd uintptr) {
			app.MarkSocket(int(fd))
		})
	}
}
