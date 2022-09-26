package bundle

import (
	"context"

	opRegistryApi "github.com/operator-framework/operator-registry/pkg/api"
	opRegistryClient "github.com/operator-framework/operator-registry/pkg/client"
)

func ListBundles(catalogPodName string) ([]*opRegistryApi.Bundle, error) {

	c, err := opRegistryClient.NewClient(catalogPodName)
	if err != nil {
		return nil, err
	}

	i, err := c.ListBundles(context.TODO())
	if err != nil {
		return nil, err
	}

	bundleList := []*opRegistryApi.Bundle{}

	for {

		b := i.Next()

		if b == nil {
			break
		}

		bundleList = append(bundleList, b)
	}

	return bundleList, nil
}
