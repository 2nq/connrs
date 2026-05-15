//go:build windows

package collector

func NewDefault() (Poller, error) {
	return New(newWindowsSampler(), isAdministrator), nil
}
