package capability

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/opdev/opcap/internal/logger"
	"github.com/opdev/opcap/internal/operator"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type AuditorOptions struct {
	// AuditPlan holds the tests that should be run during an audit
	AuditPlan []string

	// CatalogSource may be built-in OLM or custom
	CatalogSource string
	// CatalogSourceNamespace will be openshift-marketplace or custom
	CatalogSourceNamespace string

	// Packages is a subset of packages to be tested from a catalogSource
	Packages []string

	// WorkQueue holds capAudits in a buffered channel in order to execute them
	WorkQueue chan *capAudit

	// AllInstallModes will test all install modes supported by an operator
	AllInstallModes bool

	// extraCustomResources associates packages to a list of Custom Resources (in addition to ALMExamples)
	// to be audited by the OperandInstall AuditPlan.
	ExtraCustomResources string

	// OpCapClient is the main OpenShift client interface
	OpCapClient operator.Client

	// Fs is an afero filesystem used by the auditor
	Fs afero.Fs

	// Timeout is the audit timeout
	Timeout time.Duration

	//  ReportWriter is any io.Writer for the text reports
	ReportWriter io.Writer

	// DetailedReports creates reports containing events and logs
	DetailedReports bool
}

type customResources = map[string][]map[string]interface{}

// ExtraCRDirectory scans the provided directory and populates the extraCustomResources field.
// Is is expected that the extraCRDirectory posesses subdirectories. Manifest files are present in each subdirectory.
// The name of the subdirectory is used to determine which package the manifest files are corresponding to.
// The resulting structure would be:
// custom_resources_directory/
// ├── package_name1
// │   ├── manifest_file1.json
// │   └── manifest_file2.yaml
// └── package_name2
//
//	   ├── manifest_file1.json
//
//     └── manifest_file2.yaml
func extraCRDirectory(ctx context.Context, options *AuditorOptions) (customResources, error) {
	logger.Debugw("scaning for extra Custom Resources", "extra CR directory", options.ExtraCustomResources)
	extraCustomResources := customResources{} // maps packages to a list of CR

	extraCRDirectoryAbsolutePath, err := filepath.Abs(options.ExtraCustomResources)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path from %s: %v", options.ExtraCustomResources, err)
	}

	err = afero.Walk(options.Fs, extraCRDirectoryAbsolutePath, func(path string, d fs.FileInfo, err error) error {
		if err != nil && path == extraCRDirectoryAbsolutePath {
			// Error reading the root directory, exit and return the error
			return err
		}

		if !d.IsDir() { // Act on files only
			manifestFilePath := path
			// Checking that the manifest file is placed in a subdirectory
			if len(strings.Split(manifestFilePath, "/")) != len(strings.Split(extraCRDirectoryAbsolutePath, "/"))+2 {
				logger.Errorf("Error handling manifest file %s. File should be placed in a subdirectory of %s", manifestFilePath, options.ExtraCustomResources)
				return nil // continue
			}

			// Get the name of the subdirectory containing the manifest file. This corresponds to the package name.
			packageName := filepath.Base(filepath.Dir(manifestFilePath))

			logger.Debugw("adding Custom Resource", "source manifest file", manifestFilePath, "package", packageName)

			// Get manifest file content
			manifestBytes, err := afero.ReadFile(options.Fs, manifestFilePath)
			if err != nil {
				logger.Errorf("Error reading file %s: %v", manifestFilePath, err)
				return nil // continue
			}

			var manifest map[string]interface{}
			err = yaml.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				logger.Errorf("Error unmarshalling file %s: %v", manifestFilePath, err)
				return nil // continue
			}

			if manifest == nil {
				logger.Errorf("Empty manifest file %s", manifestFilePath)
				return nil // continue
			}

			// Add the Custom Resource to the list of extra Custom Resource for this package
			var customResourceManifests []map[string]interface{}
			if _, packageKeyPresent := extraCustomResources[packageName]; packageKeyPresent {
				customResourceManifests = extraCustomResources[packageName]
			}
			customResourceManifests = append(customResourceManifests, manifest)

			extraCustomResources[packageName] = customResourceManifests
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not read directory %s: %v", options.ExtraCustomResources, err)
	}

	return extraCustomResources, nil
}

// BuildWorkQueueByCatalog fills in the auditor workqueue with all package information found in a specific catalog
func buildWorkQueueByCatalog(ctx context.Context, options *AuditorOptions, extraCustomResources customResources) error {
	// Getting subscription data form the package manifests available in the selected catalog
	subscriptions, err := options.OpCapClient.GetSubscriptionData(ctx, options.CatalogSource, options.CatalogSourceNamespace, options.Packages)
	if err != nil {
		return fmt.Errorf("could not get bundles from CatalogSource: %s: %v", options.CatalogSource, err)
	}

	// build workqueue as buffered channel based subscriptionData list size
	options.WorkQueue = make(chan *capAudit, len(subscriptions))
	defer close(options.WorkQueue)

	// packagesToBeAudited is a subset of packages to be tested from a catalogSource
	var packagesToBeAudited []operator.SubscriptionData

	// get all install modes for all operators in the catalog
	// and add them to the packagesToBeAudited list
	if options.AllInstallModes {
		packagesToBeAudited = subscriptions
	} else {
		packages := make(map[string]bool)
		for _, subscription := range subscriptions {
			if _, exists := packages[subscription.Package]; !exists {
				packages[subscription.Package] = true
				packagesToBeAudited = append(packagesToBeAudited, subscription)
			}
		}
	}

	// add capAudits to the workqueue
	for _, subscription := range packagesToBeAudited {
		// Get extra Custom Resources for this subscription, if any
		mapExtraCustomResources := []map[string]interface{}{}
		extraCustomResources, ok := extraCustomResources[subscription.Package]
		if ok {
			mapExtraCustomResources = extraCustomResources
		}

		capAudit, err := newCapAudit(ctx, options, subscription, mapExtraCustomResources)
		if err != nil {
			return fmt.Errorf("could not build configuration for subscription: %s: %v", subscription.Name, err)
		}

		// load workqueue with capAudit
		options.WorkQueue <- capAudit
	}

	return nil
}

// RunAudits executes all selected functions in order for a given audit at a time
func RunAudits(ctx context.Context, options AuditorOptions) error {

	var extraCustomResources customResources
	if options.ExtraCustomResources != "" {
		var err error
		extraCustomResources, err = extraCRDirectory(ctx, &options)
		if err != nil {
			return fmt.Errorf("could not read extra custom resources directory: %v", err)
		}
	}

	err := buildWorkQueueByCatalog(ctx, &options, extraCustomResources)
	if err != nil {
		return fmt.Errorf("unable to build workqueue: %v", err)
	}

	// read workqueue for audits
	for capAudit := range options.WorkQueue {
		// read a particular audit's auditPlan for functions
		// to be executed against operator
		cleanups := []string{}
		for _, function := range capAudit.options.AuditPlan {

			if function == "OperatorInstall" {
				operatorInstall(ctx, capAudit)
				cleanups = append(cleanups, "OperatorCleanup")
			}

			if function == "OperandInstall" {
				operandInstall(ctx, capAudit)
				cleanups = append(cleanups, "OperandCleanup")
			}
		}
		for _, cleanup := range cleanups {

			if cleanup == "OperatorCleanup" {
				operatorCleanup(ctx, capAudit)
			}
			if cleanup == "OperandCleanup" {
				operandCleanup(ctx, capAudit)
			}
		}
	}
	return nil
}
