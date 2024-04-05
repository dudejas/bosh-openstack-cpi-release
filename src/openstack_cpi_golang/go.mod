module github.com/cloudfoundry/bosh-golang-openstack-cpi-go/src/bosh-golang-openstack-cpi

go 1.21.0

//replace github.com/cloudfoundry/bosh-cpi-go => /Users/D044133/sap/cloudfoundry/bosh-cpi-go

require (
	github.com/cloudfoundry/bosh-cpi-go v0.0.0-20240224100157-0922490cd354
	github.com/cloudfoundry/bosh-utils v0.0.446
	github.com/google/uuid v1.6.0
	github.com/gophercloud/gophercloud v1.10.0
)

require (
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/charlievieth/fs v0.0.3 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
)
