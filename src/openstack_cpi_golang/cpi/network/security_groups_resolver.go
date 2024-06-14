package network

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
)

//counterfeiter:generate . SecurityGroupsResolver
type SecurityGroupsResolver interface {
	Resolve(securityGroupIDsAndNames []string) ([]string, error)
}

type securityGroupsResolver struct {
	serviceClient    *gophercloud.ServiceClient
	networkingFacade NetworkingFacade
}

func NewSecurityGroupsResolver(serviceClient *gophercloud.ServiceClient,
	networkingFacade NetworkingFacade) securityGroupsResolver {
	return securityGroupsResolver{
		serviceClient:    serviceClient,
		networkingFacade: networkingFacade,
	}
}

func (s securityGroupsResolver) Resolve(securityGroupIDsAndNames []string) ([]string, error) {
	var securityGroupIds []string
	var resolvedSecurityGroup *groups.SecGroup
	var err error

	for _, securityGroup := range securityGroupIDsAndNames {
		resolvedSecurityGroup, err = s.resolveSecurityGroupById(securityGroup)
		if err != nil {
			return []string{}, fmt.Errorf("failed to get security group '%s' by id: %w", securityGroup, err)
		}

		if resolvedSecurityGroup != nil {
			securityGroupIds = append(securityGroupIds, resolvedSecurityGroup.ID)
			continue
		} else {
			resolvedSecurityGroup, err = s.resolveSecurityGroupByName(securityGroup)
			if err != nil {
				return []string{}, fmt.Errorf("failed to get security group '%s' by name: %w", securityGroup, err)
			}

			securityGroupIds = append(securityGroupIds, resolvedSecurityGroup.ID)
			continue
		}

	}
	return securityGroupIds, nil
}

func (s securityGroupsResolver) resolveSecurityGroupById(securityGroupID string) (*groups.SecGroup, error) {
	return s.networkingFacade.GetSecurityGroups(s.serviceClient, securityGroupID)
}

func (s securityGroupsResolver) resolveSecurityGroupByName(securityGroupName string) (*groups.SecGroup, error) {
	listOpts := groups.ListOpts{
		Name: securityGroupName,
	}

	allPages, err := s.networkingFacade.ListSecurityGroups(s.serviceClient, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	allSecurityGroups, err := s.networkingFacade.ExtractSecurityGroups(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}

	if len(allSecurityGroups) == 0 {
		return nil, fmt.Errorf("security group '%s' could not be found", securityGroupName)
	}

	return &allSecurityGroups[0], nil
}
