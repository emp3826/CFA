package provider

var (
	suspended bool
)

func (pp *proxySetProvider) Close() error {
	pp.healthCheck.close()
	pp.fetcher.Destroy()

	return nil
}

func (cp *compatibleProvider) Close() error {
	cp.healthCheck.close()

	return nil
}

func Suspend(s bool) {
	suspended = s
}
