package provider

import (
	"bytes"
	"crypto/md5"
	"os"
	"path/filepath"
	"time"

	types "github.com/Dreamacro/clash/constant/provider"
)

const (
	minInterval = time.Minute * 5
)

var (
	fileMode os.FileMode = 0o666
	dirMode  os.FileMode = 0o755
)

type parser = func([]byte) (any, error)

type fetcher struct {
	name      string
	vehicle   types.Vehicle
	interval  time.Duration
	done      chan struct{}
	hash      [16]byte
	parser    parser
}

func (f *fetcher) Name() string {
	return f.name
}

func (f *fetcher) VehicleType() types.VehicleType {
	return f.vehicle.Type()
}

func (f *fetcher) Initial() (any, error) {
	var (
		buf     []byte
		err     error
		isLocal bool
	)
	if _, fErr := os.Stat(f.vehicle.Path()); fErr == nil {
		buf, err = os.ReadFile(f.vehicle.Path())
		isLocal = true
	} else {
		buf, err = f.vehicle.Read()
	}

	if err != nil {
		return nil, err
	}

	proxies, err := f.parser(buf)
	if err != nil {
		if !isLocal {
			return nil, err
		}

		// parse local file error, fallback to remote
		buf, err = f.vehicle.Read()
		if err != nil {
			return nil, err
		}

		proxies, err = f.parser(buf)
		if err != nil {
			return nil, err
		}

		isLocal = false
	}

	if f.vehicle.Type() != types.File && !isLocal {
		if err := safeWrite(f.vehicle.Path(), buf); err != nil {
			return nil, err
		}
	}

	f.hash = md5.Sum(buf)

	// pull proxies automatically
	if f.interval > 0 {
		go f.pullLoop()
	}

	return proxies, nil
}

func (f *fetcher) Update() (any, bool, error) {
	buf, err := f.vehicle.Read()
	if err != nil {
		return nil, false, err
	}

	hash := md5.Sum(buf)
	if bytes.Equal(f.hash[:], hash[:]) {
		os.Chtimes(f.vehicle.Path(), time.Now(), time.Now())

		return nil, true, nil
	}

	proxies, err := f.parser(buf)
	if err != nil {
		return nil, false, err
	}

	if f.vehicle.Type() != types.File {
		if err := safeWrite(f.vehicle.Path(), buf); err != nil {
			return nil, false, err
		}
	}

	f.hash = hash

	return proxies, false, nil
}

func (f *fetcher) Destroy() error {
	if f.interval > 0 {
		f.done <- struct{}{}
	}
	return nil
}

func (f *fetcher) pullLoop() {
	initialInterval := f.interval
	if initialInterval < minInterval {
		initialInterval = minInterval
	}

	timer := time.NewTimer(initialInterval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			timer.Reset(f.interval)

			_, same, err := f.Update()
			if err != nil {
				continue
			}

			if same {
				continue
			}

		case <-f.done:
			return
		}
	}
}

func safeWrite(path string, buf []byte) error {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return err
		}
	}

	return os.WriteFile(path, buf, fileMode)
}

func newFetcher(name string, interval time.Duration, vehicle types.Vehicle, parser parser) *fetcher {
	return &fetcher{
		name:     name,
		interval: interval,
		vehicle:  vehicle,
		parser:   parser,
		done:     make(chan struct{}, 8),
	}
}
