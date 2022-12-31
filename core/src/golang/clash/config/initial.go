package config

import (
	"fmt"
	"os"

	C "github.com/Dreamacro/clash/constant"
)

func initMMDB() error {
	if err := os.Remove(C.Path.MMDB()); err != nil {
		return fmt.Errorf("can't remove invalid MMDB: %s", err.Error())
	}
	return nil
}

// Init prepare necessary files
func Init(dir string) error {
	// initial homedir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return fmt.Errorf("can't create config directory %s: %s", dir, err.Error())
		}
	}

	// initial config.yaml
	if _, err := os.Stat(C.Path.Config()); os.IsNotExist(err) {
		f, err := os.OpenFile(C.Path.Config(), os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("can't create file %s: %s", C.Path.Config(), err.Error())
		}
		f.Write([]byte(`mixed-port: 7890`))
		f.Close()
	}

	// initial mmdb
	if err := initMMDB(); err != nil {
		return fmt.Errorf("can't initial MMDB: %w", err)
	}
	return nil
}
