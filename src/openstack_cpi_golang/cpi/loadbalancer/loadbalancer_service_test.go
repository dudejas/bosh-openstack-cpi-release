package loadbalancer_test

import (
	"errors"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/loadbalancer/loadbalancerfakes"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/mocks"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/properties"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils"
	"github.com/cloudfoundry/bosh-openstack-cpi-release/src/openstack_cpi_golang/cpi/utils/utilsfakes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadbalancerService", func() {
	var serviceClient gophercloud.ServiceClient
	var retryableServiceClient gophercloud.ServiceClient
	var serviceClients utils.ServiceClients
	var loadbalancerFacade loadbalancerfakes.FakeLoadbalancerFacade
	var logger utilsfakes.FakeLogger
	var poolsPage mocks.MockPage

	BeforeEach(func() {
		serviceClient = gophercloud.ServiceClient{}
		retryableServiceClient = gophercloud.ServiceClient{}
		serviceClients = utils.ServiceClients{ServiceClient: &serviceClient, RetryableServiceClient: &retryableServiceClient}
		loadbalancerFacade = loadbalancerfakes.FakeLoadbalancerFacade{}
		logger = utilsfakes.FakeLogger{}
		poolsPage = mocks.MockPage{}

		loadbalancer.LoadbalancerServicePollingInterval = 0
		loadbalancerFacade.GetPoolReturns(&pools.Pool{ID: "pool-id", ProvisioningStatus: "ACTIVE"}, nil)
	})

	Context("GetPool", func() {
		BeforeEach(func() {
			loadbalancerFacade.ListPoolsReturns(poolsPage, nil)
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{{Name: "pool-name", ID: "pool-id"}}, nil)
		})

		It("lists loadbalancer pools", func() {
			loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			_, listOpts := loadbalancerFacade.ListPoolsArgsForCall(0)
			Expect(listOpts.Name).To(Equal("pool-name"))
		})

		It("returns an error if listing loadbalancer pools fails", func() {
			loadbalancerFacade.ListPoolsReturns(nil, errors.New("boom"))

			pool, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(err.Error()).To(Equal("failed to list loadbalancer pools: boom"))
			Expect(pool).To(Equal(pools.Pool{}))
		})

		It("extracts loadbalancer pools", func() {
			loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(loadbalancerFacade.ExtractPoolsArgsForCall(0)).To(Equal(poolsPage))
		})

		It("returns an error if extracting loadbalancer pools fails", func() {
			loadbalancerFacade.ExtractPoolsReturns(nil, errors.New("boom"))

			pool, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(err.Error()).To(Equal("failed to extract loadbalancer pool pages: boom"))
			Expect(pool.ID).To(Equal(""))
		})

		It("returns an error if pools are empty", func() {
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{}, nil)

			pool, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(err.Error()).To(Equal("loadbalancer pool 'pool-name' does not exist"))
			Expect(pool.ID).To(Equal(""))
		})

		It("returns an error if multiple pools with same name exists", func() {
			loadbalancerFacade.ExtractPoolsReturns([]pools.Pool{{Name: "pool-name"}, {Name: "pool-name"}}, nil)

			pool, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(err.Error()).To(Equal("found more than one loadbalancer pool with name 'pool-name'. Make sure to use unique naming"))
			Expect(pool.ID).To(Equal(""))
		})

		It("returns the pool ID", func() {
			pool, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				GetPool("pool-name")

			Expect(err).To(Not(HaveOccurred()))
			Expect(pool.ID).To(Equal("pool-id"))
		})
	})

	Context("CreatePoolMember", func() {
		BeforeEach(func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)
			loadbalancerFacade.CreatePoolMemberReturns(&pools.Member{ID: "the-member-id"}, nil)
		})

		It("waits for the pool to become ACTIVE", func() {
			loadbalancerFacade.GetPoolReturnsOnCall(0, &pools.Pool{ID: "pool-id", ProvisioningStatus: "PENDING_UPDATE"}, nil)
			loadbalancerFacade.GetPoolReturnsOnCall(1, &pools.Pool{ID: "pool-id", ProvisioningStatus: "ACTIVE"}, nil)

			_, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			retryableServiceClient, poolId := loadbalancerFacade.GetPoolArgsForCall(0)
			var utilsRetryableServiceClient utils.RetryableServiceClient
			Expect(retryableServiceClient).To(BeAssignableToTypeOf(utilsRetryableServiceClient))

			Expect(poolId).To(Equal("pool-id"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("times out while waiting for pool to become ACTIVE", func() {
			loadbalancerFacade.GetPoolReturns(&pools.Pool{ID: "pool-id", ProvisioningStatus: "PENDING_UPDATE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("timeout while waiting for pool 'pool-id' to become active"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if getting pool fails", func() {
			loadbalancerFacade.GetPoolReturns(nil, errors.New("boom"))

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("failed to retrieve pool 'pool-id': boom"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool is in state ERROR", func() {
			loadbalancerFacade.GetPoolReturns(&pools.Pool{ID: "pool-id", ProvisioningStatus: "ERROR"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("pool status ended up in ERROR state"))
			Expect(poolMember).To(BeNil())
		})

		It("creates a pool member", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{ProtocolPort: 1234}, "subnet-id", 1)

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
			loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{ProtocolPort: 1234, MonitoringPort: &monitoringPort}, "subnet-id", 1)

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

			_, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(Equal("failed to create pool member: boom"))
		})

		It("waits for the pool member to become ACTIVE", func() {
			loadbalancerFacade.GetPoolMemberReturnsOnCall(0, &pools.Member{ID: "the-member-id", ProvisioningStatus: "PENDING_CREATE"}, nil)
			loadbalancerFacade.GetPoolMemberReturnsOnCall(1, &pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(poolMember.ID).To(Equal("the-member-id"))
		})

		It("returns an error while waiting if getting pool member fails", func() {
			loadbalancerFacade.GetPoolMemberReturns(nil, errors.New("boom"))

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("failed to retrieve pool member 'the-member-id': boom"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool member creation finishes in state ERROR", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "ERROR"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("pool member creation finished with ERROR state"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool member provisioning status is unknown", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "unknown-status"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("pool member creation fails for unknown provisioning status 'unknown-status'"))
			Expect(poolMember).To(BeNil())
		})

		It("returns an error while waiting if the pool member creation times out", func() {
			loadbalancerFacade.GetPoolMemberReturns(&pools.Member{ID: "123-456", ProvisioningStatus: "PENDING_CREATE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err.Error()).To(ContainSubstring("timeout while waiting for pool member 'the-member-id' to become active"))
			Expect(poolMember).To(BeNil())
		})

		It("returns a pool member", func() {
			loadbalancerFacade.GetPoolMemberReturnsOnCall(0, &pools.Member{ID: "the-member-id", ProvisioningStatus: "ACTIVE"}, nil)

			poolMember, err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				CreatePoolMember("pool-id", "1.1.1.1", properties.LoadbalancerPool{}, "subnet-id", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(poolMember.ID).To(Equal("the-member-id"))
		})
	})

	Context("DeletePoolMember", func() {
		It("waits for the pool to become ACTIVE", func() {
			loadbalancerFacade.GetPoolReturnsOnCall(0, &pools.Pool{ID: "pool-id", ProvisioningStatus: "PENDING_UPDATE"}, nil)
			loadbalancerFacade.GetPoolReturnsOnCall(1, &pools.Pool{ID: "pool-id", ProvisioningStatus: "ACTIVE"}, nil)

			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-id", "member-id", 1)

			retryableServiceClient, poolId := loadbalancerFacade.GetPoolArgsForCall(0)
			var utilsRetryableServiceClient utils.RetryableServiceClient
			Expect(retryableServiceClient).To(BeAssignableToTypeOf(utilsRetryableServiceClient))

			Expect(poolId).To(Equal("pool-id"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("times out while waiting for pool to become ACTIVE", func() {
			loadbalancerFacade.GetPoolReturns(&pools.Pool{ID: "pool-id", ProvisioningStatus: "PENDING_UPDATE"}, nil)

			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-id", "member-id", 1)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout while waiting for pool 'pool-id' to become active"))
		})

		It("returns an error while waiting if getting pool fails", func() {
			loadbalancerFacade.GetPoolReturns(nil, errors.New("boom"))

			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-id", "member-id", 1)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to retrieve pool 'pool-id': boom"))
		})

		It("returns an error while waiting if the pool is in state ERROR", func() {
			loadbalancerFacade.GetPoolReturns(&pools.Pool{ID: "pool-id", ProvisioningStatus: "ERROR"}, nil)

			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-id", "member-id", 1)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pool status ended up in ERROR state"))
		})

		It("deletes pool member", func() {
			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-id", "member-id", 1)

			retryableServiceClient, _, _ := loadbalancerFacade.DeletePoolMemberArgsForCall(0)
			var utilsRetryableServiceClient utils.RetryableServiceClient
			Expect(retryableServiceClient).To(BeAssignableToTypeOf(utilsRetryableServiceClient))

			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if deleting pool member fails", func() {
			loadbalancerFacade.DeletePoolMemberReturns(errors.New("boom"))

			err := loadbalancer.NewLoadbalancerService(serviceClients, &loadbalancerFacade, &logger).
				DeletePoolMember("pool-name", "member-id", 1)

			Expect(err.Error()).To(Equal("failed to delete pool member: boom"))
		})
	})

})
