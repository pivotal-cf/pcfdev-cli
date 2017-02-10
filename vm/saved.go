package vm

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
)

type Saved struct {
	VMConfig *config.VMConfig
	Config   *config.Config

	FS        FS
	UI        UI
	VBox      VBox
	SSHClient SSH
}

func (s *Saved) VerifyStartOpts(opts *StartOpts) error {
	if opts.Memory != uint64(0) {
		return errors.New("memory cannot be changed once the vm has been created")
	}
	if opts.CPUs != 0 {
		return errors.New("cores cannot be changed once the vm has been created")
	}
	if opts.Services != "" {
		return errors.New("services cannot be changed once the vm has been created")
	}
	if opts.Domain != "" {
		return errors.New("the -d flag cannot be used if the VM has already been created")
	}
	if opts.IP != "" {
		return errors.New("the -i flag cannot be used if the VM has already been created")
	}
	if err := s.checkMemory(); err != nil {
		return err
	}
	return nil
}

func (s *Saved) Start(opts *StartOpts) error {
	return s.Resume()
}

func (s *Saved) Stop() error {
	s.UI.Say("Your VM is currently suspended. You must resume your VM with `cf dev resume` to shut it down.")
	return nil
}

func (s *Saved) Status() string {
	return "Suspended"
}

func (s *Saved) Suspend() error {
	s.UI.Say("Your VM is suspended.")
	return nil
}

func (s *Saved) Resume() error {
	if err := s.checkMemory(); err != nil {
		return err
	}
	s.UI.Say("Resuming VM...")
	if err := s.VBox.ResumeSavedVM(s.VMConfig); err != nil {
		return &ResumeVMError{err}
	}

	privateKeyBytes, err := s.FS.Read(s.Config.PrivateKeyPath)
	if err != nil {
		return &ResumeVMError{err}
	}

	addresses := []ssh.SSHAddress{
		{IP: "127.0.0.1", Port: s.VMConfig.SSHPort},
		{IP: s.VMConfig.IP, Port: "22"},
	}
	if err := s.SSHClient.WaitForSSH(addresses, privateKeyBytes, 5*time.Minute); err != nil {
		return &ResumeVMError{err}
	}

	s.UI.Say("PCF Dev is now running.")

	return nil
}

func (s *Saved) checkMemory() error {
	if s.VMConfig.Memory > s.Config.FreeMemory {
		if !s.UI.Confirm(fmt.Sprintf("Less than %d MB of free memory detected, continue (y/N): ", s.VMConfig.Memory)) {
			return errors.New("user declined to continue, exiting")
		}
	}
	return nil
}

func (s *Saved) GetDebugLogs() error {
	s.UI.Say("Your VM is suspended. Resume to retrieve debug logs.")
	return nil
}

func (s *Saved) Trust(startOps *StartOpts) error {
	s.UI.Say("Your VM is suspended. Resume to trust VM certificates.")
	return nil
}

func (s *Saved) Target(autoTarget bool) error {
	s.UI.Say("Your VM is suspended. Resume to target PCF Dev.")
	return nil
}

func (s *Saved) SSH() error {
	s.UI.Say("Your VM is suspended. Resume to SSH to PCF Dev.")
	return nil
}
