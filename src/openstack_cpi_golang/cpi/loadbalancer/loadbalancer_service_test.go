package loadbalancer_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer/loadbalancerfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/mocks"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadbalancerService", func() {
	var serviceClient gophercloud.ServiceClient
	var loadbalancerFacade loadbalancerfakes.FakeLoadbalancerFacade
	var logger utilsfakes.FakeLogger
	var poolsPage mocks.MockPage

	BeforeEach(func() {
		providerClient := gophercloud.ProviderClient{TokenID: "the_token"}
		serviceClient = gophercloud.ServiceClient{ProviderClient: &providerClient}
		loadbalancerFacade = loadbalancerfakes.FakeLoadbalancerFacade{}
		logger = utilsfakes.FakeLogger{}
		poolsPage = mocks.MockPage{}
	})

	Context("GetPoolID", func() {

		BeforeEach(func() {
			loadbalancerFacade.ListPoolsReturns(poolsPage, nil)
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{{Name: "pool-name", ID: "pool-id"}}, nil)
		})

		It("lists loadbalancer pools", func() {
			loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			_, listOpts := loadbalancerFacade.ListPoolsArgsForCall(0)
			Expect(listOpts.Name).To(Equal("pool-name"))
		})

		It("returns an error if listing loadbalancer pools fails", func() {
			loadbalancerFacade.ListPoolsReturns(nil, errors.New("boom"))

			poolID, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(err.Error()).To(Equal("failed to list loadbalancer pools: boom"))
			Expect(poolID).To(Equal(""))
		})

		It("extracts loadbalancer pools", func() {
			loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(loadbalancerFacade.ExtractPoolsArgsForCall(0)).To(Equal(poolsPage))
		})

		It("returns an error if extracting loadbalancer pools fails", func() {
			loadbalancerFacade.ExtractPoolsReturns(nil, errors.New("boom"))

			poolID, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(err.Error()).To(Equal("failed to extract loadbalancer pool pages: boom"))
			Expect(poolID).To(Equal(""))
		})

		It("returns an error if pools are empty", func() {
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{}, nil)

			poolID, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(err.Error()).To(Equal("loadbalancer pool 'pool-name' does not exist"))
			Expect(poolID).To(Equal(""))
		})

		It("returns an error if multiple pools with same name exists", func() {
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{{Name: "pool-name"}, {Name: "pool-name"}}, nil)

			poolID, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(err.Error()).To(Equal("found more than one loadbalancer pool with name 'pool-name'. Make sure to use unique naming"))
			Expect(poolID).To(Equal(""))
		})

		It("returns the pool ID", func() {
			poolID, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				GetPoolID("pool-name")

			Expect(err).To(Not(HaveOccurred()))
			Expect(poolID).To(Equal("pool-id"))
		})
	})

	Context("CreatePoolMember", func() {

		BeforeEach(func() {
			loadbalancerFacade.CreatePoolMemberReturns(&pools.Member{ID: "the-member-id"}, nil)
		})

		It("creates a pool member", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{ProtocolPort: 1234}, "subnet-id", 0)

			_, poolID, createMemberOpts := loadbalancerFacade.CreatePoolMemberArgsForCall(0)
			Expect(poolID).To(Equal("pool-id"))
			Expect(createMemberOpts).To(Equal(pools.CreateMemberOpts{
				Address:      "1.1.1.1",
				ProtocolPort: 1234,
				SubnetID:     "subnet-id",
			}))
		})

		It("creates a pool member with monitoring port if provided", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			monitoringPort := 5678
			loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{ProtocolPort: 1234, MonitoringPort: &monitoringPort}, "subnet-id", 0)

			_, poolID, createMemberOpts := loadbalancerFacade.CreatePoolMemberArgsForCall(0)
			Expect(poolID).To(Equal("pool-id"))
			Expect(createMemberOpts).To(Equal(pools.CreateMemberOpts{
				Address:      "1.1.1.1",
				ProtocolPort: 1234,
				SubnetID:     "subnet-id",
				MonitorPort:  &monitoringPort,
			}))
		})

		It("returns an error if creating a pool member fails", func() {
			loadbalancerFacade.CreatePoolMemberReturns(nil, errors.New("boom"))

			_, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 0)

			Expect(err.Error()).To(Equal("failed to create pool member: boom"))
		})

		It("waits for the pool member to become ACTIVE", func() {
			loadbalancerFacade.GetPoolMemberReturnsOnCall(0, &pools.Member{ID: "the-member-id", ProvisioningStatus: "PENDING_CREATE"}, nil)
			loadbalancerFacade.GetPoolMemberReturnsOnCall(1, &pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			loadbalancer.LoadbalancerServicePollingInterval = 0

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 10)

			Expect(err).ToNot(HaveOccurred())
			Expect(poolMember.ID).To(Equal("the-member-id"))
		})

		It("returns an error while waiting if getting pool member fails", func() {
			loadbalancerFacade.GetPoolMemberReturns(nil, errors.New("boom"))

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 100)

			Expect(err.Error()).To(ContainSubstring("failed to retrieve pool member 'the-member-id': boom"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool member creation finishes in state ERROR", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "ERROR"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 10)

			Expect(err.Error()).To(ContainSubstring("pool member creation finished with ERROR state"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool member provisioning status is unknown", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "unknown-status"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 10)

			Expect(err.Error()).To(ContainSubstring("pool member creation fails for unknown provisioning status 'unknown-status'"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the server creation times out", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "PENDING_CREATE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 0)

			Expect(err.Error()).To(ContainSubstring("imeout while waiting for pool member 'the-member-id' to become active"))
			Expect(poolMember).To(BeNil())
		})

		It("returns a pool member", func() {
			loadbalancerFacade.GetPoolMemberReturnsOnCall(0, &pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 10)

			Expect(err).ToNot(HaveOccurred())
			Expect(poolMember.ID).To(Equal("the-member-id"))
		})
	})

	Context("DeletePoolMember", func() {

		BeforeEach(func() {
			loadbalancerFacade.ListPoolsReturns(poolsPage, nil)
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{{Name: "pool-name", ID: "pool-id"}}, nil)
		})

		It("deletes pool member", func() {
			loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-name", "member-id")

			test, _, _ := loadbalancerFacade.DeletePoolMemberArgsForCall(0)
			Expect(test).To(Equal(&serviceClient))
			Expect(test.RetryFunc).ToNot(Equal(nil))
		})

		It("returns an error if deleting pool member fails", func() {
			loadbalancerFacade.DeletePoolMemberReturns(errors.New("boom"))

			err := loadbalancer.NewLoadbalancerService(&serviceClient, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-name", "member-id")

			Expect(err.Error()).To(Equal("failed to delete pool member: boom"))
		})
	})

})
