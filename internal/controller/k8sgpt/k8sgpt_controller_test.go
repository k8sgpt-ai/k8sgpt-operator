/*
	Copyright 2023 The K8sGPT Authors.
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

package k8sgpt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TIMEOUT      = time.Second * time.Duration(60)
	POLLINTERVAL = time.Second * time.Duration(1)
	DURATION     = time.Second * time.Duration(10)
)

var _ = Describe("K8sGPT controller", func() {
	SetDefaultEventuallyTimeout(TIMEOUT)
	SetDefaultEventuallyPollingInterval(POLLINTERVAL)
	SetDefaultConsistentlyDuration(DURATION)

	Describe("Create a new Analysis", Label("integration"), func() {
		Context("when the analysis doesn't exist beforehand", func() {
			k8sgpt := corev1alpha1.GetValidProjectResource("my-test", "default")
			nn := types.NamespacedName{
				Namespace: k8sgpt.Namespace,
				Name:      k8sgpt.Name,
			}
			It("Should create a fake secret", func() {
				Expect(k8sClient.Create(ctx, createFakeSecret(k8sgpt.Spec.AI.Secret.Name, "default"))).Should(Succeed())
			})

			It("Should create a k8sgpt serviceAccount", func() {
				Expect(k8sClient.Create(ctx, createServiceAccount("default"))).Should(Succeed())
			})

			It("Should create CR", func() {
				Expect(k8sClient.Create(ctx, &k8sgpt)).Should(Succeed())
			})

			It("Should K8SGPT have a finalizer", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					if len(k.Finalizers) == 0 {
						return errors.New("k8sgpt doesnt have finalizer")
					}

					return nil
				}).Should(BeNil())
			})

			It("Should deploy k8sgpt server in current namespace", func() {
				Eventually(func() error {
					deployment := v1.Deployment{}
					err := k8sClient.Get(ctx, nn, &deployment)
					if err != nil {
						return err
					}

					if *deployment.Spec.Replicas == 0 {
						return errors.New("deployment is not correct")
					}

					return nil
				}).Should(BeNil())
			})
		})

		Context("when configuring analysis interval", func() {
			k8sgpt := corev1alpha1.GetValidProjectResource("interval-test", "default")
			k8sgpt.Spec.Analysis = &corev1alpha1.AnalysisConfig{
				Interval: "5m",
			}
			nn := types.NamespacedName{
				Namespace: k8sgpt.Namespace,
				Name:      k8sgpt.Name,
			}

			It("Should create CR with custom interval", func() {
				Expect(k8sClient.Create(ctx, &k8sgpt)).Should(Succeed())
			})

			It("Should parse and use custom interval", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					if k.Spec.Analysis == nil || k.Spec.Analysis.Interval != "5m" {
						return errors.New("analysis interval not set correctly")
					}

					return nil
				}).Should(BeNil())
			})

			It("Should handle invalid interval gracefully", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					// Update with invalid interval
					k.Spec.Analysis.Interval = "invalid"
					err = k8sClient.Update(ctx, &k)
					if err == nil {
						return errors.New("expected validation error for invalid interval")
					}

					// Verify the error is a validation error
					if !strings.Contains(err.Error(), "spec.analysis.interval in body should match '^[0-9]+[smh]$'") {
						return fmt.Errorf("unexpected error: %v", err)
					}

					// Verify the original interval is still set
					err = k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}
					if k.Spec.Analysis.Interval != "5m" {
						return fmt.Errorf("interval was changed to %s, expected 5m", k.Spec.Analysis.Interval)
					}

					return nil
				}).Should(BeNil())
			})

			It("Should handle different interval formats", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					// Test seconds
					k.Spec.Analysis.Interval = "30s"
					err = k8sClient.Update(ctx, &k)
					if err != nil {
						return err
					}

					// Test minutes
					k.Spec.Analysis.Interval = "5m"
					err = k8sClient.Update(ctx, &k)
					if err != nil {
						return err
					}

					// Test hours
					k.Spec.Analysis.Interval = "1h"
					err = k8sClient.Update(ctx, &k)
					if err != nil {
						return err
					}

					return nil
				}).Should(BeNil())
			})

			It("Should reject invalid interval formats", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					// Test invalid formats
					invalidFormats := []string{
						"invalid",
						"5",     // missing unit
						"5d",    // invalid unit
						"abc",   // no numbers
						"5m30s", // multiple units
						"5.5m",  // decimal
						"-5m",   // negative
					}

					for _, format := range invalidFormats {
						k.Spec.Analysis.Interval = format
						err = k8sClient.Update(ctx, &k)
						if err == nil {
							return fmt.Errorf("expected validation error for interval %s", format)
						}
						if !strings.Contains(err.Error(), "spec.analysis.interval in body should match '^[0-9]+[smh]$'") {
							return fmt.Errorf("unexpected error for interval %s: %v", format, err)
						}
					}

					return nil
				}).Should(BeNil())
			})
		})
	})

	Describe("Secret rotation triggers deployment rollout", Label("integration"), func() {
		It("updates the Deployment checksum when the referenced Secret changes", func() {
			secretName := "rotation-test-secret"
			k8sgptName := "secret-rotation-test"

			// Create Secret with initial data
			secret := createFakeSecret(secretName, "default")
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			// Create K8sGPT CR referencing the Secret.
			// Set a long analysis interval so the periodic reconcile cannot fire within
			// the test window — the checksum update must be driven by the Secret watch.
			k8sgpt := corev1alpha1.GetValidProjectResource(k8sgptName, "default")
			k8sgpt.Spec.AI.Secret.Name = secretName
			k8sgpt.Spec.Analysis = &corev1alpha1.AnalysisConfig{Interval: "5m"}
			Expect(k8sClient.Create(ctx, &k8sgpt)).Should(Succeed())

			nn := types.NamespacedName{
				Namespace: "default",
				Name:      k8sgptName,
			}

			var checksumBefore string

			// Wait for Deployment to be created with initial checksum
			Eventually(func() error {
				deployment := v1.Deployment{}
				if err := k8sClient.Get(ctx, nn, &deployment); err != nil {
					return err
				}

				checksum, exists := deployment.Spec.Template.Annotations["core.k8sgpt.ai/ai-secret-checksum"]
				if !exists {
					return errors.New("checksum annotation not found")
				}

				if len(checksum) != 64 {
					return fmt.Errorf("checksum should be 64 characters, got %d", len(checksum))
				}

				checksumBefore = checksum
				return nil
			}).Should(BeNil())

			// Rotate the Secret data
			secretNN := types.NamespacedName{
				Namespace: "default",
				Name:      secretName,
			}

			Eventually(func() error {
				updatedSecret := &corev1.Secret{}
				if err := k8sClient.Get(ctx, secretNN, updatedSecret); err != nil {
					return err
				}

				updatedSecret.Data["openai-api-key"] = []byte("rotated-key-value")
				return k8sClient.Update(ctx, updatedSecret)
			}).Should(BeNil())

			// Verify checksum changes after Secret rotation.
			// 10-second window: the 5m periodic reconcile cannot fire here, so only
			// the Secret watch event can drive this update.
			Eventually(func() error {
				deployment := v1.Deployment{}
				if err := k8sClient.Get(ctx, nn, &deployment); err != nil {
					return err
				}

				checksumAfter, exists := deployment.Spec.Template.Annotations["core.k8sgpt.ai/ai-secret-checksum"]
				if !exists {
					return errors.New("checksum annotation not found after rotation")
				}

				if checksumAfter == checksumBefore {
					return fmt.Errorf("checksum unchanged: expected different checksum after Secret rotation")
				}

				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(BeNil())
		})
	})
})

func createFakeSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"openai-api-key": []byte("fake-key"),
		},
	}
}

func createServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: namespace,
		},
	}
}
