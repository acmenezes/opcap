package capability

import (
	"context"

	"github.com/opdev/opcap/internal/logger"
)

func operatorCleanup(ctx context.Context, audit *capAudit) error {
	// delete subscription
	if err := audit.options.OpCapClient.DeleteSubscription(ctx, audit.subscription.Name, audit.namespace); err != nil {
		logger.Debugf("Error while deleting Subscription: %w", err)
		return err
	}

	// get csv using csvWatcher
	csv, err := audit.options.OpCapClient.GetCompletedCsvWithTimeout(ctx, audit.namespace, audit.options.Timeout)
	if err != nil {
		return err
	}

	// delete cluster service version
	if err := audit.options.OpCapClient.DeleteCSV(ctx, csv.ObjectMeta.Name, audit.namespace); err != nil {
		logger.Debugf("Error while deleting ClusterServiceVersion: %w", err)
		return err
	}

	// delete operator group
	if err := audit.options.OpCapClient.DeleteOperatorGroup(ctx, audit.operatorGroupData.Name, audit.namespace); err != nil {
		logger.Debugf("Error while deleting OperatorGroup: %w", err)
		return err
	}

	// delete target namespaces
	for _, ns := range audit.operatorGroupData.TargetNamespaces {
		if err := audit.options.OpCapClient.DeleteNamespace(ctx, ns); err != nil {
			logger.Debugf("Error deleting target namespace %s", ns)
			return err
		}
	}

	// delete operator's own namespace
	if err := audit.options.OpCapClient.DeleteNamespace(ctx, audit.namespace); err != nil {
		logger.Debugf("Error deleting operator's own namespace %s", audit.namespace)
		return err
	}
	return nil

}
