package client

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/scheme"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/auth"
	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/replace"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// AnnotationSyncOptions is a comma-separated list of options for syncing
const (
	// AnnotationSyncOptions is a comma-separated list of options for syncing
	AnnotationSyncOptions = toolchainv1alpha1.LabelKeyPrefix + "sync-options"
	// SyncOptionReplace option that enables use of replace or create command instead of apply
	SyncOptionReplace = "Replace=true"
	// SyncOptionForce option that enables use of --force flag, delete and re-create
	SyncOptionForce = "Force=true"
	// SyncOptionServerSideApply option that enables use of --server-side flag instead of client-side
	SyncOptionServerSideApply = "ServerSideApply=true"

	CustomResourceDefinitionKind = "CustomResourceDefinition"
	NamespaceKind                = "Namespace"
)

// AnnotationGetter defines the operations required to inspect if a resource
// has annotations
type AnnotationGetter interface {
	GetAnnotations() map[string]string
}

// GetAnnotationValues will return the value of the annotation identified by
// the given key. If the annotation has comma separated values, the returned
// list will contain all deduplicated values.
func GetAnnotationValues(obj AnnotationGetter, key string) []string {
	// may for de-duping
	valuesToBool := make(map[string]bool)
	for _, item := range strings.Split(obj.GetAnnotations()[key], ",") {
		val := strings.TrimSpace(item)
		if val != "" {
			valuesToBool[val] = true
		}
	}
	var values []string
	for val := range valuesToBool {
		values = append(values, val)
	}
	return values
}

// HasAnnotationOption will return if the given obj has an annotation defined
// as the given key and has in its values, the ocurrence of val.
func HasAnnotationOption(obj AnnotationGetter, key, val string) bool {
	for _, item := range GetAnnotationValues(obj, key) {
		if item == val {
			return true
		}
	}
	return false
}

func IsCRDGroupVersionKind(gvk schema.GroupVersionKind) bool {
	return gvk.Kind == CustomResourceDefinitionKind && gvk.Group == "apiextensions.k8s.io"
}

func IsCRD(obj *unstructured.Unstructured) bool {
	return IsCRDGroupVersionKind(obj.GroupVersionKind())
}

type KubectlResourceOperations struct {
	config              *rest.Config
	fact                cmdutil.Factory
	streams             genericclioptions.IOStreams
	extensionsclientset *clientset.Clientset
	dynamicClient       dynamic.Interface
}

func NewKubectlResourceOperations(factory cmdutil.Factory, ioStreams genericclioptions.IOStreams, config *rest.Config, dynamicClient dynamic.Interface) (*KubectlResourceOperations, error) {
	extensionsclientset, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubectlResourceOperations{
		config:              config,
		streams:             ioStreams,
		fact:                factory,
		extensionsclientset: extensionsclientset,
		dynamicClient:       dynamicClient,
	}, nil
}

func (k *KubectlResourceOperations) ApplyUnstructuredObject(ctx context.Context, targetObj, liveObj *unstructured.Unstructured, serverSideApply, validate bool, serverSideApplyManager string) (bool, error) {
	var err error
	var message string
	shouldReplace := HasAnnotationOption(targetObj, AnnotationSyncOptions, SyncOptionReplace)
	force := HasAnnotationOption(targetObj, AnnotationSyncOptions, SyncOptionForce)
	serverSideApply = serverSideApply || HasAnnotationOption(targetObj, AnnotationSyncOptions, SyncOptionServerSideApply)
	if shouldReplace {
		if liveObj != nil {
			// Avoid using `kubectl replace` for CRDs since 'replace' might recreate resource and so delete all CRD instances.
			// The same thing applies for namespaces, which would delete the namespace as well as everything within it,
			// so we want to avoid using `kubectl replace` in that case as well.
			if IsCRD(targetObj) || targetObj.GetKind() == NamespaceKind {
				update := targetObj.DeepCopy()
				update.SetResourceVersion(liveObj.GetResourceVersion())
				_, err = k.UpdateResource(ctx, update)
			} else {
				message, err = k.ReplaceResource(targetObj, force)
			}
		} else {
			message, err = k.CreateResource(targetObj, validate)
		}
	} else {
		message, err = k.ApplyResource(targetObj, force, validate, serverSideApply, serverSideApplyManager)
	}
	log.Info(message)
	if err != nil {
		return false, err
	}
	if IsCRD(targetObj) {
		crdName := targetObj.GetName()
		if err = k.ensureCRDReady(crdName); err != nil {
			log.Error(err, fmt.Sprintf("failed to ensure that CRD %s is ready", crdName))
		}
	}
	return true, nil
}

