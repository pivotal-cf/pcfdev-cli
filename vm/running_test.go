package vm_test

import (
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	conf "github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vm"
	"github.com/pivotal-cf/pcfdev-cli/vm/mocks"

	"github.com/docker/docker/pkg/term"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running", func() {
	var (
		mockCtrl       *gomock.Controller
		mockFS         *mocks.MockFS
		mockUI         *mocks.MockUI
		mockVBox       *mocks.MockVBox
		mockBuilder    *mocks.MockBuilder
		mockSSH        *mocks.MockSSH
		mockVM         *mocks.MockVM
		mockLogFetcher *mocks.MockLogFetcher
		mockCertStore  *mocks.MockCertStore
		mockCmdRunner  *mocks.MockCmdRunner

		runningVM vm.Running
		config    *conf.VMConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockUI = mocks.NewMockUI(mockCtrl)
		mockVBox = mocks.NewMockVBox(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		mockVM = mocks.NewMockVM(mockCtrl)
		mockBuilder = mocks.NewMockBuilder(mockCtrl)
		mockLogFetcher = mocks.NewMockLogFetcher(mockCtrl)
		mockCertStore = mocks.NewMockCertStore(mockCtrl)
		mockCmdRunner = mocks.NewMockCmdRunner(mockCtrl)
		config = &conf.VMConfig{}

		runningVM = vm.Running{
			VMConfig: &conf.VMConfig{
				Name:    "some-vm",
				Domain:  "some-domain",
				IP:      "some-ip",
				SSHPort: "some-port",
			},
			Config: &conf.Config{
				PrivateKeyPath: "some-private-key-path",
			},

			VBox:       mockVBox,
			FS:         mockFS,
			UI:         mockUI,
			Builder:    mockBuilder,
			SSHClient:  mockSSH,
			LogFetcher: mockLogFetcher,
			CertStore:  mockCertStore,
			CmdRunner:  mockCmdRunner,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Stop", func() {
		It("should stop the vm", func() {
			gomock.InOrder(
				mockUI.EXPECT().Say("Stopping VM..."),
				mockVBox.EXPECT().StopVM(runningVM.VMConfig),
				mockUI.EXPECT().Say("PCF Dev is now stopped."),
			)

			runningVM.Stop()
		})

		Context("when stopped the vm fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockUI.EXPECT().Say("Stopping VM..."),
					mockVBox.EXPECT().StopVM(runningVM.VMConfig).Return(errors.New("some-error")),
				)

				Expect(runningVM.Stop()).To(MatchError("failed to stop VM: some-error"))
			})
		})
	})

	Describe("VerifyStartOpts", func() {
		Context("when desired memory is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					Memory: 4000,
				})).To(MatchError("memory cannot be changed once the vm has been created"))
			})
		})

		Context("when cores is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					CPUs: 2,
				})).To(MatchError("cores cannot be changed once the vm has been created"))
			})
		})

		Context("when no opts are passed", func() {
			It("should succeed", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{})).To(Succeed())
			})
		})

		Context("when services are passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					Services: "redis",
				})).To(MatchError("services cannot be changed once the vm has been created"))
			})
		})

		Context("when desired IP is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					IP: "some-ip",
				})).To(MatchError("the -i flag cannot be used if the VM has already been created"))
			})
		})

		Context("when desired domain is passed", func() {
			It("should return an error", func() {
				Expect(runningVM.VerifyStartOpts(&vm.StartOpts{
					Domain: "some-domain",
				})).To(MatchError("the -d flag cannot be used if the VM has already been created"))
			})
		})
	})

	Describe("Start", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running.")

			runningVM.Start(&vm.StartOpts{})
		})
	})

	Describe("Start - with reprovision", func() {
		It("should provision the VM", func() {
			sshAddresses := []ssh.SSHAddress{
				{IP: "127.0.0.1", Port: "some-port"},
				{IP: "some-ip", Port: "22"},
			}

			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockSSH.EXPECT().GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("", nil),
				mockBuilder.EXPECT().VM("some-vm").Return(mockVM, nil),
				mockVM.EXPECT().Start(&vm.StartOpts{Provision: true}),
			)

			runningVM.Start(&vm.StartOpts{Provision: true})
		})

		Context("when removing healthcheck file fails", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("", errors.New("some-error")),
				)

				Expect(runningVM.Start(&vm.StartOpts{Provision: true})).To(MatchError("some-error"))
			})
		})

		Context("when retrieving the private key fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(runningVM.Start(&vm.StartOpts{Provision: true})).To(MatchError("some-error"))
			})
		})

		Context("when building unprovisioned vm fails", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("", nil),
					mockBuilder.EXPECT().VM("some-vm").Return(nil, errors.New("some-error")),
				)

				Expect(runningVM.Start(&vm.StartOpts{Provision: true})).To(MatchError("some-error"))
			})
		})

		Context("when running the provision command fails", func() {
			It("should return an error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("sudo rm -f /run/pcfdev-healthcheck", sshAddresses, []byte("some-private-key"), 30*time.Second).Return("", nil),
					mockBuilder.EXPECT().VM("some-vm").Return(mockVM, nil),
					mockVM.EXPECT().Start(&vm.StartOpts{Provision: true}).Return(errors.New("some-error")),
				)

				Expect(runningVM.Start(&vm.StartOpts{Provision: true})).To(MatchError("some-error"))
			})
		})
	})

	Describe("Status", func() {
		It("should return 'Running' with login instructions", func() {
			Expect(runningVM.Status()).To(Equal("Running\nCLI Login: cf login -a https://api.some-domain --skip-ssl-validation\nApps Manager URL: https://some-domain\nAdmin user => Email: admin / Password: admin\nRegular user => Email: user / Password: pass"))
		})
	})

	Describe("Suspend", func() {
		It("should suspend the vm", func() {
			mockUI.EXPECT().Say("Suspending VM...")
			mockVBox.EXPECT().SuspendVM(runningVM.VMConfig)
			mockUI.EXPECT().Say("PCF Dev is now suspended.")

			Expect(runningVM.Suspend()).To(Succeed())
		})

		Context("when suspending the vm fails", func() {
			It("should return an error", func() {
				mockUI.EXPECT().Say("Suspending VM...")
				mockVBox.EXPECT().SuspendVM(runningVM.VMConfig).Return(errors.New("some-error"))

				Expect(runningVM.Suspend()).To(MatchError("failed to suspend VM: some-error"))
			})
		})
	})

	Describe("Resume", func() {
		It("should say a message", func() {
			mockUI.EXPECT().Say("PCF Dev is running.")

			Expect(runningVM.Resume()).To(Succeed())
		})
	})

	Describe("GetDebugLogs", func() {
		It("should succeed", func() {
			gomock.InOrder(
				mockLogFetcher.EXPECT().FetchLogs(),
				mockUI.EXPECT().Say("Debug logs written to pcfdev-debug.tgz. While some scrubbing has taken place, please remove any remaining sensitive information from these logs before sharing."),
			)

			Expect(runningVM.GetDebugLogs()).To(Succeed())
		})

		Context("when fetching logs fails", func() {
			It("should return the error", func() {
				mockLogFetcher.EXPECT().FetchLogs().Return(errors.New("some-error"))

				Expect(runningVM.GetDebugLogs()).To(MatchError("failed to retrieve logs: some-error"))
			})
		})
	})

	Describe("Trust", func() {
		It("should trust VM certificates", func() {
			sshAddresses := []ssh.SSHAddress{
				{IP: "127.0.0.1", Port: "some-port"},
				{IP: "some-ip", Port: "22"},
			}
			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", sshAddresses, []byte("some-private-key"), 5*time.Minute).Return("some-cert", nil),
				mockCertStore.EXPECT().Store("some-cert"),
				mockUI.EXPECT().Say("***Warning: a self-signed certificate for *.some-domain has been inserted into your OS certificate store. To remove this certificate, run: cf dev untrust***"),
			)

			Expect(runningVM.Trust(&vm.StartOpts{})).To(Succeed())
		})

		Context("when there is an error reading the private key", func() {
			It("should return the error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(runningVM.Trust(&vm.StartOpts{})).To(MatchError("failed to trust VM certificates: some-error"))
			})
		})

		Context("when there is an error getting SSH output", func() {
			It("should return the error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", sshAddresses, []byte("some-private-key"), 5*time.Minute).Return("", errors.New("some-error")),
				)

				Expect(runningVM.Trust(&vm.StartOpts{})).To(MatchError("failed to trust VM certificates: some-error"))
			})
		})

		Context("when there is an error storing the certificate", func() {
			It("should return the error", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", sshAddresses, []byte("some-private-key"), 5*time.Minute).Return("some-cert", nil),
					mockCertStore.EXPECT().Store("some-cert").Return(errors.New("some-error")),
				)

				Expect(runningVM.Trust(&vm.StartOpts{})).To(MatchError("failed to trust VM certificates: some-error"))
			})
		})

		Context("when the user specifies the 'PrintCA' flag", func() {
			It("should print the CA", func() {
				sshAddresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().GetSSHOutput("cat /var/pcfdev/openssl/ca_cert.pem", sshAddresses, []byte("some-private-key"), 5*time.Minute).Return("some-cert", nil),
					mockUI.EXPECT().Say("some-cert"),
				)

				Expect(runningVM.Trust(&vm.StartOpts{PrintCA: true})).To(Succeed())
			})
		})
	})

	Describe("SSH", func() {
		It("should execute ssh on the client", func() {
			addresses := []ssh.SSHAddress{
				{IP: "127.0.0.1", Port: "some-port"},
				{IP: "some-ip", Port: "22"},
			}
			stdin, stdout, stderr := term.StdStreams()

			gomock.InOrder(
				mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
				mockSSH.EXPECT().StartSSHSession(addresses, []byte("some-private-key"), 5*time.Minute, stdin, stdout, stderr),
			)

			Expect(runningVM.SSH()).To(Succeed())
		})

		Context("when executing ssh fails", func() {
			It("should return an error", func() {
				addresses := []ssh.SSHAddress{
					{IP: "127.0.0.1", Port: "some-port"},
					{IP: "some-ip", Port: "22"},
				}
				stdin, stdout, stderr := term.StdStreams()

				gomock.InOrder(
					mockFS.EXPECT().Read("some-private-key-path").Return([]byte("some-private-key"), nil),
					mockSSH.EXPECT().StartSSHSession(addresses, []byte("some-private-key"), 5*time.Minute, stdin, stdout, stderr).Return(errors.New("some-error")),
				)

				Expect(runningVM.SSH()).To(MatchError("some-error"))
			})
		})

		Context("when reading the private key fails", func() {
			It("should return an error", func() {
				mockFS.EXPECT().Read("some-private-key-path").Return(nil, errors.New("some-error"))

				Expect(runningVM.SSH()).To(MatchError("some-error"))
			})
		})
	})

	Describe("Target", func() {
		Context("when autoTarget is set", func() {
			It("target PCF Dev", func() {
				mockCmdRunner.EXPECT().Run(
					"cf",
					"login",
					"-a", "api.some-domain",
					"--skip-ssl-validation",
					"-u", "user",
					"-p", "pass",
					"-o", "pcfdev-org",
					"-s", "pcfdev-space",
				)
				Expect(runningVM.Target(true)).To(Succeed())
			})
		})
	})

	Context("when autoTarget is NOT set", func() {
		It("target PCF Dev and prints an output message to the user", func() {
			mockCmdRunner.EXPECT().Run(
				"cf",
				"login",
				"-a", "api.some-domain",
				"--skip-ssl-validation",
				"-u", "user",
				"-p", "pass",
				"-o", "pcfdev-org",
				"-s", "pcfdev-space",
			)
			mockUI.EXPECT().Say("Successfully logged in to api.some-domain as user.")

			Expect(runningVM.Target(false)).To(Succeed())
		})
	})

	Context("when there is an error", func() {
		It("should return the error", func() {
			mockCmdRunner.EXPECT().Run(
				"cf",
				"login",
				"-a", "api.some-domain",
				"--skip-ssl-validation",
				"-u", "user",
				"-p", "pass",
				"-o", "pcfdev-org",
				"-s", "pcfdev-space",
			).Return(nil, errors.New("some-error"))

			Expect(runningVM.Target(false)).To(MatchError("failed to target PCF Dev: some-error"))
		})
	})
})
