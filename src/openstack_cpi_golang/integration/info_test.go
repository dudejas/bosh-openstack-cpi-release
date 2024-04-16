package integration_test

import (
	"bytes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("Testing the 'info' CPI method", func() {
	var configPath string

	BeforeEach(func() {
		configPath = integration.MakeConfigFile(&defaultConfig)
	})

	AfterEach(func() {
		defer func() { _ = os.Remove(configPath) }()
	})

	Describe("Invoking `info`", func() {
		It("receives expected response", func() {

			defer func() {
				cliSession, err := integration.RunCli(cliPath, configPath,
					`{"method":"info","arguments":[],"context":{"director_uuid":"uuid","request_id":"cpi-id","vm":{"stemcell":{"api_version":3}}}}`,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(cliSession.ExitCode()).To(BeZero())

				consoleOutput := bytes.NewBuffer(cliSession.Out.Contents()).String()
				Expect(consoleOutput).To(ContainSubstring(
					`{"result":{"api_version":2,"stemcell_formats":["openstack-raw","openstack-qcow2","openstack-light"]},"error":null,"log":""}`,
				))
			}()

		})
	})

})
