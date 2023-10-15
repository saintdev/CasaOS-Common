package init_system

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type OpenRc struct{}

type serviceCmd string

const (
	cmdStart  serviceCmd = serviceCmd("start")
	cmdStop   serviceCmd = serviceCmd("stop")
	cmdList   serviceCmd = serviceCmd("list")
	cmdStatus serviceCmd = serviceCmd("status")
)

func (rc *OpenRc) ListServices(pattern string) ([]InitService, error) {
	svc_list, err := RcServiceList()
	if err != nil {
		return nil, err
	}

	if pattern != "" && pattern != "*" {
		n := 0
		for _, svc_name := range svc_list {
			matched, err := filepath.Match(pattern, svc_name)
			if err != nil {
				return nil, err
			}
			if matched {
				svc_list[n] = svc_name
				n++
			}
		}
		svc_list = svc_list[:n]
	}

	services := make([]InitService, 0, len(svc_list))
	for _, svc_name := range svc_list {
		running, err := rc.IsServiceRunning(svc_name)
		if err != nil {
			return nil, err
		}
		services = append(services, InitService{
			Name:    svc_name,
			Running: running,
		})
	}

	return services, nil
}

func (rc *OpenRc) IsServiceRunning(name string) (bool, error) {
	return RcServiceStatus(name)
}

func (rc *OpenRc) IsServiceEnabled(name string) (bool, error) {
	// TODO
	return false, nil
}

func (rc *OpenRc) EnableService(name string) error {
	return RcUpdate("add", name, "default")
}

func (rc *OpenRc) DisableService(name string) error {
	return RcUpdate("del", name, "default")
}

func (rc *OpenRc) StartService(name string) error {
	return RcServiceStart(name)
}

func (rc *OpenRc) StopService(name string) error {
	return RcServiceStop(name)
}

func (rc *OpenRc) Reload() error {
	// Nothing needed to reload OpenRC
	return nil
}

func RcUpdate(command string, name string, runlevel string) error {
	args := []string{"--quiet", command, name, runlevel}
	cmd := exec.Command("/sbin/rc-update", args...)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func RcService(name string, command serviceCmd) ([]byte, error) {
	args := make([]string, 0)
	if command != cmdList {
		args = append(args, "--quiet")
	}

	// Strip the ".service" suffix if the service name got passed with it.
	name, _ = strings.CutSuffix(name, ".service")

	switch command {
	case cmdStart:
		args = append(args, name, "start")
	case cmdStop:
		args = append(args, name, "stop")
	case cmdStatus:
		args = append(args, name, "status")
	case cmdList:
		args = append(args, "--list")
	}
	cmd := exec.Command("/sbin/rc-service", args...)

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func RcServiceStart(name string) error {
	if _, err := RcService(name, cmdStart); err != nil {
		return err
	}
	return nil
}

func RcServiceStop(name string) error {
	if _, err := RcService(name, cmdStop); err != nil {
		return err
	}
	return nil
}

func RcServiceStatus(name string) (bool, error) {
	if _, err := RcService(name, cmdStatus); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func RcServiceList() ([]string, error) {
	out, err := RcService("", cmdList)
	if err != nil {
		return nil, err
	}

	str_out := fmt.Sprint(out)

	return strings.Split(str_out, "\n"), nil
}
