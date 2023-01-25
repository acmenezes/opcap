package capability

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/opdev/opcap/internal/logger"
	"github.com/opdev/opcap/internal/operator"
	"github.com/opdev/opcap/internal/report"
)

func operatorInstall(ctx context.Context, audit *capAudit) error {
	logger.Debugw("installing package", "package", audit.subscription.Package, "channel", audit.subscription.Channel, "installmode", audit.subscription.InstallModeType)

	// create operator's own namespace
	if _, err := audit.options.OpCapClient.CreateNamespace(ctx, audit.namespace); err != nil {
		return err
	}

	// create remaining target namespaces watched by the operator
	for _, ns := range audit.operatorGroupData.TargetNamespaces {
		if ns != audit.namespace {
			audit.options.OpCapClient.CreateNamespace(ctx, ns)
		}
	}

	// create operator group for operator package/channel
	audit.options.OpCapClient.CreateOperatorGroup(ctx, audit.operatorGroupData, audit.namespace)

	// create subscription for operator package/channel
	if _, err := audit.options.OpCapClient.CreateSubscription(ctx, audit.subscription, audit.namespace); err != nil {
		logger.Debugf("Error creating subscriptions: %w", err)
		return err
	}

	// Get a Succeeded or Failed CSV with one minute timeout
	resultCSV, err := audit.options.OpCapClient.GetCompletedCsvWithTimeout(ctx, audit.namespace, audit.options.Timeout)
	if err != nil {
		// If error is timeout than don't log phase but timeout
		if errors.Is(err, operator.TimeoutError) {
			audit.csvTimeout = true
			audit.csv = resultCSV
			if err = CollectDebugData(ctx, audit, "operator_detailed_report_timeout.json"); err != nil {
				return fmt.Errorf("couldn't collect debug data: %s", err)
			}

		} else {
			return err
		}
	}
	audit.csv = resultCSV

	file, err := audit.options.Fs.OpenFile("operator_install_report.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	err = report.OperatorInstallJsonReport(file, report.TemplateData{
		OcpVersion:   audit.ocpVersion,
		Subscription: audit.subscription,
		Csv:          audit.csv,
		CsvTimeout:   audit.csvTimeout,
	})
	if err != nil {
		return fmt.Errorf("could not generate operator install JSON report: %v", err)
	}

	err = report.OperatorInstallTextReport(audit.options.ReportWriter, report.TemplateData{
		OcpVersion:   audit.ocpVersion,
		Subscription: audit.subscription,
		Csv:          audit.csv,
		CsvTimeout:   audit.csvTimeout,
	})
	if err != nil {
		return fmt.Errorf("could not generate operator install text report: %v", err)
	}
	if audit.options.DetailedReports {
		if err = CollectDebugData(ctx, audit, "operator_detailed_report_all.json"); err != nil {
			return fmt.Errorf("couldn't collect debug data: %s", err)
		}
	}

	return nil
}
