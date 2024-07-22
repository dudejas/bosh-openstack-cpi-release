package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Calculate VM cloud properties", func() {

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
						"endpoints": [
							{"id": "1", "interface": "public", "region": "RegionOne", "url": "%s/v2.1"},
							{"id": "2", "interface": "admin", "region": "RegionOne", "url": "%s/v2.1"},
							{"id": "3", "interface": "internal", "region": "RegionOne", "url": "%s/v2.1"}
						],
						"type": "compute", 
						"name": "nova"
					}]
  				}
			}`, Endpoint(), Endpoint(), Endpoint())
		})

		Mux.HandleFunc("/v2.1/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"flavors": [
					{
						"disk": 1,
						"id": "1",
						"vcpus": 1,
						"name": "m1.tiny",
						"ram": 512
					},
					{
						"disk": 64,
						"id": "138",
						"vcpus": 2,
						"name": "m_c2_m16",
						"ram": 16368
					},
					{
						"disk": 64,
						"id": "50",
						"vcpus": 4,
						"name": "g_c4_m16",
						"ram": 16368
					}
				]
			}`)
		})
	})

	AfterEach(func() {
		TeardownHTTP()
	})

	It("calculate vm cloud properties without bootFromVolume", func() {
		writeJsonParamToStdIn(`{
			"method": "calculate_vm_cloud_properties",
			"arguments": [
				{
					"cpu": 1,
					"ram": 8192, 
					"ephemeral_disk_size": 4096
				}	
			],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		actual := <-outChannel
		Expect(actual).To(ContainSubstring(`"result":{"instance_type":"m_c2_m16"},"error":null`))
	})

	It("calculate vm cloud properties with bootFromVolume", func() {
		writeJsonParamToStdIn(`{
			"method": "calculate_vm_cloud_properties",
			"arguments": [
				{
					"cpu": 1,
					"ram": 8192, 
					"ephemeral_disk_size": 4096
				}	
			],
			"api_version": 2
		}`)

		err := cpi.Execute(getBootFromVolumeConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		actual := <-outChannel
		Expect(actual).To(ContainSubstring(`"result":{"instance_type":"m_c2_m16","root_disk":{"size":4099}},"error":null`))
	})

	It("fails if no flavor can fulfill the requirement", func() {
		writeJsonParamToStdIn(`{
			"method": "calculate_vm_cloud_properties",
			"arguments": [
				{
					"cpu": 10,
					"ram": 8192, 
					"ephemeral_disk_size": 4096
				}	
			],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring("Unable to meet requested VM requirements: 10 CPU, 8192 MB RAM, 4 GB Disk."))
	})
})
