package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Create VM", func() {

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
					},{
						"endpoints": [
							{"id": "1", "interface": "public", "region": "RegionOne", "url": "%s/"},
							{"id": "2", "interface": "admin", "region": "RegionOne", "url": "%s/"},
							{"id": "3", "interface": "internal", "region": "RegionOne", "url": "%s/"}
						],
						"type": "network", 
						"name": "neutron"
					}]
  				}
			}`, Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint())
		})

		Mux.HandleFunc("/v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)

			fmt.Fprintf(w, `{
				"server": {
					"id": "f5dc173b-6804-445a-a6d8-c705dad5b5eb"
				}
			}`)
		})

		Mux.HandleFunc("/v2.1/servers/f5dc173b-6804-445a-a6d8-c705dad5b5eb", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"server": {
					"id": "f5dc173b-6804-445a-a6d8-c705dad5b5eb",
					"status": "ACTIVE"
				}
			}`)
		})

		Mux.HandleFunc("/v2.1/flavors/detail", func(w http.ResponseWriter, r *http.Request) {

			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"flavors": [
					{
						"id": "1",
						"name": "m1.tiny"
					}
				]
			}`)
		})
	})

	AfterEach(func() {
		TeardownHTTP()
	})

	It("create a vm", func() {
		writeJsonParamToStdIn(`{
			"method": "create_vm",
			"arguments": [
				"a694d798-0b41-4255-9c8e-b282cd504a52",
				"5bba0da5-dfb3-49d8-a005-d799507518f7",
				{
					"instance_type": "m1.tiny"
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"net_id": "fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3",
							"security_groups": [
								"0c8a5d1a-8922-4d65-a0b2-dd78ab869e04"
							]
						},
						"default": [
							"dns",
							"gateway"
						],
						"gateway": "10.0.11.1"
					}
				},
				[],
				{}
			],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":["f5dc173b-6804-445a-a6d8-c705dad5b5eb",{"bosh":{"type":"manual","ip":"10.0.11.16","netmask":"255.255.255.0","gateway":"10.0.11.1","dns":null,"default":["dns","gateway"],"routes":null,"cloud_properties":{"net_id":"fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3","security_groups":["0c8a5d1a-8922-4d65-a0b2-dd78ab869e04"]}}}],"error":null`))
	})

	It("fails if a wrong flavorName is given", func() {
		writeJsonParamToStdIn(`{
			"method": "create_vm",
			"arguments": [
				"a694d798-0b41-4255-9c8e-b282cd504a52",
				"5bba0da5-dfb3-49d8-a005-d799507518f7",
				{
					"instance_type": "wrong_flavor"
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"net_id": "fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3",
							"security_groups": [
								"0c8a5d1a-8922-4d65-a0b2-dd78ab869e04"
							]
						},
						"default": [
							"dns",
							"gateway"
						],
						"gateway": "10.0.11.1"
					}
				},
				[],
				{}
			],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring("failed to get flavor of instance type: flavor 'wrong_flavor' not found"))
	})
})
