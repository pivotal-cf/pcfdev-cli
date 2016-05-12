package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/vbox"
)

type Plugin struct {
	PivnetClient        Client
	SSH                 SSH
	UI                  UI
	VBox                VBox
	FS                  FS
	Config              Config
	RequirementsChecker RequirementsChecker

	ExpectedMD5 string
	VMName      string
}

//go:generate mockgen -package mocks -destination mocks/client.go github.com/pivotal-cf/pcfdev-cli/plugin Client
type Client interface {
	DownloadOVA(token string) (ova *pivnet.DownloadReader, err error)
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/plugin SSH
type SSH interface {
	RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin UI
type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
}

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/plugin VBox
type VBox interface {
	StartVM(name string) (vm *vbox.VM, err error)
	StopVM(name string) error
	DestroyVMs(name []string) error
	ImportVM(path string, name string) error
	Status(name string) (status string, err error)
	ConflictingVMPresent(name string) (conflict bool, err error)
	GetPCFDevVMs() (names []string, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin FS
type FS interface {
	Exists(path string) (exists bool, err error)
	Write(path string, contents io.Reader) error
	CreateDir(path string) error
	RemoveFile(path string) error
	MD5(path string) (md5 string, err error)
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/pivotal-cf/pcfdev-cli/plugin Config
type Config interface {
	GetToken() string
}

//go:generate mockgen -package mocks -destination mocks/requirements_checker.go github.com/pivotal-cf/pcfdev-cli/plugin RequirementsChecker
type RequirementsChecker interface {
	Check() error
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	if len(args) != 2 {
		p.UI.Failed("Usage: %s", p.GetMetadata().Commands[0].UsageDetails.Usage)
		return
	}

	switch args[1] {
	case "download":
		if err := p.downloadVM(); err != nil {
			p.UI.Failed(err.Error())
		}
	case "start":
		if err := p.start(); err != nil {
			p.UI.Failed(err.Error())
		}
	case "status":
		if status, err := p.VBox.Status(p.VMName); err != nil {
			p.UI.Failed(err.Error())
		} else {
			p.UI.Say(status)
		}
	case "stop":
		if err := p.stop(); err != nil {
			p.UI.Failed(err.Error())
		}
	case "destroy":
		if err := p.destroy(); err != nil {
			p.UI.Failed(err.Error())
		}
	}
}

func (p *Plugin) downloadVM() error {
	return p.getOVAFile()
}

func (p *Plugin) start() error {
	if err := p.RequirementsChecker.Check(); err != nil {
		return fmt.Errorf("Could not start PCF Dev: %s", err)
	}

	if err := p.getOVAFile(); err != nil {
		return err
	}

	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %s", err)
	}

	if status == vbox.StatusRunning {
		p.UI.Say("PCF Dev is running")
		return nil
	}

	if status == vbox.StatusNotCreated {
		p.UI.Say("Importing VM...")
		err = p.VBox.ImportVM(p.ovaPath(), p.VMName)
		if err != nil {
			return fmt.Errorf("failed to import VM: %s", err)
		}
		p.UI.Say("PCF Dev is now imported to Virtualbox")
	}

	p.UI.Say("Starting VM...")
	vm, err := p.VBox.StartVM(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to start VM: %s", err)
	}
	p.UI.Say("Provisioning VM...")
	err = p.provision(vm)
	if err != nil {
		return fmt.Errorf("failed to provision VM: %s", err)
	}

	p.UI.Say("PCF Dev is now running")
	return nil
}

func (p *Plugin) stop() error {
	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return err
	}

	if status == vbox.StatusNotCreated {
		conflict, err := p.VBox.ConflictingVMPresent(p.VMName)
		if err != nil {
			return err
		}
		if conflict {
			return errors.New("Old version of PCF Dev detected. You must run `cf dev destroy` to continue.")
		}
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	if status == vbox.StatusStopped {
		p.UI.Say("PCF Dev is stopped")
		return nil
	}

	p.UI.Say("Stopping VM...")
	err = p.VBox.StopVM(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %s", err)
	}
	p.UI.Say("PCF Dev is now stopped")
	return nil
}

func (p *Plugin) destroy() error {
	vms, err := p.VBox.GetPCFDevVMs()
	if err != nil {
		return fmt.Errorf("failed to query VM: %s", err)
	}

	if len(vms) == 0 {
		p.UI.Say("PCF Dev VM has not been created")
		return nil
	}

	p.UI.Say("Destroying VM...")
	err = p.VBox.DestroyVMs(vms)
	if err != nil {
		return fmt.Errorf("failed to destroy VM: %s", err)
	}
	p.UI.Say("PCF Dev VM has been destroyed")
	return nil
}

func (p *Plugin) provision(vm *vbox.VM) error {
	return p.SSH.RunSSHCommand(fmt.Sprintf("sudo /var/pcfdev/run %s %s '$2a$04$EpJtIJ8w6hfCwbKYBkn3t.GCY18Pk6s7yN66y37fSJlLuDuMkdHtS'", vm.Domain, vm.IP), vm.SSHPort, 2*time.Minute, os.Stdout, os.Stderr)
}

func (p *Plugin) downloadOVAFile() error {
	token := p.Config.GetToken()

	p.UI.Say("Downloading VM...")
	ova, err := p.PivnetClient.DownloadOVA(token)
	if err != nil {
		return err
	}
	defer ova.Close()

	p.FS.Write(p.ovaPath(), ova)
	p.UI.Say("\nFinished downloading VM")
	return nil
}

func (p *Plugin) pcfdevDir() string {
	if pcfdevHome := os.Getenv("PCFDEV_HOME"); pcfdevHome != "" {
		return pcfdevHome
	}

	return filepath.Join(os.Getenv("HOME"), ".pcfdev")
}

func (p *Plugin) ovaPath() string {
	return filepath.Join(p.pcfdevDir(), "pcfdev.ova")
}

func (p *Plugin) getOVAFile() error {
	err := p.FS.CreateDir(p.pcfdevDir())
	if err != nil {
		return err
	}

	ovaExists, err := p.FS.Exists(p.ovaPath())
	if err != nil {
		return err
	}

	if !ovaExists {
		return p.downloadOVAFile()
	}

	ovaMD5, err := p.FS.MD5(p.ovaPath())
	if err != nil {
		return fmt.Errorf("failed to compute checksum of %s", p.ovaPath())
	}

	if ovaMD5 == p.ExpectedMD5 {
		p.UI.Say("VM already downloaded")
		return nil
	}

	status, err := p.VBox.Status(p.VMName)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %s", err)
	}

	if status != vbox.StatusNotCreated {
		return errors.New("Old version of PCF Dev detected. You must run `cf dev destroy` to continue.")
	}

	p.UI.Say("Upgrading your locally stored version of PCF Dev...")
	err = p.FS.RemoveFile(p.ovaPath())
	if err != nil {
		return fmt.Errorf("failed to remove old machine image %s", p.ovaPath())
	}

	return p.downloadOVAFile()

}

func (*Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "pcfdev",
		Commands: []plugin.Command{
			plugin.Command{
				Name:  "dev",
				Alias: "pcfdev",
				UsageDetails: plugin.Usage{
					Usage: "cf dev download|start|status|stop|destroy",
				},
			},
		},
	}
}
