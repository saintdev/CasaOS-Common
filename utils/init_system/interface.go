package init_system

import (
	"os"
)

type InitService struct {
	Name    string
	Running bool
}

type InitManager interface {
	ListServices(pattern string) ([]InitService, error)
	IsServiceEnabled(name string) (bool, error)
	IsServiceRunning(name string) (bool, error)
	EnableService(name string) error
	DisableService(name string) error
	StartService(name string) error
	StopService(name string) error
	Reload() error
}

func NewInitManager() InitManager {
	// From the man page for `sd_booted();`
	// Internally, this function checks whether the directory /run/systemd/system/ exists. A simple check like this can also be implemented trivially in shell or any other language.
	if _, err := os.Stat("/run/systemd/system/"); os.IsNotExist(err) {
		// Not using systemd
		return &OpenRc{}
	}

	return &SystemCtl{}
}
