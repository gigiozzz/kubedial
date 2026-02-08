package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// ApplyOptions contains options for apply operation
type ApplyOptions struct {
	Namespace  string
	ServerSide bool
	DryRun     bool
	Force      bool
}

// DeleteOptions contains options for delete operation
type DeleteOptions struct {
	Namespace string
	Force     bool
}

// Applyer defines the interface for applying/deleting Kubernetes manifests
type Applyer interface {
	// Apply applies manifests to the cluster
	Apply(ctx context.Context, manifests []byte, opts ApplyOptions) (string, error)

	// Delete deletes manifests from the cluster
	Delete(ctx context.Context, manifests []byte, opts DeleteOptions) (string, error)
}

// K8sApplyer implements Applyer using k8s.io/cli-runtime pattern
type K8sApplyer struct {
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	mapper          meta.RESTMapper
}

// NewK8sApplyer creates a new K8sApplyer
func NewK8sApplyer(config *rest.Config) (*K8sApplyer, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get API group resources: %w", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	return &K8sApplyer{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		mapper:          mapper,
	}, nil
}

// Apply applies manifests to the cluster
func (a *K8sApplyer) Apply(ctx context.Context, manifests []byte, opts ApplyOptions) (string, error) {
	objects, err := a.decodeManifests(manifests)
	if err != nil {
		return "", fmt.Errorf("failed to decode manifests: %w", err)
	}

	var output strings.Builder
	for _, obj := range objects {
		result, err := a.applyObject(ctx, obj, opts)
		if err != nil {
			return output.String(), fmt.Errorf("failed to apply %s/%s: %w",
				obj.GetKind(), obj.GetName(), err)
		}
		output.WriteString(result + "\n")
	}

	return output.String(), nil
}

// Delete deletes manifests from the cluster
func (a *K8sApplyer) Delete(ctx context.Context, manifests []byte, opts DeleteOptions) (string, error) {
	objects, err := a.decodeManifests(manifests)
	if err != nil {
		return "", fmt.Errorf("failed to decode manifests: %w", err)
	}

	var output strings.Builder
	// Delete in reverse order
	for i := len(objects) - 1; i >= 0; i-- {
		obj := objects[i]
		result, err := a.deleteObject(ctx, obj, opts)
		if err != nil {
			return output.String(), fmt.Errorf("failed to delete %s/%s: %w",
				obj.GetKind(), obj.GetName(), err)
		}
		output.WriteString(result + "\n")
	}

	return output.String(), nil
}

func (a *K8sApplyer) decodeManifests(manifests []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(manifests), 4096)
	decSerializer := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if rawObj.Raw == nil {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, _, err := decSerializer.Decode(rawObj.Raw, nil, obj)
		if err != nil {
			return nil, err
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

func (a *K8sApplyer) applyObject(ctx context.Context, obj *unstructured.Unstructured, opts ApplyOptions) (string, error) {
	gvk := obj.GroupVersionKind()

	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("failed to get REST mapping: %w", err)
	}

	// Set namespace if specified and resource is namespaced
	namespace := obj.GetNamespace()
	if opts.Namespace != "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			namespace = opts.Namespace
			obj.SetNamespace(namespace)
		}
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = a.dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		dr = a.dynamicClient.Resource(mapping.Resource)
	}

	// Build apply options
	applyOpts := metav1.ApplyOptions{
		FieldManager: "kubedialer",
	}
	if opts.DryRun {
		applyOpts.DryRun = []string{metav1.DryRunAll}
	}
	if opts.Force {
		applyOpts.Force = true
	}

	if opts.ServerSide {
		// Server-side apply
		data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
		if err != nil {
			return "", fmt.Errorf("failed to encode object: %w", err)
		}

		_, err = dr.Patch(ctx, obj.GetName(), "application/apply-patch+yaml", data, metav1.PatchOptions{
			FieldManager: "kubedialer",
			DryRun:       applyOpts.DryRun,
			Force:        &applyOpts.Force,
		})
		if err != nil {
			return "", err
		}
	} else {
		// Client-side apply (create or update)
		existing, err := dr.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				createOpts := metav1.CreateOptions{}
				if opts.DryRun {
					createOpts.DryRun = []string{metav1.DryRunAll}
				}
				_, err = dr.Create(ctx, obj, createOpts)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s/%s created", obj.GetKind(), obj.GetName()), nil
			}
			return "", err
		}

		obj.SetResourceVersion(existing.GetResourceVersion())
		updateOpts := metav1.UpdateOptions{}
		if opts.DryRun {
			updateOpts.DryRun = []string{metav1.DryRunAll}
		}
		_, err = dr.Update(ctx, obj, updateOpts)
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%s/%s configured", obj.GetKind(), obj.GetName()), nil
}

func (a *K8sApplyer) deleteObject(ctx context.Context, obj *unstructured.Unstructured, opts DeleteOptions) (string, error) {
	gvk := obj.GroupVersionKind()

	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("failed to get REST mapping: %w", err)
	}

	namespace := obj.GetNamespace()
	if opts.Namespace != "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			namespace = opts.Namespace
		}
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = a.dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		dr = a.dynamicClient.Resource(mapping.Resource)
	}

	deletePolicy := metav1.DeletePropagationForeground
	if opts.Force {
		deletePolicy = metav1.DeletePropagationBackground
	}

	err = dr.Delete(ctx, obj.GetName(), metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Sprintf("%s/%s not found (skipped)", obj.GetKind(), obj.GetName()), nil
		}
		return "", err
	}

	return fmt.Sprintf("%s/%s deleted", obj.GetKind(), obj.GetName()), nil
}
