package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Delete VM", func() {
	var getServerCount = 0

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

		Mux.HandleFunc("/v2.0/ports/1", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprintf(w, `{}`)
			}
		})

		Mux.HandleFunc("/v2.0/ports/wrong-port-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{}`)
			}
		})

		Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprintf(w, `{}`)

			case http.MethodGet:
				deviceID := r.URL.Query().Get("device_id")
				if deviceID != "1" && deviceID != "2" && deviceID != "wrong-vm-id" {
					return
				}

				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				if deviceID == "wrong-vm-id" {
					fmt.Fprintf(w, `{"ports": []}`)
					return
				} else if deviceID == "1" {
					fmt.Fprintf(w, `{
						"ports": [
							{
								"device_id": "1",
								"id": "1"
							}
						]
					}`)
				} else if deviceID == "2" {
					fmt.Fprintf(w, `{
						"ports": [
							{
								"device_id": "2",
								"id": "1"
							},
							{
								"device_id": "2",
								"id": "wrong-port-id"
							}
						]
					}`)
				}
			}
		})

		Mux.HandleFunc("/v2.1/servers/1", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprintf(w, `{}`)

			case http.MethodGet:
				getServerCount++
				switchCase := getServerCount % 2

				w.WriteHeader(http.StatusOK)

				switch switchCase {
				case 1:
					fmt.Fprintf(w, `{
						"server": {
							"id": "1",
							"status": "ACTIVE"
						}
					}`)
				case 0:
					fmt.Fprintf(w, `{
						"server": {
							"id": "1",
							"status": "DELETED"
						}
					}`)
				}
			}
		})

		Mux.HandleFunc("/v2.1/servers/2", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprintf(w, `{}`)

			case http.MethodGet:
				getServerCount++
				switchCase := getServerCount % 2

				w.WriteHeader(http.StatusOK)

				switch switchCase {
				case 1:
					fmt.Fprintf(w, `{
						"server": {
							"id": "2",
							"status": "ACTIVE"
						}
					}`)
				case 0:
					fmt.Fprintf(w, `{
						"server": {
							"id": "2",
							"status": "DELETED"
						}
					}`)
				}
			}
		})

		Mux.HandleFunc("/v2.1/servers/wrong-vm-id", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{}`)
			}
		})

	})

	AfterEach(func() {
		TeardownHTTP()
	})

	It("deletes a vm", func() {
		writeJsonParamToStdIn(`{
			"method": "delete_vm",
			"arguments": ["1"],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":null,"error":null`))
	})

	It("does not fail when deleting not-existing vm", func() {
		writeJsonParamToStdIn(`{
			"method": "delete_vm",
			"arguments": ["wrong-vm-id"],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":null,"error":null`))
	})

	It("does not fail when deleting a vm with not-existing port", func() {
		writeJsonParamToStdIn(`{
			"method": "delete_vm",
			"arguments": ["2"],
			"api_version": 2
		}`)

		err := cpi.Execute(getDefaultConfig(Endpoint()), logger)
		Expect(err).ShouldNot(HaveOccurred())

		stdOutWriter.Close()
		Expect(<-outChannel).To(ContainSubstring(`"result":null,"error":null`))
	})
})
