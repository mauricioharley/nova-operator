/*
Copyright 2024.

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

package functional_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2" //revive:disable-line:dot-imports
	. "github.com/onsi/gomega"    //revive:disable-line:dot-imports
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
)

var _ = Describe("ApplicationCredential", func() {
	var (
		namespace string
	)

	BeforeEach(func() {
		namespace = uuid.New().String()
		th.CreateNamespace(namespace)
		DeferCleanup(th.DeleteNamespace, namespace)

		// Create KeystoneAPI
		keystoneAPIName := keystone.CreateKeystoneAPI(namespace)
		DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPIName)

		// Create MariaDB account and secret for testing
		apiAccount, apiSecret := mariadb.CreateMariaDBAccountAndSecret(
			types.NamespacedName{Name: "nova-api-account", Namespace: namespace},
			mariadbv1.MariaDBAccountSpec{})
		DeferCleanup(k8sClient.Delete, ctx, apiAccount)
		DeferCleanup(k8sClient.Delete, ctx, apiSecret)

		// Create MariaDB database for testing
		mariadb.CreateMariaDBDatabase(namespace, "nova-api-db", mariadbv1.MariaDBDatabaseSpec{})
		DeferCleanup(k8sClient.Delete, ctx, mariadb.GetMariaDBDatabase(types.NamespacedName{Name: "nova-api-db", Namespace: namespace}))

		// Create the service password secret for all tests
		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nova-service-password",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"ServicePassword": []byte("test-password"),
			},
		}
		Expect(k8sClient.Create(ctx, passwordSecret)).Should(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, passwordSecret)
	})

	When("ApplicationCredential types are available", func() {
		It("should support creating KeystoneApplicationCredential for Nova service user", func() {
			// Create KeystoneApplicationCredential CR
			appCred := &keystonev1.KeystoneApplicationCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nova-applicationcredential",
					Namespace: namespace,
				},
				Spec: keystonev1.KeystoneApplicationCredentialSpec{
					UserName:        "nova",
					Secret:          "nova-service-password",
					ExpirationDays:  3,
					GracePeriodDays: 1,
				},
			}

			Expect(k8sClient.Create(ctx, appCred)).Should(Succeed())

			// Wait for KeystoneApplicationCredential to be ready
			Eventually(func(g Gomega) {
				appCredInstance := &keystonev1.KeystoneApplicationCredential{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "nova-applicationcredential", Namespace: namespace}, appCredInstance)).Should(Succeed())
				g.Expect(appCredInstance.Status.Conditions.IsTrue(condition.ReadyCondition)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Verify that the application credential secret was created
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				secretName := "nova-applicationcredential-application-credential"
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)).Should(Succeed())
				g.Expect(secret.Data["id"]).ShouldNot(BeEmpty())
				g.Expect(secret.Data["secret"]).ShouldNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("should support KeystoneApplicationCredential rotation", func() {
			// Create KeystoneApplicationCredential with short expiration
			appCred := &keystonev1.KeystoneApplicationCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nova-rotating-credential",
					Namespace: namespace,
				},
				Spec: keystonev1.KeystoneApplicationCredentialSpec{
					UserName:        "nova",
					Secret:          "nova-service-password",
					ExpirationDays:  2, // 2 day expiration (minimum allowed)
					GracePeriodDays: 1, // 1 day grace period (immediate rotation)
				},
			}

			Expect(k8sClient.Create(ctx, appCred)).Should(Succeed())

			// Wait for initial creation
			Eventually(func(g Gomega) {
				appCredInstance := &keystonev1.KeystoneApplicationCredential{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "nova-rotating-credential", Namespace: namespace}, appCredInstance)).Should(Succeed())
				g.Expect(appCredInstance.Status.Conditions.IsTrue(condition.ReadyCondition)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Get the initial secret
			secret := &corev1.Secret{}
			secretName := "nova-rotating-credential-application-credential" // #nosec G101
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)).Should(Succeed())
			initialID := string(secret.Data["id"])

			// Force reconciliation by updating the KeystoneApplicationCredential
			appCredInstance := &keystonev1.KeystoneApplicationCredential{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "nova-rotating-credential", Namespace: namespace}, appCredInstance)).Should(Succeed())
			appCredInstance.Annotations = map[string]string{"test-trigger": "rotation"}
			Expect(k8sClient.Update(ctx, appCredInstance)).Should(Succeed())

			// Verify that a new credential was created (different ID)
			Eventually(func(g Gomega) {
				updatedSecret := &corev1.Secret{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, updatedSecret)).Should(Succeed())
				newID := string(updatedSecret.Data["id"])
				g.Expect(newID).ShouldNot(Equal(initialID))
			}, timeout, interval).Should(Succeed())
		})

		It("should clean up KeystoneApplicationCredential resources on deletion", func() {
			// Create KeystoneApplicationCredential
			appCred := &keystonev1.KeystoneApplicationCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nova-cleanup-test",
					Namespace: namespace,
				},
				Spec: keystonev1.KeystoneApplicationCredentialSpec{
					UserName:        "nova",
					Secret:          "nova-service-password",
					ExpirationDays:  3,
					GracePeriodDays: 1,
				},
			}

			Expect(k8sClient.Create(ctx, appCred)).Should(Succeed())

			// Wait for it to be ready
			Eventually(func(g Gomega) {
				appCredInstance := &keystonev1.KeystoneApplicationCredential{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "nova-cleanup-test", Namespace: namespace}, appCredInstance)).Should(Succeed())
				g.Expect(appCredInstance.Status.Conditions.IsTrue(condition.ReadyCondition)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Verify secret exists
			secret := &corev1.Secret{}
			secretName := "nova-cleanup-test-application-credential" // #nosec G101
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)).Should(Succeed())

			// Delete the KeystoneApplicationCredential
			Expect(k8sClient.Delete(ctx, appCred)).Should(Succeed())

			// Verify the KeystoneApplicationCredential is deleted
			Eventually(func(g Gomega) {
				appCredInstance := &keystonev1.KeystoneApplicationCredential{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "nova-cleanup-test", Namespace: namespace}, appCredInstance)
				g.Expect(k8s_errors.IsNotFound(err)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Verify the secret is cleaned up
			Eventually(func(g Gomega) {
				cleanupSecret := &corev1.Secret{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, cleanupSecret)
				g.Expect(k8s_errors.IsNotFound(err)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	When("KeystoneApplicationCredential integration with Nova services", func() {
		It("should support using KeystoneApplicationCredentials for Nova authentication", func() {
			// This test should verify that Nova services can use Application Credentials
			// for authentication instead of passwords. This would involve:
			// 1. Creating a KeystoneApplicationCredential for the nova user
			// 2. Configuring Nova services to use the application credential
			// 3. Verifying that Nova services can authenticate successfully
			// 4. Testing that Nova operations work with application credentials

			// For now, just verify basic KeystoneApplicationCredential functionality
			appCred := &keystonev1.KeystoneApplicationCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nova-integration-test",
					Namespace: namespace,
				},
				Spec: keystonev1.KeystoneApplicationCredentialSpec{
					UserName:        "nova",
					Secret:          "nova-service-password",
					ExpirationDays:  3,
					GracePeriodDays: 1,
				},
			}

			Expect(k8sClient.Create(ctx, appCred)).Should(Succeed())

			Eventually(func(g Gomega) {
				appCredInstance := &keystonev1.KeystoneApplicationCredential{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "nova-integration-test", Namespace: namespace}, appCredInstance)).Should(Succeed())
				g.Expect(appCredInstance.Status.Conditions.IsTrue(condition.ReadyCondition)).Should(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
