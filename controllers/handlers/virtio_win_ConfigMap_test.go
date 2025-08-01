package handlers

import (
	"context"
	"maps"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("VirtioWin", func() {

	var testLogger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("VirtioWin_test")

	Context("Virtio-Win ConfigMap", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		It("should error if VIRTIOWIN_CONTAINER environment var not specified", func() {
			os.Unsetenv("VIRTIOWIN_CONTAINER")

			cl := commontestutils.InitClient([]client.Object{})
			handler, err := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)

			Expect(err).To(HaveOccurred())
			Expect(handler).To(BeNil())
		})

		It("should create if not present", func() {
			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{})
			handler, _ := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).To(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler, _ := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commontestutils.GetScheme(), expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile according to env values and HCO CR", func() {
			virtiowink := "virtio-win-image"
			updatableKeys := [...]string{virtiowink}
			toBeRemovedKey := "toberemoved"

			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			outdatedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			// values we should update
			outdatedResource.Data[virtiowink] = "old-virtiowin-container-value-we-have-to-update"

			// add values we should remove
			outdatedResource.Data[toBeRemovedKey] = "value-we-should-remove"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler, _ := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for _, k := range updatableKeys {
				Expect(foundResource.Data[k]).To(Not(Equal(outdatedResource.Data[k])))
				Expect(foundResource.Data[k]).To(Equal(expectedResource.Data[k]))
			}

			Expect(foundResource.Data).To(Not(HaveKey(toBeRemovedKey)))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commontestutils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler, _ := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler, _ := NewVirtioWinCmHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

	})

	Context("ConfigMap Reader Role", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})
		It("should do nothing if exists", func() {
			expectedRole := NewVirtioWinCmReaderRole(hco)
			cl := commontestutils.InitClient([]client.Object{hco, expectedRole})

			handler, _ := NewVirtioWinCmReaderRoleHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRole := &rbacv1.Role{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
					foundRole),
			).ToNot(HaveOccurred())

			Expect(expectedRole.ObjectMeta).To(Equal(foundRole.ObjectMeta))
			Expect(expectedRole.Rules).To(Equal(foundRole.Rules))
		})

		It("should update if labels are missing", func() {
			expectedRole := NewVirtioWinCmReaderRole(hco)
			expectedLabels := expectedRole.Labels
			expectedRole.Labels = nil

			cl := commontestutils.InitClient([]client.Object{hco, expectedRole})

			handler, _ := NewVirtioWinCmReaderRoleHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRole := &rbacv1.Role{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
					foundRole),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedLabels, foundRole.Labels)).To(BeTrue())
		})
	})

	Context("ConfigMap Reader Role Binding", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})
		It("should do nothing if exists", func() {
			expectedRoleBinding := NewVirtioWinCmReaderRoleBinding(hco)

			cl := commontestutils.InitClient([]client.Object{hco, expectedRoleBinding})

			handler, _ := NewVirtioWinCmReaderRoleBindingHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRoleBinding := &rbacv1.RoleBinding{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
					foundRoleBinding),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedRoleBinding.Labels, foundRoleBinding.Labels)).To(BeTrue())
		})

		It("should update if labels are missing", func() {
			expectedRoleBinding := NewVirtioWinCmReaderRoleBinding(hco)
			expectedLabels := expectedRoleBinding.Labels
			expectedRoleBinding.Labels = nil

			cl := commontestutils.InitClient([]client.Object{hco, expectedRoleBinding})

			handler, _ := NewVirtioWinCmReaderRoleBindingHandler(testLogger, cl, commontestutils.GetScheme(), hco)
			res := handler.Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRoleBinding := &rbacv1.RoleBinding{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
					foundRoleBinding),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedLabels, foundRoleBinding.Labels)).To(BeTrue())
		})
	})
})
