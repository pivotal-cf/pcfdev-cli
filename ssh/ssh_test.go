package ssh_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"time"

	gossh "golang.org/x/crypto/ssh"
	"github.com/docker/docker/pkg/term"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/ssh/mocks"
	"github.com/pivotal-cf/pcfdev-cli/test_helpers"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ssh", func() {
	var (
		vBoxManagePath string
		vmName string
		ip string
		port string
		privateKeyBytes []byte
		mockCtrl        *gomock.Controller
		mockTerminal    *mocks.MockTerminal
		mockWindowsResizer *mocks.MockWindowResizer

		s *ssh.SSH
	)

	BeforeSuite(func() {
		var err error
		vBoxManagePath, err = helpers.VBoxManagePath()
		Expect(err).NotTo(HaveOccurred())

		privateKeyBytes, err = ioutil.ReadFile(filepath.Join("..", "assets", "insecure.key"))
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockTerminal = mocks.NewMockTerminal(mockCtrl)
		mockWindowsResizer = mocks.NewMockWindowResizer(mockCtrl)
		s = &ssh.SSH{
			Terminal: mockTerminal,
			WindowResizer: mockWindowsResizer,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("GenerateAddress", func() {
		It("Should return a host and free port", func() {
			host, port, err := s.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("127.0.0.1"))
			Expect(port).To(MatchRegexp("^[\\d]+$"))
		})
	})

	Describe("#RunSSHCommand", func() {
		Context("when SSH is available", func() {
			var (
				stdout *gbytes.Buffer
				stderr *gbytes.Buffer
			)

			BeforeEach(func() {
				var err error
				stdout = gbytes.NewBuffer()
				stderr = gbytes.NewBuffer()
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			Context("when the command succeeds", func() {
				It("should stream stdout to the terminal", func() {
					Expect(s.RunSSHCommand("echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute, stdout, stderr)).To(Succeed())
					Eventually(string(stdout.Contents()), 20 * time.Second).Should(Equal("some-output"))
				})

				It("should stream stderr to the terminal", func() {
					Expect(s.RunSSHCommand(">&2 echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute, stdout, stderr)).To(Succeed())
					Eventually(string(stderr.Contents()), 20 * time.Second).Should(Equal("some-output"))
				})
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					Expect(s.RunSSHCommand("false", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute, stdout, stderr)).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				Expect(s.RunSSHCommand("echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: "some-bad-port"}}, privateKeyBytes, time.Second, ioutil.Discard, ioutil.Discard)).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when private key is bad", func() {
			It("should return an error", func() {
				Expect(s.RunSSHCommand("false", []ssh.SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), 5 * time.Minute, ioutil.Discard, ioutil.Discard)).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#WaitForSSH", func() {
		Context("when SSH is available", func() {
			BeforeEach(func() {
				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should succeed", func() {
				Expect(s.WaitForSSH([]ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute)).To(Succeed())
			})

			Context("when a bad ssh address is passed in along with a good one", func() {
				It("should succeed", func() {
					Expect(s.WaitForSSH([]ssh.SSHAddress{{IP: ip, Port: port}, {IP: "some-bad-ip", Port: "some-port"}}, privateKeyBytes, 5 * time.Minute)).To(Succeed())
				})
			})
		})

		Context("when there is more than one ssh port to the VM", func() {
			BeforeEach(func() {
				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh2,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should succeed", func() {
				Expect(s.WaitForSSH([]ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute)).To(Succeed())
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				Expect(s.WaitForSSH([]ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Second)).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when private key is bad", func() {
			It("should return an error", func() {
				Expect(s.WaitForSSH([]ssh.SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), 5 * time.Second)).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#GetSSHOutput", func() {
		Context("when SSH is available", func() {
			BeforeEach(func() {
				var err error
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should return the output of the ssh command", func() {
				Expect(s.GetSSHOutput("echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute)).To(Equal("some-output"))
			})

			It("should return the stderr of the ssh command", func() {
				Expect(s.GetSSHOutput(">&2 echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute)).To(Equal("some-output"))
			})

			Context("when the command fails", func() {
				It("should return an error", func() {
					output, err := s.GetSSHOutput("echo -n some-output; false", []ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute)
					Expect(output).To(Equal("some-output"))
					Expect(err).To(MatchError(ContainSubstring("Process exited with: 1")))
				})
			})
		})

		Context("when SSH connection times out", func() {
			It("should return an error", func() {
				_, err := s.GetSSHOutput("echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: "some-bad-port"}}, privateKeyBytes, time.Second)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when private key is bad", func() {
			It("should return an error", func() {
				_, err := s.GetSSHOutput("echo -n some-output", []ssh.SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), time.Second)
				Expect(err).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#StartSSHSession", func() {
		Context("when SSH is available", func() {
			var (
				stdin  *gbytes.Buffer
				stdout *gbytes.Buffer
				stderr *gbytes.Buffer
			)

			BeforeEach(func() {
				var err error
				stdin = gbytes.NewBuffer()
				stdout = gbytes.NewBuffer()
				stderr = gbytes.NewBuffer()
				vmName, err = test_helpers.ImportSnappy()
				Expect(err).NotTo(HaveOccurred())

				ip, port, err = s.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				Expect(exec.Command(vBoxManagePath, "modifyvm", vmName, "--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%s,,22", port)).Run()).To(Succeed())
				Expect(exec.Command(vBoxManagePath, "startvm", vmName, "--type", "headless").Run()).To(Succeed())
			})

			AfterEach(func() {
				Expect(exec.Command(vBoxManagePath, "controlvm", vmName, "poweroff").Run()).To(Succeed())
				Eventually(func() error {
					return exec.Command(vBoxManagePath, "unregistervm", vmName, "--delete").Run()
				}, "10s").Should(Succeed())
			})

			It("should start an ssh session into the VM using a raw terminal", func() {
				stdinX, stdoutX, _ := term.StdStreams()
				stdinFd, _ := term.GetFdInfo(stdinX)
				stdoutFd, _ := term.GetFdInfo(stdoutX)

				go func() {
					time.Sleep(5 * time.Second)
					fmt.Fprintln(stdin, "exit")
				}()

				terminalState := &term.State{}

				mockTerminal.EXPECT().GetFdInfo(stdin).Return(stdinFd)
				mockTerminal.EXPECT().GetFdInfo(stdout).Return(stdoutFd)
				mockTerminal.EXPECT().SetRawTerminal(gomock.Any()).Return(terminalState, nil)
				mockWindowsResizer.EXPECT().StartResizing(gomock.Any())
				mockWindowsResizer.EXPECT().StopResizing()
				mockTerminal.EXPECT().RestoreTerminal(gomock.Any(), terminalState)

				err := s.StartSSHSession([]ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute, stdin, stdout, stderr)
				Expect(err).NotTo(HaveOccurred())

				Eventually(stdout).Should(gbytes.Say("Welcome to Ubuntu"))
			})

			Context("when there is an error making the terminal raw", func() {
				It("should return the error", func() {
					gomock.InOrder(
						mockTerminal.EXPECT().GetFdInfo(gomock.Any()),
						mockTerminal.EXPECT().GetFdInfo(gomock.Any()),
						mockTerminal.EXPECT().SetRawTerminal(gomock.Any()).Return(nil, errors.New("some-error")),
					)

					err := s.StartSSHSession([]ssh.SSHAddress{{IP: ip, Port: port}}, privateKeyBytes, 5 * time.Minute, gbytes.NewBuffer(), ioutil.Discard, ioutil.Discard)
					Expect(err).To(MatchError("some-error"))
				})
			})
		})

		Context("when there is an error creating the ssh session", func() {
			It("should return the error", func() {
				err := s.StartSSHSession([]ssh.SSHAddress{{IP: ip, Port: "some-bad-port"}}, privateKeyBytes, time.Second, gbytes.NewBuffer(), ioutil.Discard, ioutil.Discard)
				Expect(err).To(MatchError(ContainSubstring("ssh connection timed out:")))
			})
		})

		Context("when the private key is bad", func() {
			It("should return the error", func() {
				err := s.StartSSHSession([]ssh.SSHAddress{{IP: ip, Port: port}}, []byte("some-bad-private-key"), time.Second, gbytes.NewBuffer(), ioutil.Discard, ioutil.Discard)
				Expect(err).To(MatchError(ContainSubstring("could not parse private key:")))
			})
		})
	})

	Describe("#GenerateKeypair", func() {
		It("should generate an rsa keypair", func() {
			privateKey, publicKey, err := s.GenerateKeypair()
			Expect(err).NotTo(HaveOccurred())

			signer, err := gossh.ParsePrivateKey(privateKey)
			Expect(err).NotTo(HaveOccurred())

			Expect(gossh.MarshalAuthorizedKey(signer.PublicKey())).To(Equal(publicKey))
		})
	})
})
