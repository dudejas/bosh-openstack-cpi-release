package integration

import (
	"bytes"
	"encoding/json"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os"
	"os/exec"
	"time"
)

func RunCli(cliPath string, configPath string, jsonInput string) (*gexec.Session, error) {
	command := exec.Command(cliPath, "-configFile", configPath)
	command.Stdin = bytes.NewBufferString(jsonInput)

	gexecSession, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return nil, err
	}
	gexecSession.Wait(1 * time.Minute)
	return gexecSession, nil
}

func MakeConfigFile(cfg *config.CpiConfig) string {
	cfgBytes, err := json.Marshal(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	tmpFile, err := os.CreateTemp("", "bosh-openstack-cli-test")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	_, err = tmpFile.Write(cfgBytes)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = tmpFile.Close()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return tmpFile.Name()
}
