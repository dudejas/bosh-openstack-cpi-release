package integration_test

import (
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/onsi/gomega/gexec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var cliPath string
var defaultConfig config.CpiConfig

var _ = BeforeSuite(func() {
	if len(cliPath) == 0 {
		var err error
		cliPath, err = gexec.Build("github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang")
		Expect(err).ShouldNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
