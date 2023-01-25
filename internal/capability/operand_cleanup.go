package capability

import (
	"context"
	"fmt"

	"github.com/opdev/opcap/internal/logger"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// OperandCleanup removes the operand from the OCP cluster in the ca.namespace
func operandCleanup(ctx context.Context, audit *capAudit) error {
	logger.Debugw("cleaningUp operand for operator", "package", audit.subscription.Package, "channel", audit.subscription.Channel, "installmode",
		audit.subscription.InstallModeType)

	if len(audit.customResources) > 0 {
		for _, cr := range audit.customResources {
			obj := &unstructured.Unstructured{Object: cr}

			// extract name from CustomResource object and delete it
			name := obj.Object["metadata"].(map[string]interface{})["name"].(string)

			// check if CR exists, only then cleanup the operand
			err := audit.options.OpCapClient.GetUnstructured(ctx, audit.namespace, name, obj)
			if !apierrors.IsNotFound(err) {
				// Actual error. Return it
				return fmt.Errorf("could not get operaand: %v", err)
			}
			if obj == nil || apierrors.IsNotFound(err) {
				// Did not find it. Somehow already gone.
				// Not an error condition, but no point in
				// continuing.
				return nil
			}

			// delete the resource using the dynamic client
			if err := audit.options.OpCapClient.DeleteUnstructured(ctx, obj); err != nil {
				logger.Debugf("failed operandCleanUp: package: %s error: %s\n", audit.subscription.Package, err.Error())
				return err
			}

			// Forcing cleanup of finalizers
			err = audit.options.OpCapClient.GetUnstructured(ctx, audit.namespace, name, obj)
			if apierrors.IsNotFound(err) {
				return nil
			}

			obj.SetFinalizers([]string{})

			if err := audit.options.OpCapClient.UpdateUnstructured(ctx, obj); err != nil {
				return err
			}

			if err := audit.options.OpCapClient.GetUnstructured(ctx, audit.namespace, name, obj); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("error cleaning up operand after deleting finalizer: %v", err)
			}

		}
	}
	return nil
}
