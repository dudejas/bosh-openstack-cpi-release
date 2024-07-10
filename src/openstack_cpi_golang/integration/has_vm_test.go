package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("HAS VM", func() {
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

		Mux.HandleFunc("/v2.1/servers/active-server-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{
					"server": {
						"id": "active",
						"status": "ACTIVE"
					}
				}`)
			}
		})

		Mux.HandleFunc("/v2.1/servers/deleted-server-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{
					"server": {
						"id": "deleted",
						"status": "DELETED"
					}
				}`)
			}
		})

		Mux.HandleFunc("/v2.1/servers/terminated-server-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{
					"server": {
						"id": "terminated",
						"status": "TERMINATED"
					}
				}`)
			}
		})

		Mux.HandleFunc("/v2.1/servers/wrong-vm-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{}`)
			}
		})

		Mux.HandleFunc("/v2.1/servers/error-vm-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{}`)
			}
		})

	})

	AfterEach(func() {
		TeardownHTTP()
	})

	It("returns true if the server exists and is ACTIVE", func() {
		writeJsonParamToStdIn(`{
				"method":"has_vm",
				"arguments": ["active-server-id"],
				"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":true,"error":null`))
	})

	It("returns false if the server exists and is DELETED", func() {
		writeJsonParamToStdIn(`{
				"method":"has_vm",
				"arguments": ["deleted-server-id"],
				"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":false,"error":null`))
	})

	It("returns false if the server exists and is TERMINATED", func() {
		writeJsonParamToStdIn(`{
				"method":"has_vm",
				"arguments": ["terminated-server-id"],
				"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":false,"error":null`))
	})

	It("returns false if the server does not exist", func() {
		writeJsonParamToStdIn(`{
				"method":"has_vm",
				"arguments": ["wrong-vm-id"],
				"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":false,"error":null`))
	})

	It("returns false and raises an error if server retrieval fails", func() {
		writeJsonParamToStdIn(`{
				"method":"has_vm",
				"arguments": ["error-vm-id"],
				"api_version": 2
		}`)

		cpiConfig := getDefaultConfig(Endpoint())
		cpiConfig.Cloud.Properties.RetryConfig = config.RetryConfigMap{
			"default": config.RetryConfig{
				MaxAttempts:   10,
				SleepDuration: 0,
			},
		}

		err := cpi.Execute(cpiConfig, logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`message":"has_vm: failed to retrieve server information: max retry attempts (10) reached, err: Internal Server Error`))
	})

})
