package openstack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestMethods(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Openstack Suite")
}
