package oci

import (
	"context"
	"net/http"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	v2 "github.com/distribution/distribution/v3/registry/api/v2"
	"github.com/distribution/distribution/v3/registry/extension"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/gorilla/handlers"
	"gopkg.in/yaml.v2"
)

const (
	namespaceName         = "oci"
	extensionName         = "ext"
	discoverComponentName = "discover"
)

type distributionNamespace struct {
	storageDriver   driver.StorageDriver
	discoverEnabled bool
}

type distributionOptions struct {
	RegExtensionComponents []string `yaml:"ext,omitempty"`
}

// newOciNamespace creates a new extension namespace with the name "oci"
func newOciNamespace(ctx context.Context, storageDriver driver.StorageDriver, options configuration.ExtensionConfig) (extension.Namespace, error) {

	optionsYaml, err := yaml.Marshal(options)
	if err != nil {
		return nil, err
	}

	var distOptions distributionOptions
	err = yaml.Unmarshal(optionsYaml, &distOptions)
	if err != nil {
		return nil, err
	}

	discoverEnabled := false
	for _, component := range distOptions.RegExtensionComponents {
		switch component {
		case "discover":
			discoverEnabled = true
		}
	}

	return &distributionNamespace{
		storageDriver:   storageDriver,
		discoverEnabled: discoverEnabled,
	}, nil
}

func init() {
	// register the extension namespace.
	extension.Register(namespaceName, newOciNamespace)
}

// GetManifestHandlers returns a list of manifest handlers that will be registered in the manifest store.
func (o *distributionNamespace) GetManifestHandlers(repo distribution.Repository, blobStore distribution.BlobStore) []storage.ManifestHandler {
	// This extension doesn't extend any manifest store operations.
	return []storage.ManifestHandler{}
}

// GetRepositoryRoutes returns a list of extension routes scoped at a repository level
func (d *distributionNamespace) GetRepositoryRoutes() []extension.Route {
	var routes []extension.Route

	if d.discoverEnabled {
		routes = append(routes, extension.Route{
			Namespace: namespaceName,
			Extension: extensionName,
			Component: discoverComponentName,
			Descriptor: v2.RouteDescriptor{
				Entity: "Extension",
				Methods: []v2.MethodDescriptor{
					{
						Method:      "GET",
						Description: "Get all extensions enabled for a repository.",
					},
				},
			},
			Dispatcher: d.discoverDispatcher,
		})
	}

	return routes
}

// GetRegistryRoutes returns a list of extension routes scoped at a registry level
// There are no registry scoped routes exposed by this namespace
func (d *distributionNamespace) GetRegistryRoutes() []extension.Route {
	return nil
}

func (d *distributionNamespace) discoverDispatcher(ctx *extension.Context, r *http.Request) http.Handler {
	extensionHandler := &extensionHandler{
		Context:       ctx,
		storageDriver: d.storageDriver,
	}

	return handlers.MethodHandler{
		"GET": http.HandlerFunc(extensionHandler.getExtensions),
	}
}
