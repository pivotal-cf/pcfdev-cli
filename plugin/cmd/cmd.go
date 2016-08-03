package cmd

import (
	"errors"

	"github.com/cloudfoundry/cli/flags"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vm"
)

//go:generate mockgen -package mocks -destination mocks/ui.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd UI
type UI interface {
	Say(message string, args ...interface{})
	Confirm(message string, args ...interface{}) bool
}

//go:generate mockgen -package mocks -destination mocks/vbox.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd VBox
type VBox interface {
	GetVMName() (name string, err error)
	DestroyPCFDevVMs() (err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd FS
type FS interface {
	Remove(path string) error
	Copy(source string, destination string) error
	MD5(path string) (md5 string, err error)
}

//go:generate mockgen -package mocks -destination mocks/downloader.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd Downloader
type Downloader interface {
	Download() error
	IsOVACurrent() (bool, error)
}

//go:generate mockgen -package mocks -destination mocks/vm_builder.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd VMBuilder
type VMBuilder interface {
	VM(name string) (vm vm.VM, err error)
}

//go:generate mockgen -package mocks -destination mocks/cmd.go github.com/pivotal-cf/pcfdev-cli/plugin/cmd Cmd
type Cmd interface {
	Parse([]string) error
	Run() error
}

func parse(flagContext flags.FlagContext, args []string, expectedLength int) error {
	if err := flagContext.Parse(args...); err != nil {
		return err
	}
	if len(flagContext.Args()) != expectedLength {
		return errors.New("wrong number of arguments")
	}
	return nil
}

type Builder struct {
	Client     Client
	Config     *config.Config
	Downloader Downloader
	EULAUI     EULAUI
	FS         FS
	UI         UI
	VBox       VBox
	VMBuilder  VMBuilder
}

func (b *Builder) Cmd(subcommand string) (Cmd, error) {
	switch subcommand {
	case "destroy":
		return &DestroyCmd{
			VBox:   b.VBox,
			UI:     b.UI,
			FS:     b.FS,
			Config: b.Config,
		}, nil
	case "download":
		return &DownloadCmd{
			VBox:       b.VBox,
			UI:         b.UI,
			EULAUI:     b.EULAUI,
			Client:     b.Client,
			Downloader: b.Downloader,
			FS:         b.FS,
			Config:     b.Config,
		}, nil
	case "import":
		return &ImportCmd{
			Downloader: b.Downloader,
			UI:         b.UI,
			Config:     b.Config,
			FS:         b.FS,
		}, nil
	case "provision":
		return &ProvisionCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "resume":
		return &ResumeCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "start":
		return &StartCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
			DownloadCmd: &DownloadCmd{
				VBox:       b.VBox,
				UI:         b.UI,
				EULAUI:     b.EULAUI,
				Client:     b.Client,
				Downloader: b.Downloader,
				FS:         b.FS,
				Config:     b.Config,
			},
		}, nil
	case "status":
		return &StatusCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
			UI:        b.UI,
		}, nil
	case "stop":
		return &StopCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "suspend":
		return &SuspendCmd{
			VBox:      b.VBox,
			VMBuilder: b.VMBuilder,
			Config:    b.Config,
		}, nil
	case "version", "--version":
		return &VersionCmd{
			UI:     b.UI,
			Config: b.Config,
		}, nil
	default:
		return nil, errors.New("")
	}
}