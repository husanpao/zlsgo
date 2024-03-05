package daemon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sohaha/zlsgo/zshell"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type (
	windowsSystem  struct{}
	windowsService struct {
		i            Iface
		stopStartErr error
		*Config
		errSync sync.Mutex
	}
)

const version = "windows-service"

var interactive = false

func init() {
	var err error
	chooseSystem(windowsSystem{})
	interactive, err = svc.IsAnInteractiveSession()
	if err != nil {
		panic(err)
	}
}

func (windowsSystem) String() string {
	return version
}

func (windowsSystem) Detect() bool {
	return true
}

func (windowsSystem) Interactive() bool {
	return interactive
}

func (windowsSystem) New(i Iface, c *Config) (ServiceIface, error) {
	if c.Context == nil {
		c.Context = context.Background()
	}

	ws := &windowsService{
		i:      i,
		Config: c,
	}

	return ws, nil
}

func (w *windowsService) String() string {
	if len(w.DisplayName) > 0 {
		return w.DisplayName
	}
	return w.Name
}

func (w *windowsService) setError(err error) {
	w.errSync.Lock()
	defer w.errSync.Unlock()
	w.stopStartErr = err
}

func (w *windowsService) getError() error {
	w.errSync.Lock()
	defer w.errSync.Unlock()
	return w.stopStartErr
}

func (w *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	if err := w.i.Start(w); err != nil {
		w.setError(err)
		return true, 1
	}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			if err := w.i.Stop(w); err != nil {
				w.setError(err)
				return true, 2
			}
			break loop
		default:
			continue loop
		}
	}

	return false, 0
}

func (w *windowsService) Install() error {
	m, err := connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	exepath := w.execPath()
	s, err := m.OpenService(w.Name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", w.Name)
	}
	password := ""
	if p, ok := w.Options["Password"]; ok {
		password, _ = p.(string)
	}
	s, err = m.CreateService(w.Name, exepath, mgr.Config{
		DisplayName:      w.DisplayName,
		Description:      w.Description,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: w.UserName,
		Password:         password,
	}, w.Arguments...)
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(w.Name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		_ = s.Delete()
		return fmt.Errorf("installAsEventCreate() failed: %s", err)
	}

	if isServiceRestart(w.Config) {
		_ = s.SetRecoveryActions([]mgr.RecoveryAction{
			{
				Type:  mgr.ServiceRestart,
				Delay: 0,
			},
		}, 0)
	}

	return nil
}

func (w *windowsService) Uninstall() error {
	m, err := connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.Name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", w.Name)
	}
	defer s.Close()

	_ = w.Stop()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(w.Name)
	if err != nil {
		return fmt.Errorf("removeEventLogSource() failed: %s", err)
	}

	return nil
}

func (w *windowsService) Run() error {
	w.setError(nil)
	if !interactive {
		runErr := svc.Run(w.Name, w)
		startStopErr := w.getError()
		if startStopErr != nil {
			return startStopErr
		}
		if runErr != nil {
			return runErr
		}
		return nil
	}
	err := w.i.Start(w)
	if err != nil {
		return err
	}

	select {
	case <-SingleKillSignal():
	case <-w.Config.Context.Done():
	}

	return w.i.Stop(w)
}

func (w *windowsService) Start() error {
	m, err := connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(w.Name)
	if err != nil {
		return err
	}
	defer s.Close()

	if isServiceRestart(w.Config) {
		_ = s.SetRecoveryActions([]mgr.RecoveryAction{
			{
				Type:  mgr.ServiceRestart,
				Delay: 0,
			},
		}, 0)
	}

	return s.Start()
}

func (w *windowsService) Stop() error {
	m, err := connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.Name)
	if err != nil {
		return err
	}
	defer s.Close()

	if isServiceRestart(w.Config) {
		_ = s.SetRecoveryActions([]mgr.RecoveryAction{
			{
				Type:  mgr.NoAction,
				Delay: 0,
			},
		}, 0)
	}

	return w.stopWait(s)
}

func (w *windowsService) Restart() error {
	err := w.Stop()
	if err != nil {
		return err
	}
	return w.Start()
}

func (w *windowsService) Status() string {
	m, err := connect()
	if err != nil {
		return "Unknown"
	}
	defer m.Disconnect()
	s, err := m.OpenService(w.Name)
	if err != nil {
		return err.Error()
	}
	defer s.Close()
	q, err := s.Query()
	if err != nil {
		return err.Error()
	}
	switch q.State {
	case svc.Running:
		return "Running"
	case svc.StopPending:
		return "StopPending"
	case svc.Stopped:
		return "Stop"
	}
	return strconv.Itoa(int(q.State))
}

func (w *windowsService) forceKeep(processId uint32) error {
	ss := "taskkill /F /pid " + strconv.Itoa(int(processId))
	_, _, _, err := zshell.Run(ss)
	return err
}

func (w *windowsService) stopWait(s *mgr.Service) error {
	status, err := s.Control(svc.Stop)
	if err != nil {
		if !strings.Contains(err.Error(), "not valid") {
			return err
		}
		status, err = s.Query()
		if err != nil {
			return err
		}
		_ = w.forceKeep(status.ProcessId)
	}

	timeDuration := time.Millisecond * 100
	timeout := time.After(getStopTimeout() + (timeDuration * 2))
	tick := time.NewTicker(timeDuration)
	defer tick.Stop()
	for status.State != svc.Stopped {
		select {
		case <-tick.C:
			status, err = s.Query()
			if err != nil {
				return err
			}
		case <-timeout:
			_ = w.forceKeep(status.ProcessId)
			break
		}
	}
	return nil
}

func connect() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			err = ErrNotAnAdministrator
		}
	}
	return m, err
}
func getStopTimeout() time.Duration {
	// For default and paths see https://support.microsoft.com/en-us/kb/146092
	defaultTimeout := time.Millisecond * 20000
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control`, registry.READ)
	if err != nil {
		return defaultTimeout
	}
	sv, _, err := key.GetStringValue("WaitToKillServiceTimeout")
	if err != nil {
		return defaultTimeout
	}
	v, err := strconv.Atoi(sv)
	if err != nil {
		return defaultTimeout
	}
	return time.Millisecond * time.Duration(v)
}