// ensureCRDReady waits until the specified CRD is ready (established condition is true).
func (k *KubectlResourceOperations) ensureCRDReady(name string) error {
	const crdReadinessTimeout = time.Duration(3) * time.Second
	return wait.Poll(time.Duration(100)*time.Millisecond, crdReadinessTimeout, func() (bool, error) {
		crd, err := k.extensionsclientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, condition := range crd.Status.Conditions {
			if condition.Type == v1extensions.Established {
				return condition.Status == v1extensions.ConditionTrue, nil
			}
		}
		return false, nil
	})
}

type commandExecutor func(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) error

func (k *KubectlResourceOperations) runResourceCommand(obj *unstructured.Unstructured, executor commandExecutor) (string, error) {
	var out []string
	// rbac resouces are first applied with auth reconcile kubectl feature.
	// serverSideDiff should avoid this step as the resources are not being actually
	// applied but just running in dryrun mode. Also, kubectl auth reconcile doesn't
	// currently support running dryrun in server mode.
	if obj.GetAPIVersion() == "rbac.authorization.k8s.io/v1" {
		outReconcile, err := k.rbacReconcile(obj)
		if err != nil {
			return "", fmt.Errorf("error running rbacReconcile: %s", err)
		}
		out = append(out, outReconcile)
		// We still want to fallthrough and run `kubectl apply` in order set the
		// last-applied-configuration annotation in the object.
	}

	// Run kubectl apply
	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	err := executor(k.fact, ioStreams)
	if err != nil {
		return "", err
	}
	if buf := strings.TrimSpace(ioStreams.Out.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	if buf := strings.TrimSpace(ioStreams.ErrOut.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	return strings.Join(out, ". "), nil
}

// rbacReconcile will perform reconciliation for RBAC resources. It will run
// the following command:
//
//	kubectl auth reconcile
//
// This is preferred over `kubectl apply`, which cannot tolerate changes in
// roleRef, which is an immutable field.
// `auth reconcile` will delete and recreate the resource if necessary
func (k *KubectlResourceOperations) rbacReconcile(obj *unstructured.Unstructured) (string, error) {
	outReconcile, err := k.authReconcile(obj)
	if err != nil {
		return "", fmt.Errorf("error running kubectl auth reconcile: %w", err)
	}
	return outReconcile, nil
}

func (k *KubectlResourceOperations) ReplaceResource(obj *unstructured.Unstructured, force bool) (string, error) {
	log.Info(fmt.Sprintf("Replacing resource %s/%s in cluster: %s, namespace: %s", obj.GetKind(), obj.GetName(), k.config.Host, obj.GetNamespace()))
	return k.runResourceCommand(obj, func(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) error {
		replaceOptions, err := k.newReplaceOptions(obj, force)
		if err != nil {
			return err
		}
		return replaceOptions.Run(f)
	})
}

func (k *KubectlResourceOperations) CreateResource(obj *unstructured.Unstructured, validate bool) (string, error) {
	return k.runResourceCommand(obj, func(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) error {
		createOptions, err := k.newCreateOptions()
		if err != nil {
			return err
		}
		command := &cobra.Command{}
		saveConfig := false
		command.Flags().BoolVar(&saveConfig, "save-config", false, "")
		val := false
		command.Flags().BoolVar(&val, "validate", false, "")
		if validate {
			_ = command.Flags().Set("validate", "true")
		}

		return createOptions.RunCreate(f, command)
	})
}

func (k *KubectlResourceOperations) UpdateResource(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvk := obj.GroupVersionKind()
	dynamicIf, err := dynamic.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(k.config)
	if err != nil {
		return nil, err
	}
	apiResource, err := ServerResourceForGroupVersionKind(disco, gvk, "update")
	if err != nil {
		return nil, err
	}
	resource := gvk.GroupVersion().WithResource(apiResource.Name)
	resourceIf := ToResourceInterface(dynamicIf, apiResource, resource, obj.GetNamespace())

	return resourceIf.Update(ctx, obj, metav1.UpdateOptions{})
}

// ServerResourceForGroupVersionKind looks up and returns the API resource from
// the server for a given GVK scheme. If verb is set to the non-empty string,
// it will return the API resource which supports the verb. There are some edge
// cases, where the same GVK is represented by more than one API.
//
// See: https://github.com/ksonnet/ksonnet/blob/master/utils/client.go
func ServerResourceForGroupVersionKind(disco discovery.DiscoveryInterface, gvk schema.GroupVersionKind, verb string) (*metav1.APIResource, error) {
	// default is to return a not found for the requested resource
	retErr := apierr.NewNotFound(schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}, "")
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, err
	}
	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			if isSupportedVerb(&r, verb) {
				return &r, nil
			} else {
				// We have a match, but the API does not support the action
				// that was requested. Memorize this.
				retErr = apierr.NewMethodNotSupported(schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}, verb)
			}
		}
	}
	return nil, retErr
}

