package vm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVBox(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PCF Dev VM Suite")
}
