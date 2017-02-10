package vm_test

import (
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stopped", func() {
	var (
		mockCtrl          *gomock.Controller
		mockFS            *mocks.MockFS
		mockUI            *mocks.MockUI
		mockVBox          *mocks.MockVBox
		mockSSH           *mocks.MockSSH
		mockBuilder       *mocks.MockBuilder
		mockUnprovisioned *mocks.MockVM
		stoppedVM         vm.Stopped
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockUnprovisioned = mocks.NewMockVM(mockCtrl)

		stoppedVM = vm.Stopped{
			VMConfig: &config.VMConfig{
				Name:     "some-vm",
				Domain:   "some-domain",
				IP:       "some-ip",
				SSHPort:  "some-port",
				Provider: "some-provider",
			},

			VBox:      mockVBox,
			FS:        mockFS,
			UI:        mockUI,
			SSHClient: mockSSH,
			Builder:   mockBuilder,
			Config: &config.Config{
				PrivateKeyPath: "some-private-key-path",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is stopped.")
			stoppedVM.Stop()
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired IP is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					IP: "some-ip",
				})).To(MatchError("the -i flag cannot be used if the VM has already been created"))
			})
		})

		Context("when desired domain is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Domain: "some-domain",
				})).To(MatchError("the -d flag cannot be used if the VM has already been created"))
			})
		})

		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when desired cores is passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when services are passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Services: "redis",
				})).To(MatchError("services cannot be changed once the vm has been created"))
			})
		})

		Context("when registries are passed", func() {
			It("should return an error", func() {
				Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{
					Registries: "some-private-registry",
				})).To(MatchError("private registries cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			Context("when free memory is greater than or equal to the VM's memory", func() {
				It("should succeed", func() {
					stoppedVM.Config.FreeMemory = uint64(3000)
					stoppedVM.VMConfig.Memory = uint64(2000)
					Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
				})
			})

			Context("when free memory is less than the VM's memory", func() {
				Context("when the user accepts to continue", func() {
					It("should succeed", func() {
						stoppedVM.Config.FreeMemory = uint64(2000)
						stoppedVM.VMConfig.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(true)

						Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
					})
				})

				Context("when the user declines to continue", func() {
					It("should return an error", func() {
						stoppedVM.Config.FreeMemory = uint64(2000)
						stoppedVM.VMConfig.Memory = uint64(3000)

						mockUI.EXPECT().Confirm("Less than 3000 MB of free memory detected, continue (y/N): ").Return(false)

						Expect(stoppedVM.VerifyStartOpts(&vm.StartOpts{})).To(MatchError("user declined to continue, exiting"))
					})
				})
			})
		})
	})

	Describe("Start", func() {

		var addresses []ssh.SSHAddress

		BeforeEach(func() {
			addresses = []ssh.SSHAddress{
				{
					IP:   "127.0.0.1",
					Port: "some-port",
				},
				{
					IP:   "some-ip",
					Port: "22",
				},
			}
		})

		allowHappyPathInteractions := func() {
			mockSSH.EXPECT().RunSSHCommand(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockUI.EXPECT().Say(gomock.Any()).AnyTimes()
			mockVBox.EXPECT().StartVM(gomock.Any()).AnyTimes()
			mockBuilder.EXPECT().VM(gomock.Any()).AnyTimes().Return(mockUnprovisioned, nil)
			mockFS.EXPECT().Read(gomock.Any()).AnyTimes().Return([]byte("some-private-key"), nil)
			mockUnprovisioned.EXPECT().Start(gomock.Any()).AnyTimes()
		}

		Context("when 'none' services are specified", func() {
			It("should start vm with no extra services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "none"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "none"})
			})
		})

		Context("when 'all' services are specified", func() {
			It("should start the vm with services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis,spring-cloud-services","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "all"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "all"})
			})
		})

		Context("when 'default' services are specified", func() {
			It("should start the vm with rabbitmq and redis", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "default"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "default"})
			})
		})

		Context("when 'spring-cloud-services' services are specified", func() {
			It("should start the vm with spring-cloud-services and rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,spring-cloud-services","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "spring-cloud-services"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "spring-cloud-services"})
			})
		})

		Context("when 'scs' is specified", func() {
			It("should start the vm with spring-cloud-services and rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,spring-cloud-services","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "scs"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "scs"})
			})
		})

		Context("when 'rabbitmq' services are specified", func() {
			It("should start the vm with rabbitmq", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "rabbitmq"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "rabbitmq"})
			})
		})

		Context("when 'redis' services are specified", func() {
			It("should start the vm with redis", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "redis"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "redis"})
			})
		})

		Context("when 'mysql' services are specified", func() {
			It("should start the vm with no extra services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "mysql"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "mysql"})
			})
		})

		Context("when duplicate services are specified", func() {
			It("should start the vm without duplicates services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis,spring-cloud-services","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Services: "default,spring-cloud-services,scs"}),
				)

				stoppedVM.Start(&vm.StartOpts{Services: "default,spring-cloud-services,scs"})
			})
		})

		Context("when '' services are specified", func() {
			It("should start the vm with default services", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true}),
				)

				stoppedVM.Start(&vm.StartOpts{})
			})
		})

		Context("when ip is specified specified", func() {
			It("should start the vm with the custom ip", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-custom-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, IP: "some-custom-ip"}),
				)

				stoppedVM.Start(&vm.StartOpts{IP: "some-custom-ip"})
			})
		})

		Context("when domain is specified specified", func() {
			It("should start the vm with the custom domain", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-custom-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Domain: "some-custom-domain"}),
				)

				stoppedVM.Start(&vm.StartOpts{Domain: "some-custom-domain"})
			})
		})

		Context("when docker registries are specified", func() {
			It("should start the vm with the registries accessible", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":["some-private-registry","some-other-private-registry"],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, Registries: "some-private-registry,some-other-private-registry"}),
				)

				stoppedVM.Start(&vm.StartOpts{Registries: "some-private-registry,some-other-private-registry"})
			})
		})

		Context("when the master password is specified", func() {
			It("should start the vm with the master password", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm").Return(mockUnprovisioned, nil),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true, MasterPassword: "some-master-password"}),
				)

				stoppedVM.Start(&vm.StartOpts{MasterPassword: "some-master-password"})
			})
		})

		Context("when '-n' (no-provision) flag is passed in", func() {
			It("should not provision the vm", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Starting VM..."),
					mockVBox.EXPECT().StartVM(stoppedVM.VMConfig),
					mockBuilder.EXPECT().VM("some-vm"),
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().RunSSHCommand("echo "+
						`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
						addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr),
					mockUI.EXPECT().Say("VM will not be provisioned because '-n' (no-provision) flag was specified."),
				)

				stoppedVM.Start(&vm.StartOpts{
					NoProvision: true,
				})
			})
		})

		Context("when starting the vm fails", func() {
			It("should return an error", func() {
				mockVBox.EXPECT().StartVM(stoppedVM.VMConfig).Return(errors.New("some-error"))
				allowHappyPathInteractions()

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when reading the private key fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))
				allowHappyPathInteractions()

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when SSHing provisions into the vm fails", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().RunSSHCommand("echo "+
					`'{"domain":"some-domain","ip":"some-ip","services":"rabbitmq,redis","registries":[],"provider":"some-provider"}' | sudo tee /var/pcfdev/provision-options.json >/dev/null`,
					addresses, []byte("some-private-key"), 5*time.Minute, os.Stdout, os.Stderr).Return(errors.New("some-error"))
				allowHappyPathInteractions()

				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when retrieving the unprovisioned vm fails", func() {
			It("should return an error", func() {
				mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("failed to start VM: some-error"))
			})
		})

		Context("when provisioning the unprovisioned vm fails", func() {
			It("should return an error", func() {
				mockUnprovisioned.EXPECT().Start(&vm.StartOpts{Provision: true}).Return(errors.New("some-error"))
				allowHappyPathInteractions()
				Expect(stoppedVM.Start(&vm.StartOpts{})).To(MatchError("some-error"))
			})
		})

	})

	Describe("Status", func() {
		It("should return 'Stopped'", func() {
			Expect(stoppedVM.Status()).To(Equal("Stopped"))
		})
	})

	Describe("Suspend", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped and cannot be suspended.")

			Expect(stoppedVM.Suspend()).To(Succeed())
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Only a suspended VM can be resumed.")

			Expect(stoppedVM.Resume()).To(Succeed())
		})
	})

	Describe("GetDebugLogs", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Start VM to retrieve debug logs.")
			Expect(stoppedVM.GetDebugLogs()).To(Succeed())
		})
	})

	Describe("Trust", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Start VM to trust VM certificates.")
			Expect(stoppedVM.Trust(&vm.StartOpts{})).To(Succeed())
		})
	})

	Describe("Target", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Start VM to target PCF Dev.")
			Expect(stoppedVM.Target(false)).To(Succeed())
		})
	})

	Describe("SSH", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("Your VM is currently stopped. Start VM to SSH to PCF Dev.")
			Expect(stoppedVM.SSH()).To(Succeed())
		})
	})
})