// isSupportedVerb returns whether a APIResource supports a specific verb.
// The verb will be matched case-insensitive.
func isSupportedVerb(apiResource *metav1.APIResource, verb string) bool {
	if verb == "" || verb == "*" {
		return true
	}
	for _, v := range apiResource.Verbs {
		if strings.EqualFold(v, verb) {
			return true
		}
	}
	return false
}

func ToResourceInterface(dynamicIf dynamic.Interface, apiResource *metav1.APIResource, resource schema.GroupVersionResource, namespace string) dynamic.ResourceInterface {
	if apiResource.Namespaced {
		return dynamicIf.Resource(resource).Namespace(namespace)
	}
	return dynamicIf.Resource(resource)
}

// ApplyResource performs an apply of unstructured resource
func (k *KubectlResourceOperations) ApplyResource(obj *unstructured.Unstructured, force, validate, serverSideApply bool, manager string) (string, error) {
	log.Info(fmt.Sprintf("Applying resource %s/%s in cluster: %s, namespace: %s", obj.GetKind(), obj.GetName(), k.config.Host, obj.GetNamespace()))

	return k.runResourceCommand(obj, func(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) error {
		return k.newApplyOptions(obj, validate, force, serverSideApply, manager)
	})
}

func (k *KubectlResourceOperations) newApplyOptions(obj *unstructured.Unstructured, validate bool, force, serverSideApply bool, manager string) error {
	cmd := apply.NewCmdApply("kubectl", k.fact, k.streams)
	flags := apply.NewApplyFlags(k.fact, k.streams)
	o, err := flags.ToOptions(cmd, "kubectl", []string{})
	if err != nil {
		return err
	}
	o.Builder = k.fact.NewBuilder().Stream(k.streams.In, "input")

	return o.Run()
}

func (k *KubectlResourceOperations) newCreateOptions() (*create.CreateOptions, error) {
	o := create.NewCreateOptions(k.streams)

	recorder, err := o.RecordFlags.ToRecorder()
	if err != nil {
		return nil, err
	}
	o.Recorder = recorder
	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return nil, err
	}
	o.PrintObj = func(obj runtime.Object) error {
		return printer.PrintObj(obj, o.Out)
	}
	return o, nil
}

func (k *KubectlResourceOperations) newReplaceOptions(obj *unstructured.Unstructured, force bool) (*replace.ReplaceOptions, error) {
	o := replace.NewReplaceOptions(k.streams)

	recorder, err := o.RecordFlags.ToRecorder()
	if err != nil {
		return nil, err
	}
	o.Recorder = recorder

	o.DeleteOptions, err = o.DeleteFlags.ToOptions(k.dynamicClient, o.IOStreams)
	if err != nil {
		return nil, err
	}

	o.Builder = func() *resource.Builder {
		return k.fact.NewBuilder()
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return nil, err
	}
	o.PrintObj = func(obj runtime.Object) error {
		return printer.PrintObj(obj, o.Out)
	}

	o.Namespace = obj.GetNamespace()
	o.DeleteOptions.ForceDeletion = force
	return o, nil
}

func (k *KubectlResourceOperations) newReconcileOptions(kubeClient *kubernetes.Clientset, obj *unstructured.Unstructured) (*auth.ReconcileOptions, error) {
	o := auth.NewReconcileOptions(k.streams)
	o.RBACClient = kubeClient.RbacV1()
	o.NamespaceClient = kubeClient.CoreV1()
	jsonObj, err := obj.MarshalJSON()
	if err != nil {
		return o, err
	}
	r := k.fact.NewBuilder().
		WithScheme(runtime.NewScheme(), scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(obj.GetNamespace()).DefaultNamespace().
		Stream(bytes.NewBuffer(jsonObj), obj.GetName()).Flatten().Do()
	o.Visitor = r

	if o.DryRun {
		err := o.PrintFlags.Complete("%s (dry run)")
		if err != nil {
			return nil, err
		}
	}
	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return nil, err
	}
	o.PrintObject = printer.PrintObj
	return o, nil
}

func (k *KubectlResourceOperations) authReconcile(obj *unstructured.Unstructured) (string, error) {
	kubeClient, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}
	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	reconcileOpts, err := k.newReconcileOptions(kubeClient, obj)
	if err != nil {
		return "", fmt.Errorf("error calling newReconcileOptions: %w", err)
	}
	err = reconcileOpts.Validate()
	if err != nil {
		return "", err
	}
	err = reconcileOpts.RunReconcile()
	if err != nil {
		return "", err
	}

	var out []string
	if buf := strings.TrimSpace(ioStreams.Out.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	if buf := strings.TrimSpace(ioStreams.ErrOut.(*bytes.Buffer).String()); len(buf) > 0 {
		out = append(out, buf)
	}
	return strings.Join(out, ". "), nil
}
