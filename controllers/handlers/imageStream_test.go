package handlers

import (
	"context"
	"maps"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	imagev1 "github.com/openshift/api/image/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("imageStream tests", func() {

	schemeForTest := commontestutils.GetScheme()

	var (
		testFilesLocation = getTestFilesLocation() + "/imageStreams"
		hco               *hcov1beta1.HyperConverged
		testLogger        = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("imagestream_test")
	)

	BeforeEach(func() {
		hco = commontestutils.NewHco()

		envVal, envDefined := os.LookupEnv(ImageStreamManifestLocationVarName)
		Expect(os.Setenv(ImageStreamManifestLocationVarName, testFilesLocation)).To(Succeed())

		DeferCleanup(func() {
			if envDefined {
				Expect(os.Setenv(ImageStreamManifestLocationVarName, envVal)).To(Succeed())
			} else {
				Expect(os.Unsetenv(ImageStreamManifestLocationVarName)).To(Succeed())
			}
		})
	})

	Context("test imageStreamHandler", func() {
		It("should not create the ImageStream resource if the FG is not set", func() {
			hco.Spec.EnableCommonBootImageImport = ptr.To(false)

			cli := commontestutils.InitClient([]client.Object{})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
			Expect(imageStreamObjects.Items).To(BeEmpty())
		})

		It("should delete the ImageStream resource if the FG is not set", func() {
			hco.Spec.EnableCommonBootImageImport = ptr.To(false)

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
					UID:       types.UID("0987654321"),
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: true, Scheduled: false},
							Name:         "latest",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			ref, err := reference.GetReference(commontestutils.GetScheme(), exists)
			Expect(err).ToNot(HaveOccurred())
			Expect(objectreferencesv1.SetObjectReference(&hco.Status.RelatedObjects, *ref)).To(Succeed())

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())

			imageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
			Expect(imageStreamObjects.Items).To(BeEmpty())

			newRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(newRef).To(BeNil())
		})

		It("should create the ImageStream resource if not exists", func() {
			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)
			cli := commontestutils.InitClient([]client.Object{hco})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			req := commontestutils.NewReq(hco)
			res := handlers[0].Ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeTrue())

			ImageStreamObjects := &imagev1.ImageStreamList{}
			Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
			Expect(ImageStreamObjects.Items).To(HaveLen(1))
			Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
		})

		It("should update the ImageStream resource if the docker image was changed", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
							},
							Name: "latest",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)
			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should update the ImageStream resource if the tag name was changed", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
							},
							Name: "old",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should remove tags if they are not required", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							Name: "latest",
						},
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
							},
							Name: "old",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).ToNot(BeEmpty())

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if the tag name and the from.name fields are the same", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Scheduled: true},
							Name:         "latest",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if the it not controlled by HCO (even if the details are not the same)", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/old-test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: true, Scheduled: false},
							Name:         "old",
						},
					},
				},
			}

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("old"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/old-test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: true, Scheduled: false}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should not update the imageStream if nothing has changed", func() {
			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
								UID:  types.UID("1234567890"),
							},
							ImportPolicy: imagev1.TagImportPolicy{Insecure: false, Scheduled: true},
							Name:         "latest",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeFalse()) // <=== should not update the imageStream

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))
				// check that this tag was not changed by the handler, by checking a field that is not controlled by it.
				Expect(tag.From.UID).To(Equal(types.UID("1234567890")))
				Expect(tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Insecure: false, Scheduled: true}))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefOutdated))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})

		It("should update the ImageStream labels", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"

			exists := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-image-stream",
					Namespace: "test-image-stream-ns",
				},

				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Kind: "DockerImage",
								Name: "test-registry.io/test/test-image",
							},
							Name: "latest",
						},
					},
				},
			}
			exists.Labels = operands.GetLabels(hco, util.AppComponentCompute)
			expectedLabels := maps.Clone(exists.Labels)
			exists.Labels[userLabelKey] = userLabelValue
			for k, v := range expectedLabels {
				exists.Labels[k] = "wrong_" + v
			}
			exists.Labels[util.AppLabelManagedBy] = expectedLabels[util.AppLabelManagedBy]

			cli := commontestutils.InitClient([]client.Object{exists})
			handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(imageStreamNames).To(ContainElement("test-image-stream"))

			hco := commontestutils.NewHco()
			hco.Spec.EnableCommonBootImageImport = ptr.To(true)

			By("apply the ImageStream CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				imageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), imageStreamObjects)).To(Succeed())
				Expect(imageStreamObjects.Items).To(HaveLen(1))

				is := imageStreamObjects.Items[0]

				Expect(is.Name).To(Equal("test-image-stream"))
				// check that the existing object was reconciled
				Expect(is.Spec.Tags).To(HaveLen(1))
				tag := is.Spec.Tags[0]
				Expect(tag.Name).To(Equal("latest"))
				Expect(tag.From.Name).To(Equal("test-registry.io/test/test-image"))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &imageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

				for k, v := range expectedLabels {
					Expect(is.Labels).To(HaveKeyWithValue(k, v))
				}
				Expect(is.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
			})
		})

		Context("imagestream namespace", func() {
			const customNS = "custom-ns"
			It("should create imagestream in a custom namespace", func() {
				hco := commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))
			})

			It("should delete an imagestream from one namespace, and create it in another one", func() {
				By("create imagestream in the default namespace")
				hco := commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal("test-image-stream-ns"))

				ref, err := reference.GetReference(commontestutils.GetScheme(), &ImageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())

				By("replace the image stream with a new one in the custom namespace")
				hco = commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)
				Expect(objectreferencesv1.SetObjectReference(&hco.Status.RelatedObjects, *ref)).To(Succeed())

				req = commontestutils.NewReq(hco)
				res = handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))

				newRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *ref)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRef).To(BeNil())
			})

			It("should remove an imagestream from a custom namespace, and create it in the default one", func() {
				By("create imagestream in a custom namespace")
				hco := commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))

				ref, err := reference.GetReference(commontestutils.GetScheme(), &ImageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())

				By("replace the image stream with a new one in the default namespace")
				hco = commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				Expect(objectreferencesv1.SetObjectReference(&hco.Status.RelatedObjects, *ref)).To(Succeed())

				req = commontestutils.NewReq(hco)
				res = handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal("test-image-stream-ns"))

				newRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *ref)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRef).To(BeNil())
			})

			It("should remove an imagestream from a custom namespace, and create it in the new custom namespace", func() {
				By("create imagestream in a custom namespace")
				hco := commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

				cli := commontestutils.InitClient([]client.Object{hco})
				handlers, err := GetImageStreamHandlers(testLogger, cli, schemeForTest, hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(imageStreamNames).To(ContainElement("test-image-stream"))

				req := commontestutils.NewReq(hco)
				res := handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects := &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS))

				ref, err := reference.GetReference(commontestutils.GetScheme(), &ImageStreamObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())

				By("replace the image stream with a new one in another custom namespace")
				hco = commontestutils.NewHco()
				hco.Spec.EnableCommonBootImageImport = ptr.To(true)
				hco.Spec.CommonBootImageNamespace = ptr.To(customNS + "1")
				Expect(objectreferencesv1.SetObjectReference(&hco.Status.RelatedObjects, *ref)).To(Succeed())

				req = commontestutils.NewReq(hco)
				res = handlers[0].Ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				ImageStreamObjects = &imagev1.ImageStreamList{}
				Expect(cli.List(context.TODO(), ImageStreamObjects)).To(Succeed())
				Expect(ImageStreamObjects.Items).To(HaveLen(1))
				Expect(ImageStreamObjects.Items[0].Name).To(Equal("test-image-stream"))
				Expect(ImageStreamObjects.Items[0].Namespace).To(Equal(customNS + "1"))

				newRef, err := objectreferencesv1.FindObjectReference(hco.Status.RelatedObjects, *ref)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRef).To(BeNil())
			})
		})
	})

	Context("test compareAndUpgradeImageStream", func() {
		required := &imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testStream",
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v1",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "v1",
					},
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v2",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "v2",
					},
					{
						From: &corev1.ObjectReference{
							Name: "my-image-registry:5000/my-image:v2",
							Kind: "DockerImage",
						},
						ImportPolicy: imagev1.TagImportPolicy{
							Scheduled: true,
						},
						Name: "latest",
					},
				},
			},
		}

		hook := newIsHook(required, required.Namespace)

		It("should do nothing if there is no difference", func() {
			found := required.DeepCopy()

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeFalse())

			Expect(found.Spec.Tags).To(HaveLen(3))

			validateImageStream(found, hook)
		})

		It("should add all tag if missing", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
			}

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should add missing tags", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v1",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
					},
				},
			}

			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should delete unknown tags", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v1",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v2",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v3",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v3",
						},
						{
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v3",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
					},
				},
			}
			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)
		})

		It("should fix tag from and import policy, but leave the rest", func() {
			found := &imagev1.ImageStream{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testStream",
				},
				Spec: imagev1.ImageStreamSpec{
					Tags: []imagev1.TagReference{
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v45",
								Kind: "somethingElse",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "v1",
						},
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: false,
							},
							Name: "v2",
						},
						{
							Annotations: map[string]string{
								"test-annotation": "should stay here",
							},
							From: &corev1.ObjectReference{
								Name: "my-image-registry:5000/my-image:v2",
								Kind: "DockerImage",
							},
							ImportPolicy: imagev1.TagImportPolicy{
								Scheduled: true,
							},
							Name: "latest",
						},
					},
				},
			}
			Expect(hook.compareAndUpgradeImageStream(found)).To(BeTrue())

			validateImageStream(found, hook)

			for _, tag := range found.Spec.Tags {
				Expect(tag.Annotations).To(HaveLen(1))
				Expect(tag.Annotations).To(HaveKeyWithValue("test-annotation", "should stay here"))
			}

		})
	})
})

func validateImageStream(found *imagev1.ImageStream, hook *isHooks) {
	ExpectWithOffset(1, found.Spec.Tags).To(HaveLen(3))

	validationTagMap := map[string]bool{
		"v1":     false,
		"v2":     false,
		"latest": false,
	}

	for i := range 3 {
		tagName := found.Spec.Tags[i].Name
		tag := getTagByName(found.Spec.Tags, tagName)
		Expect(tag).ToNot(BeNil())
		validationTagMap[tagName] = true

		ExpectWithOffset(1, tag.From).To(Equal(hook.tags[tagName].From))
		ExpectWithOffset(1, tag.ImportPolicy).To(Equal(imagev1.TagImportPolicy{Scheduled: true}))
	}

	ExpectWithOffset(1, validateAllTags(validationTagMap)).To(BeTrue())
}

func getTagByName(tags []imagev1.TagReference, name string) *imagev1.TagReference {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag
		}
	}
	return nil
}

func validateAllTags(m map[string]bool) bool {
	for _, toughed := range m {
		if !toughed {
			return false
		}
	}
	return true
}
