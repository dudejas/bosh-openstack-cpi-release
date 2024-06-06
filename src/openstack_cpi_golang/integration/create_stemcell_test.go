package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"sync/atomic"
)

var _ = Describe("Create Stemcell", func() {
	var count int64

	BeforeEach(func() {
		SetupHTTP()

		Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
				"versions": {"values": [
					{"status": "stable","id": "v3.0","links": [{ "href": "%s/v3", "rel": "self" }]},
					{"status": "stable","id": "v2.0","links": [{ "href": "%s/v2.0", "rel": "self" }]}
				]}
			}`, Endpoint(), Endpoint())
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

	It("create a heavy stemcell image", func() {
		Mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)

			fmt.Fprintf(w, `{
				"status": "queued",
				"visibility": "private",
				"id": "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
				"file": "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
				"schema": "/v2/schemas/image"
			}`)
		})

		Mux.HandleFunc("/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		writeJsonParamToStdIn(`{
			"method":"create_stemcell",
			"arguments":[
				"./testdata/image",
				{
					"disk":5120,"disk_format":
					"vmdk","container_format":"bare",
					"architecture":"x86_64",
					"vmware_ostype":"ubuntu64Guest"
				}
			]
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":"b2173dd3-7ad6-4362-baa6-a68bce3565cb"`))
	})

	It("create a light stemcell image", func() {
		Mux.HandleFunc("/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"status": "active",
				"visibility": "private",
				"id": "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
				"file": "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
				"schema": "/v2/schemas/image"
			}`)
		})

		writeJsonParamToStdIn(`{
			"method":"create_stemcell",
			"arguments":[
				"./testdata/image",
				{
					"image_id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb",
					"disk":5120,"disk_format":
					"vmdk","container_format":"bare",
					"architecture":"x86_64",
					"vmware_ostype":"ubuntu64Guest"
				}
			]
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":"b2173dd3-7ad6-4362-baa6-a68bce3565cb"`))
	})

	It("retries the light stemcell creation", func() {
		services.DefaultRetrySleepDuration = 0
		Mux.HandleFunc("/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb", func(w http.ResponseWriter, r *http.Request) {

			if atomic.LoadInt64(&count) == 0 {
				// fail on first request
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{}`)

				atomic.AddInt64(&count, 1)
			} else {
				// succeed on second request
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{
					"status": "active",
					"visibility": "private",
					"id": "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
					"file": "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
					"schema": "/v2/schemas/image"
				}`)
			}
		})

		writeJsonParamToStdIn(`{
			"method":"create_stemcell",
			"arguments":[
				"./testdata/image",
				{
					"image_id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb",
					"disk":5120,"disk_format":
					"vmdk","container_format":"bare",
					"architecture":"x86_64",
					"vmware_ostype":"ubuntu64Guest"
				}
			]
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":"b2173dd3-7ad6-4362-baa6-a68bce3565cb"`))
	})
})
