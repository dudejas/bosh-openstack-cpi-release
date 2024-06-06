package integration_test

import (
	"bytes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var defaultConfig config.CpiConfig
var logger = utils.NewLogger(boshlog.NewWriterLogger(boshlog.LevelDebug, os.Stderr))
var Mux *http.ServeMux
var Server *httptest.Server
var originalStdin *os.File
var originalStdout *os.File
var outChannel chan string
var stdOutWriter *os.File

var _ = BeforeEach(func() {
	originalStdin = os.Stdin
	originalStdout = os.Stdout

	outChannel, stdOutWriter = setupReadableStdOut()
})

var _ = AfterEach(func() {
	os.Stdin = originalStdin
	os.Stdout = originalStdout
})

func getDefaultConfig(url string) config.CpiConfig {
	defaultConfig.Cloud.Properties.Openstack = config.OpenstackConfig{
		AuthURL:                 url,
		Username:                "admin",
		APIKey:                  "admin",
		DomainName:              "domain",
		Tenant:                  "tenant",
		Region:                  "region",
		DefaultKeyName:          "default_key_name",
		StemcellPubliclyVisible: true,
	}

	return defaultConfig
}

func SetupHTTP() {
	Mux = http.NewServeMux()

	loggingMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Mux.ServeHTTP(w, r)
		log.Printf("Received request: %s %s\n", r.Method, r.URL.String())
	})

	Server = httptest.NewServer(loggingMux)
}

func Endpoint() string {
	return Server.URL
}

func TeardownHTTP() {
	Server.Close()
}

func writeJsonParamToStdIn(json string) {
	reader, writer, _ := os.Pipe()
	os.Stdin = reader

	go func() {
		writer.WriteString(json)
		writer.Close()
	}()
}

func setupReadableStdOut() (chan string, *os.File) {
	reader, writer, _ := os.Pipe()
	os.Stdout = writer
	outChannel := make(chan string)
	// copy the output in a separate goroutine so reading from the pipe doesn't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, reader)
		outChannel <- buf.String()
	}()

	return outChannel, writer
}
