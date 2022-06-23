package operator

import (
	"context"
	"fmt"
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c opcapClient) InstallPlanApprove(namespace string) error {

	installPlanList := operatorv1alpha1.InstallPlanList{}

	listOpts := runtimeClient.ListOptions{
		Namespace: namespace,
	}

	err := c.Client.List(context.Background(), &installPlanList, &listOpts)
	if err != nil {
		logger.Errorf("Unable to list InstallPlans in Namespace %s: %w", namespace, err)
		return err
	}

	if len(installPlanList.Items) == 0 {
		logger.Errorf("no installPlan found in namespace %s: %w", namespace, err)
		return fmt.Errorf("no installPlan found in namespace %s", fmt.Sprint(len(installPlanList.Items)))
	}

	installPlan := operatorv1alpha1.InstallPlan{}

	err = c.Client.Get(context.Background(), types.NamespacedName{Name: installPlanList.Items[0].ObjectMeta.Name, Namespace: namespace}, &installPlan)

	if err != nil {
		logger.Errorf("no installPlan found in namespace %s: %w", namespace, err)
		return err
	}

	if installPlan.Spec.Approval == operatorv1alpha1.ApprovalManual {

		installPlan.Spec.Approved = true
		logger.Debugf("%s installPlan approved in Namespace %s", installPlan.ObjectMeta.Name, namespace)
		err := c.Client.Update(context.Background(), &installPlan)
		if err != nil {
			logger.Errorf("Error: %w", err)
			return err
		}
	}
	return nil
}

func (c opcapClient) WaitForInstallPlan(ctx context.Context, sub *operatorv1alpha1.Subscription) error {
	subKey := types.NamespacedName{
		Namespace: sub.GetNamespace(),
		Name:      sub.GetName(),
	}

	ipCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := c.Client.Get(ctx, subKey, sub); err != nil {
			return false, err
		}
		if sub.Status.InstallPlanRef != nil {
			return true, nil
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, ipCheck, ctx.Done()); err != nil {
		logger.Errorf("install plan is not available for the subscription %s: %w", sub.Name, err)
		return fmt.Errorf("install plan is not available for the subscription %s: %v", sub.Name, err)
	}
	return nil
}
