package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
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
					},{
						"endpoints": [{"url": "%s/","interface": "public","region": "RegionOne"}],
						"type": "image",
						"name": "glance"
					},{
					   "endpoints": [
						 { "id": "1", "interface": "public",  "region": "RegionOne", "url": "%s/v2.0"},
						 { "id": "2", "interface": "admin",   "region": "RegionOne", "url": "%s/v2.0"},
						 { "id": "3", "interface": "internal","region": "RegionOne", "url": "%s/v2.0"}
					  ],
					  "type": "load-balancer",
					  "name": "octavia"
					}]
  				}
			}`, Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint(), Endpoint())
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

		Mux.HandleFunc("/v2/images/5bba0da5-dfb3-49d8-a005-d799507518f7", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"status": "active",
				"visibility": "private",
				"id": "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
				"file": "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
				"schema": "/v2/schemas/image"
			}`)
		})

		Mux.HandleFunc("/v2.0/security-groups/0c8a5d1a-8922-4d65-a0b2-dd78ab869e04", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"security_group": {
					"id": "85cc3048-abc3-43cc-89b3-377341426ac5"
				}
			}`)
		})

		Mux.HandleFunc("/v2.0/security-groups/bosh_acceptance_tests", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
		})

		Mux.HandleFunc("/v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"security_groups": [
					{
						"id": "191e4194-1b33-4886-8b2e-4a4e5de3f9ff",
						"name": "bosh_acceptance_tests"
					}
				]
			}`)
		})

		Mux.HandleFunc("/v2.1/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"flavors": [
					{
						"disk": 1,
						"id": "1",
						"name": "m1.tiny",
						"ram": 512
					}
				]
			}`)
		})

		Mux.HandleFunc("/v2.1/os-keypairs/default_key_name", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"keypair": {
					"name": "default_key_name",
					"id": 1
				}
			}`)
		})

		Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				fmt.Fprintf(w, `{
					"ports": [
					]
				}`)
			}

			if r.Method == "POST" {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)

				fmt.Fprintf(w, `{
					"port": {
						"device_id": "",
						"device_owner": "",
						"id": "65c0ee9f-d634-4522-8954-51021b570b0d",
						"status": "ACTIVE"
					}
				}`)
			}
		})

		Mux.HandleFunc("/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{
				"subnets": [
					{
						"cidr": "10.0.11.0/24",
						"id": "08eae331-0402-425a-923c-34f7cfe39c1b"
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
					"instance_type": "m1.tiny",
					"key_name": "default_key_name",
					"availability_zones": ["z1"]
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"availability_zone": "z1",
							"net_id": "fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3",
							"security_groups": [
								"0c8a5d1a-8922-4d65-a0b2-dd78ab869e04",
								"bosh_acceptance_tests"
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
		actual := <-outChannel
		Expect(actual).To(ContainSubstring(`"result":["f5dc173b-6804-445a-a6d8-c705dad5b5eb",{"bosh":{"type":"manual","ip":"10.0.11.16","netmask":"255.255.255.0","gateway":"10.0.11.1","dns":null,"default":["dns","gateway"],"routes":null,"cloud_properties":{"net_id":"fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3","security_groups":["0c8a5d1a-8922-4d65-a0b2-dd78ab869e04","bosh_acceptance_tests"]}}}],"error":null`))
	})

	It("fails if a wrong flavorName is given", func() {
		writeJsonParamToStdIn(`{
			"method": "create_vm",
			"arguments": [
				"a694d798-0b41-4255-9c8e-b282cd504a52",
				"5bba0da5-dfb3-49d8-a005-d799507518f7",
				{
					"instance_type": "wrong_flavor",
					"availability_zones": ["z1"]
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"availability_zone": "z1",
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
		Expect(<-outChannel).To(ContainSubstring("failed to resolve flavor of instance type 'wrong_flavor': flavor for instance type 'wrong_flavor' not found"))
	})

	It("fails if a security rule cannot be resolved", func() {
		writeJsonParamToStdIn(`{
			"method": "create_vm",
			"arguments": [
				"a694d798-0b41-4255-9c8e-b282cd504a52",
				"5bba0da5-dfb3-49d8-a005-d799507518f7",
				{
					"instance_type": "m1.tiny",
					"availability_zones": ["z1"]
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"availability_zone":"z1",
							"net_id": "fbe64fb7-b47c-4fd1-b158-9411d5c3ebf3",
							"security_groups": [
								"not-exiting-group"
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
		Expect(<-outChannel).To(ContainSubstring("failed to resolve security group: could not resolve security group 'not-exiting-group'"))
	})

	It("fails if a key pair name cannot be resolved", func() {
		Mux.HandleFunc("/v2.1/os-keypairs/unknown_key_name", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
		})

		writeJsonParamToStdIn(`{
			"method": "create_vm",
			"arguments": [
				"a694d798-0b41-4255-9c8e-b282cd504a52",
				"5bba0da5-dfb3-49d8-a005-d799507518f7",
				{
					"instance_type": "m1.tiny",
					"availability_zones": ["z1"]
				},
				{
					"bosh": {
						"type": "manual",
						"ip": "10.0.11.16",
						"netmask": "255.255.255.0",
						"cloud_properties": {
							"availability_zone":"z1",
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

		defaultConfig.Cloud.Properties.Openstack = config.OpenstackConfig{
			AuthURL:                 Endpoint(),
			Username:                "admin",
			APIKey:                  "admin",
			DomainName:              "domain",
			Tenant:                  "tenant",
			Region:                  "region",
			DefaultKeyName:          "unknown_key_name",
			StemcellPubliclyVisible: true,
		}

		err := cpi.Execute(defaultConfig, logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring("failed to resolve keypair: failed to retrieve 'unknown_key_name'"))
	})
})
