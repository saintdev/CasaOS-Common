package init_system

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

var (
	// `done` indicates successful execution of a job.
	ResultDone = "done"

	// `canceled` indicates that a job has been canceled before it finished execution.
	ResultCanceled = "canceled"
	ErrorCanceled  = errors.New("job has been canceled before it finished execution")

	// `timeout` indicates that the job timeout was reached.
	ResultTimeout = "timeout"
	ErrorTimeout  = errors.New("job timeout was reached")

	// `failed` indicates that the job failed.
	ResultFailed = "failed"
	ErrorFailed  = errors.New("job failed")

	// `dependency` indicates that a job this job has been depending on failed and the job hence has been removed too.
	ResultDependency = "dependency"
	ErrorDependency  = errors.New("another job this job has been depending on failed and the job hence has been removed too")

	// `skipped` indicates that a job was skipped because it didn't apply to the units current state.
	ResultSkipped = "skipped"
	ErrorSkipped  = errors.New("job was skipped because it didn't apply to the units current state")

	ErrorMap = map[string]error{
		ResultDone:       nil,
		ResultCanceled:   ErrorCanceled,
		ResultTimeout:    ErrorTimeout,
		ResultFailed:     ErrorFailed,
		ResultDependency: ErrorDependency,
		ResultSkipped:    ErrorSkipped,
	}

	ErrorUnknown = errors.New("unknown error")
)

type SystemCtl struct{}

func (s *SystemCtl) ListServices(pattern string) ([]InitService, error) {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	var files []dbus.UnitFile

	if pattern == "" || pattern == "*" {
		_files, err := conn.ListUnitFilesContext(ctx)
		if err != nil {
			return nil, err
		}

		files = _files
	} else {
		_files, err := conn.ListUnitFilesByPatternsContext(ctx, nil, []string{pattern})
		if err != nil {
			return nil, err
		}
		files = _files
	}

	services := make([]InitService, 0, len(files))

	for _, file := range files {
		serviceName := filepath.Base(file.Path)

		running, err := s.IsServiceRunning(serviceName)

		services = append(services, InitService{
			Name:    serviceName,
			Running: err == nil && running,
		})
	}

	return services, nil
}

func (s *SystemCtl) IsServiceEnabled(name string) (bool, error) {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return false, err
	}

	defer conn.Close()

	property, err := conn.GetUnitPropertyContext(ctx, name, "UnitFileState")
	if err != nil {
		return false, err
	}

	if property.Value.Value() == "enabled" {
		return true, nil
	}

	return false, nil
}

func (s *SystemCtl) IsServiceRunning(name string) (bool, error) {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return false, err
	}

	defer conn.Close()

	property, err := conn.GetUnitPropertyContext(ctx, name, "ActiveState")
	if err != nil {
		return false, err
	}

	return property.Value.Value() == "active", nil
}

func (s *SystemCtl) EnableService(name string) error {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	defer conn.Close()

	_, _, err = conn.EnableUnitFilesContext(ctx, []string{name}, false, true)
	if err != nil {
		return err
	}

	// ensure service is enabled
	property, err := conn.GetUnitPropertyContext(ctx, name, "ActiveState")
	if err != nil {
		return err
	}

	if property.Value.Value() != "active" {
		return s.StartService(name)
	}

	return nil
}

func (s *SystemCtl) DisableService(name string) error {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	defer conn.Close()

	// ensure service is stopped
	properties, err := conn.GetUnitPropertiesContext(ctx, name)
	if err != nil {
		return err
	}

	if properties["ActiveState"] == "active" {
		return s.StopService(name)
	}

	_, err = conn.DisableUnitFilesContext(ctx, []string{name}, false)
	if err != nil {
		return err
	}

	return nil
}

func (s *SystemCtl) StartService(name string) error {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	defer conn.Close()

	ch := make(chan string)
	_, err = conn.StartUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return err
	}

	result := <-ch
	if result != ResultDone {
		err, ok := ErrorMap[result]
		if !ok {
			return ErrorUnknown
		}

		return err
	}

	return nil
}

func (s *SystemCtl) StopService(name string) error {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	defer conn.Close()

	ch := make(chan string)
	_, err = conn.StopUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return err
	}

	result := <-ch
	if result != ResultDone {
		err, ok := ErrorMap[result]
		if !ok {
			return ErrorUnknown
		}

		return err
	}

	return nil
}

func (s *SystemCtl) Reload() error {
	// connect to systemd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	defer conn.Close()

	return conn.ReloadContext(ctx)
}
