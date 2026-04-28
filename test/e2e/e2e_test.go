/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/kyma-project/rt-bootstrapper/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// namespace where the project is deployed in
const namespace = "kyma-system"

const (
	testNamespace1 = "rt-bootstrapper-test1"
	testNamespace2 = "rt-bootstrapper-test2"
)

// serviceAccountName created for the project
const serviceAccountName = "rt-bootstrapper-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "rt-bootstrapper-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "rt-bootstrapper-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By(fmt.Sprintf("creating test namespace: %s", testNamespace1))
		cmd = exec.Command("kubectl", "create", "ns", testNamespace1)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By(fmt.Sprintf("creating test namespace: %s", testNamespace2))
		cmd = exec.Command("kubectl", "create", "ns", testNamespace2)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By(fmt.Sprintf("opt out altering image registry for namespace: %s", testNamespace2))
		cmd = exec.Command("kubectl", "annotate", "ns", testNamespace2,
			"rt-cfg.kyma-project.io/alter-img-registry=true",
			"rt-cfg.kyma-project.io/add-img-pull-secret=true",
			"rt-cfg.kyma-project.io/add-cluster-trust-bundle=true",
			"rt-cfg.kyma-project.io/set-fips-mode=true",
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("creating master-password secret")
		cmd = exec.Command("kubectl", "create", "secret", "docker-registry", "registry-credentials",
			"--docker-username=admin",
			"--docker-password=password123",
			"--docker-server=localhost:5001",
			"-n", namespace)

		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create master-password secret")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("patching egress network policy to allow k3d apiserver port 6443")
		Eventually(func(g Gomega) {
			cmd = exec.Command("kubectl", "patch", "networkpolicy",
				"rt-bootstrapper-kyma-project.io--rt-bootstrapper--allow-egress-to-apiserver",
				"-n", namespace,
				"--type=json",
				`-p=[{"op":"replace","path":"/spec/egress/0/ports/0/port","value":6443}]`,
			)
			_, err = utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to patch egress network policy")
		}).Should(Succeed())
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=rt-bootstrapper-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring(":8080"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"metadata": {
						"labels": {
							"networking.kyma-project.io/metrics-scraping": "allowed"
						}
					},
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v http://%s.%s.svc.cluster.local:8080/metrics"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 3*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		It("should provisioned cert-manager", func() {
			By("validating that cert-manager has the certificate Secret")
			verifyCertManager := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "rt-bootstrapper-webhook-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertManager).Should(Succeed())
		})

		It("should have CA injection for mutating webhooks", func() {
			By("checking CA injection for mutating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"mutatingwebhookconfigurations.admissionregistration.k8s.io",
					"rt-bootstrapper-mutating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				mwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		It("should provisioned cluster-trust-bundle", func() {
			By("checking rt-bootstrapper-k3d.test:ctb:1")
			cmd := exec.Command("kubectl", "get",
				"clustertrustbundles.certificates.k8s.io",
				"rt-bootstrapper-k3d.test:ctb:1",
				"-o", "go-template={{ .spec.signerName }}")
			signerName, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(signerName).To(Equal("rt-bootstrapper-k3d.test/ctb"))
		})

		It("should work with all features activated on ns lvl", func() {
			By("applying the deployment in opt in namespace")
			cmd := exec.Command("kubectl", "apply",
				"-f", "./test/e2e/testdata/test2.yaml",
				"-n", testNamespace2)

			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "wait", "deployment.apps/pause-test2",
				"--for", "condition=Available",
				"--namespace", testNamespace2,
				"--timeout", "20s",
			)

			_, err = utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("kubectl", "get", "pod",
				"-l", "app=pause-test2",
				"-n", testNamespace2,
				"-o", "jsonpath={.items[0]}")
			output, err := utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			pod, err := utils.ToPod(output)
			Expect(err).ShouldNot(HaveOccurred())

			By("having registry name replaced on pod")
			Expect(pod.Spec.Containers[0].Image).ShouldNot(HavePrefix("replace.me"))

			By("having image-pull-secret added on pod")
			Expect(pod.Spec.ImagePullSecrets).Should(ContainElement(corev1.LocalObjectReference{
				Name: "registry-credentials",
			}))

			By("having cluster-trust-bundle volume mounted on pod")
			Expect(pod.Spec.Containers[0].VolumeMounts).Should(ContainElement(corev1.VolumeMount{
				Name:      "rt-bootstrapper-certs",
				ReadOnly:  true,
				MountPath: "/etc/ssl/certs",
			}))

			By("having cluster-trust-bundle volume created on pod")
			Expect(pod.Spec.Volumes[1].VolumeSource.Projected.Sources).Should(ContainElement(corev1.VolumeProjection{
				ClusterTrustBundle: &corev1.ClusterTrustBundleProjection{
					Name: ptr.To("rt-bootstrapper-k3d.test:ctb:1"),
					Path: "kube-apiserver-serving.pem",
				},
			}))

			By("having KYMA_FIPS_MODE_ENABLED env var set on all containers")
			allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)
			for _, container := range allContainers {
				Expect(container.Env).Should(ContainElement(corev1.EnvVar{
					Name:  apiv1.EnvKymaFipsModeEnabled,
					Value: "true",
				}))
			}

			By("having 'modified' annotation added on pod")
			Expect(pod.Annotations[apiv1.AnnotationModified]).Should(Equal("true"))
		})

		It("should work with all features activated on pod lvl", func() {
			By("applying the deployment in opt in namespace")
			cmd := exec.Command("kubectl", "apply",
				"-f", "./test/e2e/testdata/test1.yaml",
				"-n", testNamespace1)

			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "wait", "deployment.apps/pause-test1",
				"--for", "condition=Available",
				"--namespace", testNamespace1,
				"--timeout", "20s",
			)

			_, err = utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("kubectl", "get", "pod",
				"-l", "app=pause-test1",
				"-n", testNamespace1,
				"-o", "jsonpath={.items[0]}")
			output, err := utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			pod, err := utils.ToPod(output)
			Expect(err).ShouldNot(HaveOccurred())

			By("having registry name replaced on pod")
			Expect(pod.Spec.Containers[0].Image).ShouldNot(HavePrefix("replace.me"))

			By("having image-pull-secret added on pod")
			Expect(pod.Spec.ImagePullSecrets).Should(ContainElement(corev1.LocalObjectReference{
				Name: "registry-credentials",
			}))

			By("having cluster-trust-bundle volume mounted on pod")
			Expect(pod.Spec.Containers[0].VolumeMounts).Should(ContainElement(corev1.VolumeMount{
				Name:      "rt-bootstrapper-certs",
				ReadOnly:  true,
				MountPath: "/etc/ssl/certs",
			}))

			By("having cluster-trust-bundle volume created on pod")
			Expect(pod.Spec.Volumes[1].VolumeSource.Projected.Sources).Should(ContainElement(corev1.VolumeProjection{
				ClusterTrustBundle: &corev1.ClusterTrustBundleProjection{
					Name: ptr.To("rt-bootstrapper-k3d.test:ctb:1"),
					Path: "kube-apiserver-serving.pem",
				},
			}))

			By("having 'modified' annotation added on pod")
			Expect(pod.Annotations[apiv1.AnnotationModified]).Should(Equal("true"))

			By("having KYMA_FIPS_MODE_ENABLED env var set on all containers")
			allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)
			for _, container := range allContainers {
				Expect(container.Env).Should(ContainElement(corev1.EnvVar{
					Name:  apiv1.EnvKymaFipsModeEnabled,
					Value: "true",
				}))
			}
		})

		It("should work with all features inactive", func() {
			By("applying the deployment in opt in namespace")
			cmd := exec.Command("kubectl", "apply",
				"-f", "./test/e2e/testdata/test3.yaml",
				"-n", testNamespace1)

			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "wait", "deployment.apps/pause-test3",
				"--for", "condition=Available",
				"--namespace", testNamespace1,
				"--timeout", "20s",
			)

			_, err = utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			cmd = exec.Command("kubectl", "get", "pod",
				"-l", "app=pause-test3",
				"-n", testNamespace1,
				"-o", "jsonpath={.items[0]}")
			output, err := utils.Run(cmd)
			Expect(err).ShouldNot(HaveOccurred())

			pod, err := utils.ToPod(output)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(pod.Spec.Containers[0].Image).Should(HavePrefix("localhost:5002"))
			Expect(pod.Annotations[apiv1.AnnotationModified]).ShouldNot(Equal("true"))
			Expect(pod.Spec.Containers[0].VolumeMounts).ShouldNot(ContainElement(corev1.VolumeMount{
				Name:      "rt-bootstrapper-certs",
				ReadOnly:  true,
				MountPath: "/etc/ssl/certs",
			}))

			By("not having cluster-trust-bundle volume created on pod")
			Expect(len(pod.Spec.Volumes)).Should(Equal(1))

			By("not having 'modified' annotation added on pod")
			Expect(pod.Annotations[apiv1.AnnotationModified]).Should(BeEmpty())

			By("not havling KYMA_FIPS_MODE_ENABLED env var set on all containers")
			allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)
			for _, container := range allContainers {
				Expect(container.Env).ShouldNot(ContainElement(corev1.EnvVar{
					Name:  apiv1.EnvKymaFipsModeEnabled,
					Value: "true",
				}))
			}
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
