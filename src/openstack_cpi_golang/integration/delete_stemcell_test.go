package integration_test

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenStack Integration", func() {
	BeforeEach(func() {
		SetupHTTP()

		Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
                "versions": {"values": [
                    {"status": "stable","id": "v3.0","links": [{ "href": "%s", "rel": "self" }]},
                    {"status": "stable","id": "v2.0","links": [{ "href": "%s", "rel": "self" }]}
                ]}
            }`, Endpoint()+"/v3", Endpoint()+"/v2.0")
		})

		Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Subject-Token", "0123456789")
			w.WriteHeader(http.StatusCreated)

			fmt.Fprintf(w, `{
                "token": {
                    "expires_at": "2013-02-02T18:30:59.000000Z",
                    "catalog": [{
                        "endpoints": [{"url": "%s","interface": "public","region": "RegionOne"}],
                        "type": "image",
                        "name": "glance"
                    }]
                }
            }`, Endpoint())
		})
	})

	AfterEach(func() {
		TeardownHTTP()
	})

	It("delete the stemcell image", func() {
		Mux.HandleFunc("/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			r.Method = "DELETE"
		})

		writeJsonParamToStdIn(`{
			"method":"delete_stemcell",
			"arguments":[
				 "b2173dd3-7ad6-4362-baa6-a68bce3565cb"
			]
		  }`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"error":null`))
	})
})
