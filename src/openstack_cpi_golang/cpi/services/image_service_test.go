package services

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/clients/clientsfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/cloud_properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/config"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/services/facades/facadesfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"net/http"
	"strings"
)

var _ = Describe("ImageService", func() {
	var serviceClient gophercloud.ServiceClient
	var imagesFacade facadesfakes.FakeImagesFacade
	var httpClient clientsfakes.FakeHttpClient
	var body map[string]interface{}

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		imagesFacade = facadesfakes.FakeImagesFacade{}
	})

	Context("CreateImage", func() {
		BeforeEach(func() {
			body = map[string]interface{}{
				"id": "123-456",
			}
		})

		It("returns the id of the created image entity in OpenStack", func() {
			imagesFacade.CreateReturns(createResult(body, nil, 201, nil))

			imageID, err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				CreateImage(cloud_properties.CreateStemcell{}, config.OpenstackConfig{})

			Expect(err).ToNot(HaveOccurred())
			Expect(imageID).To(Equal("123-456"))
		})

		It("create an image entity in OpenStack", func() {
			imagesFacade.CreateReturns(createResult(body, nil, 201, nil))

			cloudProps := cloud_properties.CreateStemcell{
				Name:            "the_stemcell_name",
				Version:         "the_stemcell_version",
				DiskFormat:      "the_disk_format",
				ContainerFormat: "the_container_format",
				OsType:          "the_os_type",
			}

			openstackConfig := config.OpenstackConfig{
				StemcellPubliclyVisible: true,
			}

			NewImageService(&serviceClient, &imagesFacade, &httpClient).
				CreateImage(cloudProps, openstackConfig)

			public := images.ImageVisibilityPublic
			createOpts := images.CreateOpts{
				Name:            "the_stemcell_name/the_stemcell_version",
				Visibility:      &public,
				DiskFormat:      "the_disk_format",
				ContainerFormat: "the_container_format",
				Properties: map[string]string{
					"version":             "the_stemcell_version",
					"os_type":             "the_os_type",
					"os_distro":           "",
					"architecture":        "",
					"auto_disk_config":    "",
					"hw_vif_model":        "",
					"hypervisor":          "",
					"vmware_adaptertype":  "",
					"vmware_disktype":     "",
					"vmware_linked_clone": "",
					"vmware_ostype":       "",
				},
			}

			serviceClient, opts := imagesFacade.CreateArgsForCall(0)
			Expect(serviceClient).To(Equal(serviceClient))
			Expect(opts).To(Equal(createOpts))
		})

		It("returns an error if image entity creation in OpenStack fails", func() {
			imagesFacade.CreateReturns(createResult(body, nil, 0, errors.New("boom")))

			imageID, err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				CreateImage(cloud_properties.CreateStemcell{}, config.OpenstackConfig{})

			Expect(err.Error()).To(Equal("failed to create image: boom"))
			Expect(imageID).To(Equal(""))
		})
	})

	Context("GetImage", func() {
		BeforeEach(func() {
			body = map[string]interface{}{
				"id":     "123-456-id",
				"status": "active",
			}
		})

		It("returns the id of an existing image entity in OpenStack", func() {
			imagesFacade.GetReturns(getResult(body, nil, 201, nil))

			NewImageService(&serviceClient, &imagesFacade, &httpClient).
				GetImage("123-456")

			serviceClient, imageID := imagesFacade.GetArgsForCall(0)
			Expect(serviceClient).To(Equal(serviceClient))
			Expect(imageID).To(Equal("123-456"))
		})

		It("get an existing image entity in OpenStack", func() {
			imagesFacade.GetReturns(getResult(body, nil, 201, nil))

			imageID, err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				GetImage("123-456")

			Expect(err).ToNot(HaveOccurred())
			Expect(imageID).To(Equal("123-456-id"))
		})

		It("returns an error if the image entity cannot be found in OpenStack", func() {
			imagesFacade.GetReturns(getResult(body, nil, 500, errors.New("boom")))

			imageID, err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				GetImage("123-456")

			Expect(err.Error()).To(Equal("could not find the image 123-456, that is referenced by the light stemcell, in OpenStack: boom"))
			Expect(imageID).To(Equal(""))
		})

		It("returns an error if the image entity is not active in OpenStack", func() {
			body = map[string]interface{}{
				"id":     "123-456-id",
				"status": "not-active",
			}

			imagesFacade.GetReturns(getResult(body, nil, 201, nil))

			imageID, err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				GetImage("123-456")

			Expect(err.Error()).To(Equal("image 123-456 is not in active state, it is in state: not-active"))
			Expect(imageID).To(Equal(""))
		})

	})

	Context("UploadImage", func() {
		BeforeEach(func() {

		})

		It("succeeds without error", func() {
			serviceClient.TokenID = "token"
			header := http.Header{}
			request := http.Request{Header: header}
			httpClient.NewRequestReturns(&request, nil)
			httpClient.DoReturns(&http.Response{StatusCode: 204}, nil)

			err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				UploadImage("123-456", "testdata/root.img")

			Expect(err).To(BeNil())
		})

		It("uploads the image via PUT", func() {
			request := http.Request{Header: http.Header{}}
			httpClient := clientsfakes.FakeHttpClient{}
			httpClient.NewRequestReturns(&request, nil)
			httpClient.DoReturns(&http.Response{StatusCode: 204}, nil)

			NewImageService(&serviceClient, &imagesFacade, &httpClient).
				UploadImage("123-456", "testdata/root.img")

			Expect(httpClient.DoCallCount()).To(Equal(1))
		})

		It("returns an error if the PUT request cannot be created", func() {
			httpClient.NewRequestReturns(nil, errors.New("boom"))

			err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				UploadImage("123-456", "testdata/root.img")

			Expect(err.Error()).To(Equal("failed to create request: boom"))
		})

		It("returns an error if the PUT request returns an error", func() {
			request := http.Request{Header: http.Header{}}
			httpClient.NewRequestReturns(&request, nil)
			httpClient.DoReturns(&http.Response{StatusCode: 204}, errors.New("boom"))

			err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				UploadImage("123-456", "testdata/root.img")

			Expect(err.Error()).To(Equal("failed to upload stemcell image to /v2/images/123-456/file, err: boom"))
		})

		It("returns an error if the PUT request returns status code != 204", func() {
			request := http.Request{Header: http.Header{}}
			response := &http.Response{StatusCode: 404, Status: "not found", Body: io.NopCloser(strings.NewReader("content"))}
			httpClient.NewRequestReturns(&request, nil)
			httpClient.DoReturns(response, nil)

			err := NewImageService(&serviceClient, &imagesFacade, &httpClient).
				UploadImage("123-456", "testdata/root.img")

			Expect(err.Error()).To(Equal("failed to upload stemcell image to /v2/images/123-456/file, response-status: 'not found', response-body:'content'\n"))
		})
	})
})

func createResult(body map[string]interface{}, header http.Header, statusCode int, err error) (result images.CreateResult) {
	result.Err = err
	result.Header = header
	result.Body = body
	result.StatusCode = statusCode
	return
}

func getResult(body map[string]interface{}, header http.Header, statusCode int, err error) (result images.GetResult) {
	result.Err = err
	result.Header = header
	result.Body = body
	result.StatusCode = statusCode
	return
}
