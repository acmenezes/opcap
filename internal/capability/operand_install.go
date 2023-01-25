package capability

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opdev/opcap/internal/logger"
	"github.com/opdev/opcap/internal/report"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func extractAlmExamples(ctx context.Context, audit *capAudit) error {
	// gets the list of CSVs present in a particular namespace
	csvList, err := audit.options.OpCapClient.ListClusterServiceVersions(ctx, audit.namespace)
	if err != nil {
		return err
	}
	almExamples := ""
	for _, csvVal := range csvList.Items {
		if strings.HasPrefix(csvVal.ObjectMeta.Name, audit.subscription.Package) {
			// map of string interface which consist of ALM examples from the CSVList
			almExamples = csvVal.ObjectMeta.Annotations["alm-examples"]
		}
	}
	var almList []map[string]interface{}

	err = yaml.Unmarshal([]byte(almExamples), &almList)
	if err != nil {
		return err
	}

	audit.customResources = append(audit.customResources, almList...)

	return nil
}

// OperandInstall installs the operand from the ALMExamples in the ca.namespace
func operandInstall(ctx context.Context, audit *capAudit) error {
	logger.Debugw("installing operand for operator", "package", audit.subscription.Package, "channel", audit.subscription.Channel, "installmode", audit.subscription.InstallModeType)

	if err := extractAlmExamples(ctx, audit); err != nil {
		logger.Errorf("could not get ALM Examples: %v", err)
	}

	if len(audit.customResources) == 0 {
		logger.Infow("exiting OperandInstall since no ALM_Examples found in CSV")
		return nil
	}

	csv, err := audit.options.OpCapClient.GetCompletedCsvWithTimeout(ctx, audit.namespace, time.Minute)
	if err != nil {
		return fmt.Errorf("could not get CSV: %v", err)
	}
	audit.csv = csv

	if strings.ToLower(string(csv.Status.Phase)) != "succeeded" {
		return fmt.Errorf("exiting OperandInstall since CSV install has failed")
	}

	for _, cr := range audit.customResources {
		obj := &unstructured.Unstructured{Object: cr}

		// set the namespace of CR to the namespace of the subscription
		obj.SetNamespace(audit.namespace)

		// create the resource using the dynamic client and log the error if it occurs
		err := audit.options.OpCapClient.CreateUnstructured(ctx, obj)
		if err != nil {
			// If there is an error, log and continue
			logger.Errorw("could not create resource", "error", err, "namespace", audit.namespace)
			continue
		}
		audit.operands = append(audit.operands, *obj)
	}

	file, err := audit.options.Fs.OpenFile("operand_install_report.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	err = report.OperandInstallJsonReport(file, report.TemplateData{
		CustomResources: audit.customResources,
		OcpVersion:      audit.ocpVersion,
		Subscription:    audit.subscription,
		Csv:             audit.csv,
		OperandCount:    len(audit.operands),
	})
	if err != nil {
		return fmt.Errorf("could not generate operand install JSON report: %v", err)
	}

	err = report.OperandInstallTextReport(audit.options.ReportWriter, report.TemplateData{
		CustomResources: audit.customResources,
		OcpVersion:      audit.ocpVersion,
		Subscription:    audit.subscription,
		Csv:             audit.csv,
		OperandCount:    len(audit.operands),
	})
	if err != nil {
		return fmt.Errorf("could not generate operand install text report: %v", err)
	}
	if audit.options.DetailedReports {
		if err = CollectDebugData(ctx, audit, "operand_detailed_report_all.json"); err != nil {
			return fmt.Errorf("couldn't collect debug data: %s", err)
		}
	}

	return nil
}
