package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing/fstest"
)

var _ = Describe("OpenstackConfig", func() {
	var fileSystem fstest.MapFS

	BeforeEach(func() {
		fileSystem = fstest.MapFS{
			"some/path/config.json": &fstest.MapFile{
				Data: []byte(`{
						"cloud": {
							"properties": {
								"openstack": {
									"auth_url": "the_auth_url",
									"username": "the_username",
									"api_key": "the_api_key",
									"domain": "the_domain",
									"tenant": "the_tenant",
									"region": "the_region",
									"default_key_name": "the_default_key_name",
									"stemcell_public_visibility": true
								}
							}
						}
					}`),
			},
			"some/path/config.txt": &fstest.MapFile{
				Data: []byte(`not a json file`),
			},
		}
	})

	Context("Validate", func() {
		It("gets the cpi configuration from filesystem", func() {
			cpiConfig, err := NewConfigFromPath(fileSystem, "some/path/config.json")

			Expect(err).ToNot(HaveOccurred())
			Expect(cpiConfig.Cloud.Properties.Openstack.AuthURL).To(Equal("the_auth_url"))
			Expect(cpiConfig.Cloud.Properties.Openstack.Username).To(Equal("the_username"))
			Expect(cpiConfig.Cloud.Properties.Openstack.APIKey).To(Equal("the_api_key"))
			Expect(cpiConfig.Cloud.Properties.Openstack.DomainName).To(Equal("the_domain"))
			Expect(cpiConfig.Cloud.Properties.Openstack.Tenant).To(Equal("the_tenant"))
			Expect(cpiConfig.Cloud.Properties.Openstack.Region).To(Equal("the_region"))
			Expect(cpiConfig.Cloud.Properties.Openstack.DefaultKeyName).To(Equal("the_default_key_name"))
			Expect(cpiConfig.Cloud.Properties.Openstack.StemcellPubliclyVisible).To(BeTrue())
		})

		It("returns an error if config file cannot be found", func() {
			_, err := NewConfigFromPath(fileSystem, "some/path/not_existing_config.json")

			Expect(err.Error()).To(ContainSubstring("failed to open configuration file: open some/path/not_existing_config.json: file does not exist"))
		})

		It("returns an error if config file cannot be found", func() {
			_, err := NewConfigFromPath(fileSystem, "some/path")

			Expect(err.Error()).To(Equal("failed to read configuration file: read some/path: invalid argument"))
		})

		It("returns an error if config file json cannot be unmarshalled", func() {
			_, err := NewConfigFromPath(fileSystem, "some/path/config.txt")

			Expect(err.Error()).To(ContainSubstring("failed to unmarshall configuration file: some/path/config.txt, err: invalid character"))
		})

	})
})
