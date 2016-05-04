package vbox_test

import (
	"errors"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/vbox"
	"github.com/pivotal-cf/pcfdev-cli/vbox/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("vbox", func() {
	var (
		mockCtrl   *gomock.Controller
		mockDriver *mocks.MockDriver
		mockSSH    *mocks.MockSSH
		vbx        *vbox.VBox
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDriver = mocks.NewMockDriver(mockCtrl)
		mockSSH = mocks.NewMockSSH(mockCtrl)
		vbx = &vbox.VBox{
			Driver: mockDriver,
			SSH:    mockSSH,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("ImportVM", func() {
		It("should import the VM", func() {
			gomock.InOrder(
				mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
				mockDriver.EXPECT().VBoxManage("import", "some-path"),
				mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
				mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
				mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22"),
			)
			err := vbx.ImportVM("some-path", "some-vm")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("fail to acquire random host port", func() {
			It("should return an error", func() {
				mockSSH.EXPECT().GenerateAddress().Return("", "", errors.New("some-error"))

				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("VM fails to import", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path").Return(nil, errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("Creation of host only interface fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("", errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("fails to attach interface", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})

		Context("Port fowarding fails", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockSSH.EXPECT().GenerateAddress().Return("some-host", "some-port", nil),
					mockDriver.EXPECT().VBoxManage("import", "some-path"),
					mockDriver.EXPECT().CreateHostOnlyInterface("192.168.11.1").Return("vboxnet1", nil),
					mockDriver.EXPECT().AttachNetworkInterface("vboxnet1", "some-vm"),
					mockDriver.EXPECT().ForwardPort("some-vm", "ssh", "some-port", "22").Return(errors.New("some-error")),
				)
				err := vbx.ImportVM("some-path", "some-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("StartVM", func() {
		Context("VM is already imported", func() {
			It("starts without reimporting", func() {
				gomock.InOrder(
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
					mockDriver.EXPECT().StartVM("some-vm"),
					mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute),
					mockDriver.EXPECT().StopVM("some-vm"),
					mockDriver.EXPECT().StartVM("some-vm"),
				)
				vm, err := vbx.StartVM("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Name).To(Equal("some-vm"))
				Expect(vm.SSHPort).To(Equal("some-port"))
			})

			Context("fails so get forward port", func() {
				It("should return an error", func() {
					mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("", errors.New("some-error"))

					vm, err := vbx.StartVM("some-vm")
					Expect(vm).To(BeNil())
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("VM fails to start", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().StartVM("some-vm").Return(errors.New("some-error")),
					)

					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("SSH Command to set static ip fails", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute).Return(nil, errors.New("some-error")),
					)
					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})

			Context("VM fails to stop", func() {
				It("should return an error", func() {
					gomock.InOrder(
						mockDriver.EXPECT().GetHostForwardPort("some-vm", "ssh").Return("some-port", nil),
						mockDriver.EXPECT().StartVM("some-vm"),
						mockSSH.EXPECT().RunSSHCommand("echo -e \"auto eth1\niface eth1 inet static\naddress 192.168.11.11\nnetmask 255.255.255.0\" | sudo tee -a /etc/network/interfaces", "some-port", 2*time.Minute),
						mockDriver.EXPECT().StopVM("some-vm").Return(errors.New("some-error")),
					)
					_, err := vbx.StartVM("some-vm")
					Expect(err).To(MatchError("some-error"))
				})
			})
		})
	})

	Describe("StopVM", func() {
		It("should stop the VM", func() {
			mockDriver.EXPECT().StopVM("some-vm")

			err := vbx.StopVM("some-vm")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Driver fails to stop VM", func() {
			It("should return the error", func() {
				expectedError := errors.New("some-error")

				mockDriver.EXPECT().StopVM("some-vm").Return(expectedError)
				err := vbx.StopVM("some-vm")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("ConflictingVMPresent", func() {
		Context("when there are no conflicting VMs with the prefix pcfdev-", func() {
			It("should return false", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"some-other-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent("pcfdev-our-vm")).To(BeFalse())
			})
		})

		Context("when there are conflicting VMs with the prefix pcfdev- running", func() {
			It("should return true", func() {
				mockDriver.EXPECT().RunningVMs().Return([]string{"pcfdev-conflicting-vm", "pcfdev-our-vm"}, nil)
				Expect(vbx.ConflictingVMPresent("pcfdev-our-vm")).To(BeTrue())
			})
		})

		Context("when getting running vms returns an error", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().RunningVMs().Return(nil, errors.New("some-error"))
				_, err := vbx.ConflictingVMPresent("pcfdev-our-vm")
				Expect(err).To(MatchError("some-error"))
			})
		})
	})

	Describe("DestroyVMs", func() {
		Context("when the VM is stopped", func() {
			It("should destroy the VM", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)
				mockDriver.EXPECT().DestroyVM("some-vm")

				Expect(vbx.DestroyVMs([]string{"some-vm"})).To(Succeed())
			})
		})

		Context("when the VM is running", func() {
			It("should power off and destroy the VM", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)
				mockDriver.EXPECT().PowerOffVM("some-vm")
				mockDriver.EXPECT().DestroyVM("some-vm")

				Expect(vbx.DestroyVMs([]string{"some-vm"})).To(Succeed())
			})
		})

		Context("when the driver fails to stop VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)
				mockDriver.EXPECT().PowerOffVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.DestroyVMs([]string{"some-vm"})).To(MatchError("some-error"))
			})
		})

		Context("when the driver fails to destroy VM", func() {
			It("should return the error", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)
				mockDriver.EXPECT().PowerOffVM("some-vm")
				mockDriver.EXPECT().DestroyVM("some-vm").Return(errors.New("some-error"))

				Expect(vbx.DestroyVMs([]string{"some-vm"})).To(MatchError("some-error"))
			})
		})

		Context("when multiple VMs are passed", func() {
			It("should destroy all of the VMs", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)
				mockDriver.EXPECT().DestroyVM("some-vm")
				mockDriver.EXPECT().VMExists("some-other-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-other-vm").Return(false)
				mockDriver.EXPECT().DestroyVM("some-other-vm")

				Expect(vbx.DestroyVMs([]string{"some-vm", "some-other-vm"})).To(Succeed())
			})
		})
	})

	Describe("#Status", func() {
		Context("VM is running", func() {
			It("should return running", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(true)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Running"))
			})
		})

		Context("VM is not created", func() {
			It("should return not created", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(false, nil)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Not created"))
			})
		})

		Context("VM is stopped", func() {
			It("should return stopped", func() {
				mockDriver.EXPECT().VMExists("some-vm").Return(true, nil)
				mockDriver.EXPECT().IsVMRunning("some-vm").Return(false)

				status, err := vbx.Status("some-vm")
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal("Stopped"))
			})
		})

		Context("An error checking the VM Status", func() {
			It("should return an error", func() {
				someError := errors.New("some-error")
				mockDriver.EXPECT().VMExists("some-vm").Return(false, someError)

				_, err := vbx.Status("some-vm")
				Expect(err).To(MatchError(someError))
			})
		})
	})

	Describe("#GetPCFDevVMs", func() {
		It("should return VM names that begin with pcfdev-", func() {
			mockDriver.EXPECT().VMs().Return([]string{"pcfdev-0.0.0", "pcfdev-0.0.1", "some-bad-vm-name"}, nil)

			Expect(vbx.GetPCFDevVMs()).To(Equal([]string{"pcfdev-0.0.0", "pcfdev-0.0.1"}))
		})

		Context("when getting VMs fails", func() {
			It("should return an error", func() {
				mockDriver.EXPECT().VMs().Return([]string{}, errors.New("some-error"))

				_, err := vbx.GetPCFDevVMs()
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
})
