package alerts

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type MetricReconciler interface {
	Kind() string
	ResourceName() string
	GetFullResource() client.Object
	EmptyObject() client.Object
	UpdateExistingResource(context.Context, client.Client, client.Object, logr.Logger) (client.Object, bool, error)
}

type MonitoringReconciler struct {
	reconcilers   []MetricReconciler
	scheme        *runtime.Scheme
	latestObjects []client.Object
	client        client.Client
	namespace     string
	eventEmitter  hcoutil.EventEmitter
}

var logger = logf.Log.WithName("hyperconverged-operator-monitoring-reconciler")

func NewMonitoringReconciler(ci hcoutil.ClusterInfo, cl client.Client, ee hcoutil.EventEmitter, scheme *runtime.Scheme) *MonitoringReconciler {
	deployment := ci.GetDeployment()

	var (
		namespace string
		owner     metav1.OwnerReference
	)

	if deployment == nil {
		var err error
		namespace, err = hcoutil.GetOperatorNamespace(logger)
		if err != nil {
			return nil
		}
	} else {
		namespace = deployment.Namespace
		owner = getDeploymentReference(deployment)
	}

	return &MonitoringReconciler{
		reconcilers:  getReconcilers(ci, namespace, owner),
		scheme:       scheme,
		client:       cl,
		namespace:    namespace,
		eventEmitter: ee,
	}
}

func getReconcilers(ci hcoutil.ClusterInfo, namespace string, owner metav1.OwnerReference) []MetricReconciler {
	alertRuleReconciler, err := newAlertRuleReconciler(namespace, owner)
	if err != nil {
		logger.Error(err, "failed to create the 'PrometheusRule' reconciler")
	}

	reconcilers := []MetricReconciler{
		alertRuleReconciler,
		newRoleReconciler(namespace, owner),
		newRoleBindingReconciler(namespace, owner, ci),
		newMetricServiceReconciler(namespace, owner),
		newSecretReconciler(namespace, owner),
		newServiceMonitorReconciler(namespace, owner),
	}

	if shouldDeployNetworkPolicy(ci) {
		reconcilers = append(reconcilers, newAlertManagerNetworkPolicyReconciler(namespace, owner, ci))
	}

	return reconcilers
}

func (r *MonitoringReconciler) Reconcile(req *common.HcoRequest, firstLoop bool) error {
	if r == nil {
		return nil
	}

	if err := reconcileNamespace(req.Ctx, r.client, r.namespace, req.Logger); err != nil {
		return err
	}

	objects := make([]client.Object, len(r.reconcilers))

	for i, rc := range r.reconcilers {
		obj, err := r.ReconcileOneResource(req, rc, firstLoop)
		if err != nil {
			return err
		} else if obj != nil {
			objects[i] = obj
		}
	}

	r.latestObjects = objects
	return nil
}

func (r *MonitoringReconciler) ReconcileOneResource(req *common.HcoRequest, reconciler MetricReconciler, firstLoop bool) (client.Object, error) {
	if r == nil {
		return nil, nil // not initialized (not running on openshift). do nothing
	}

	req.Logger.V(5).Info(fmt.Sprintf("Reconciling the %s", reconciler.Kind()))

	existing := reconciler.EmptyObject()

	req.Logger.V(5).Info(fmt.Sprintf("Reading the current %s", reconciler.Kind()))
	err := r.client.Get(req.Ctx, client.ObjectKey{Namespace: r.namespace, Name: reconciler.ResourceName()}, existing)

	if err != nil {
		if errors.IsNotFound(err) {
			req.Logger.Info(fmt.Sprintf("Can't find the %s; creating a new one", reconciler.Kind()), "name", reconciler.ResourceName())
			required := reconciler.GetFullResource()
			err := r.client.Create(req.Ctx, required)
			if err != nil {
				req.Logger.Error(err, fmt.Sprintf("failed to create %s", reconciler.Kind()))
				r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to create the %s %s", reconciler.ResourceName(), reconciler.Kind()))
				return nil, err
			}
			req.Logger.Info(fmt.Sprintf("successfully created the %s", reconciler.Kind()), "name", reconciler.ResourceName())
			r.eventEmitter.EmitEvent(required, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", reconciler.Kind(), reconciler.ResourceName()))

			return required, nil
		}

		req.Logger.Error(err, "unexpected error while reading the PrometheusRule")
		return nil, err
	}

	resource, updated, err := reconciler.UpdateExistingResource(req.Ctx, r.client, existing, req.Logger)
	if err != nil {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to update the %s %s", reconciler.ResourceName(), reconciler.Kind()))
		return nil, err
	}

	if updated {
		r.handleUpdatedResource(req, reconciler, firstLoop)
	}

	return resource, nil
}

func (r *MonitoringReconciler) handleUpdatedResource(req *common.HcoRequest, reconciler MetricReconciler, firstLoop bool) {
	if req.HCOTriggered {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", reconciler.Kind(), reconciler.ResourceName()))
	} else {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "Overwritten", fmt.Sprintf("Overwritten %s %s", reconciler.Kind(), reconciler.ResourceName()))
		if !firstLoop && !req.UpgradeMode {
			metrics.IncOverwrittenModifications(reconciler.Kind(), reconciler.ResourceName())
		}
	}
}

func (r *MonitoringReconciler) UpdateRelatedObjects(req *common.HcoRequest) error {
	if r == nil {
		return nil // not initialized (not running on openshift). do nothing
	}

	wasChanged := false
	for _, obj := range r.latestObjects {
		changed, err := hcoutil.AddCrToTheRelatedObjectList(&req.Instance.Status.RelatedObjects, obj, r.scheme)
		if err != nil {
			return err
		}

		wasChanged = wasChanged || changed
	}

	if wasChanged {
		req.StatusDirty = true
	}

	return nil
}

func getDeploymentReference(deployment *appsv1.Deployment) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         appsv1.SchemeGroupVersion.String(),
		Kind:               "Deployment",
		Name:               deployment.GetName(),
		UID:                deployment.GetUID(),
		BlockOwnerDeletion: ptr.To(false),
		Controller:         ptr.To(false),
	}
}

// update the labels and the ownerReferences in a metric resource
// return true if something was changed
func updateCommonDetails(required, existing *metav1.ObjectMeta) bool {
	if reflect.DeepEqual(required.OwnerReferences, existing.OwnerReferences) &&
		hcoutil.CompareLabels(required, existing) {
		return false
	}

	hcoutil.MergeLabels(required, existing)
	if reqLen := len(required.OwnerReferences); reqLen > 0 {
		refs := make([]metav1.OwnerReference, reqLen)
		for i, ref := range required.OwnerReferences {
			ref.DeepCopyInto(&refs[i])
		}
		existing.OwnerReferences = refs
	} else {
		existing.OwnerReferences = nil
	}

	return true
}

func shouldDeployNetworkPolicy(ci hcoutil.ClusterInfo) bool {
	selfPod := ci.GetPod()
	if selfPod == nil {
		return false
	}
	_, lbl1 := selfPod.Labels[hcoutil.AllowEgressToDNSAndAPIServerLabel]
	_, lbl2 := selfPod.Labels[hcoutil.AllowIngressToMetricsEndpointLabel]

	return lbl1 || lbl2
}
