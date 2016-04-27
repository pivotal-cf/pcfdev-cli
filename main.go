package main

import (
	"os"

	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/fs"
	"github.com/pivotal-cf/pcfdev-cli/pivnet"
	"github.com/pivotal-cf/pcfdev-cli/plugin"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	"github.com/cloudfoundry/cli/cf/terminal"
	cfplugin "github.com/cloudfoundry/cli/plugin"
)

func main() {
	ui := terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	cfplugin.Start(&plugin.Plugin{
		UI:  ui,
		SSH: &ssh.SSH{},
		PivnetClient: &pivnet.Client{
			Host: "https://network.pivotal.io",
			Config: &config.Config{
				UI: ui,
			},
		},
		VBox: &vbox.VBox{
			SSH:    &ssh.SSH{},
			Driver: &vbox.VBoxDriver{},
		},
		FS: &fs.FS{},
	})
}
