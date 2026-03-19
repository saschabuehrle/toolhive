// SPDX-FileCopyrightText: Copyright 2025 Stacklok, Inc.
// SPDX-License-Identifier: Apache-2.0

package virtualmcp

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	mcpv1alpha1 "github.com/stacklok/toolhive/cmd/thv-operator/api/v1alpha1"
	vmcpconfig "github.com/stacklok/toolhive/pkg/vmcp/config"
)

var _ = Describe("VirtualMCPServer AuthServerConfigRef Validation", Ordered, func() {
	var (
		testNamespace   = "default"
		mcpGroupName    = "auth-server-test-group"
		timeout         = 2 * time.Minute
		pollingInterval = 1 * time.Second
	)

	BeforeAll(func() {
		By("Creating MCPGroup for auth server tests")
		CreateMCPGroupAndWait(ctx, k8sClient, mcpGroupName, testNamespace,
			"Test MCP Group for AuthServerConfigRef validation", timeout, pollingInterval)
	})

	AfterAll(func() {
		By("Cleaning up MCPGroup")
		_ = k8sClient.Delete(ctx, &mcpv1alpha1.MCPGroup{
			ObjectMeta: metav1.ObjectMeta{Name: mcpGroupName, Namespace: testNamespace},
		})
	})

	Context("when AuthServerConfigRef references a nonexistent resource", func() {
		const vmcpName = "auth-server-notfound-vmcp"

		BeforeAll(func() {
			By("Creating VirtualMCPServer with nonexistent AuthServerConfigRef")
			vmcp := &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmcpName,
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.VirtualMCPServerSpec{
					IncomingAuth: &mcpv1alpha1.IncomingAuthConfig{
						Type: "anonymous",
					},
					Config: vmcpconfig.Config{Group: mcpGroupName},
					AuthServerConfigRef: &mcpv1alpha1.ExternalAuthConfigRef{
						Name: "nonexistent-auth-config",
					},
				},
			}
			Expect(k8sClient.Create(ctx, vmcp)).To(Succeed())
		})

		AfterAll(func() {
			_ = k8sClient.Delete(ctx, &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: vmcpName, Namespace: testNamespace},
			})
		})

		It("should set AuthServerConfigValidated condition to False", func() {
			WaitForCondition(ctx, k8sClient, vmcpName, testNamespace,
				mcpv1alpha1.ConditionTypeAuthServerConfigValidated, "False", timeout, pollingInterval)
		})

		It("should set phase to Failed", func() {
			Eventually(func() error {
				vmcp := &mcpv1alpha1.VirtualMCPServer{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vmcpName, Namespace: testNamespace,
				}, vmcp); err != nil {
					return err
				}
				if vmcp.Status.Phase != mcpv1alpha1.VirtualMCPServerPhaseFailed {
					return fmt.Errorf("expected phase Failed, got %s", vmcp.Status.Phase)
				}
				return nil
			}, timeout, pollingInterval).Should(Succeed())
		})
	})

	Context("when AuthServerConfigRef references wrong type", func() {
		const (
			vmcpName       = "auth-server-wrongtype-vmcp"
			authConfigName = "auth-server-wrongtype-config"
		)

		BeforeAll(func() {
			By("Creating MCPExternalAuthConfig with unauthenticated type (not embeddedAuthServer)")
			extAuth := &mcpv1alpha1.MCPExternalAuthConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      authConfigName,
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPExternalAuthConfigSpec{
					Type: mcpv1alpha1.ExternalAuthTypeUnauthenticated,
				},
			}
			Expect(k8sClient.Create(ctx, extAuth)).To(Succeed())

			By("Creating VirtualMCPServer with AuthServerConfigRef to wrong type")
			vmcp := &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmcpName,
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.VirtualMCPServerSpec{
					IncomingAuth: &mcpv1alpha1.IncomingAuthConfig{
						Type: "anonymous",
					},
					Config: vmcpconfig.Config{Group: mcpGroupName},
					AuthServerConfigRef: &mcpv1alpha1.ExternalAuthConfigRef{
						Name: authConfigName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, vmcp)).To(Succeed())
		})

		AfterAll(func() {
			_ = k8sClient.Delete(ctx, &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: vmcpName, Namespace: testNamespace},
			})
			_ = k8sClient.Delete(ctx, &mcpv1alpha1.MCPExternalAuthConfig{
				ObjectMeta: metav1.ObjectMeta{Name: authConfigName, Namespace: testNamespace},
			})
		})

		It("should set AuthServerConfigValidated condition to False", func() {
			WaitForCondition(ctx, k8sClient, vmcpName, testNamespace,
				mcpv1alpha1.ConditionTypeAuthServerConfigValidated, "False", timeout, pollingInterval)
		})

		It("should set phase to Failed", func() {
			Eventually(func() error {
				vmcp := &mcpv1alpha1.VirtualMCPServer{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vmcpName, Namespace: testNamespace,
				}, vmcp); err != nil {
					return err
				}
				if vmcp.Status.Phase != mcpv1alpha1.VirtualMCPServerPhaseFailed {
					return fmt.Errorf("expected phase Failed, got %s", vmcp.Status.Phase)
				}
				return nil
			}, timeout, pollingInterval).Should(Succeed())
		})
	})

	Context("when AuthServerConfigRef references valid embeddedAuthServer", func() {
		const (
			vmcpName       = "auth-server-valid-vmcp"
			authConfigName = "auth-server-valid-config"
		)

		BeforeAll(func() {
			By("Creating MCPExternalAuthConfig with embeddedAuthServer type")
			extAuth := &mcpv1alpha1.MCPExternalAuthConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      authConfigName,
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPExternalAuthConfigSpec{
					Type: mcpv1alpha1.ExternalAuthTypeEmbeddedAuthServer,
					EmbeddedAuthServer: &mcpv1alpha1.EmbeddedAuthServerConfig{
						Issuer: "http://localhost:9090",
						UpstreamProviders: []mcpv1alpha1.UpstreamProviderConfig{
							{
								Name: "test-provider",
								Type: mcpv1alpha1.UpstreamProviderTypeOIDC,
								OIDCConfig: &mcpv1alpha1.OIDCUpstreamConfig{
									IssuerURL: "https://accounts.google.com",
									ClientID:  "test-client-id",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, extAuth)).To(Succeed())

			By("Creating VirtualMCPServer with valid AuthServerConfigRef")
			vmcp := &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmcpName,
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.VirtualMCPServerSpec{
					IncomingAuth: &mcpv1alpha1.IncomingAuthConfig{
						Type: "anonymous",
					},
					Config: vmcpconfig.Config{Group: mcpGroupName},
					AuthServerConfigRef: &mcpv1alpha1.ExternalAuthConfigRef{
						Name: authConfigName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, vmcp)).To(Succeed())
		})

		AfterAll(func() {
			_ = k8sClient.Delete(ctx, &mcpv1alpha1.VirtualMCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: vmcpName, Namespace: testNamespace},
			})
			_ = k8sClient.Delete(ctx, &mcpv1alpha1.MCPExternalAuthConfig{
				ObjectMeta: metav1.ObjectMeta{Name: authConfigName, Namespace: testNamespace},
			})
		})

		It("should set AuthServerConfigValidated condition to True", func() {
			WaitForCondition(ctx, k8sClient, vmcpName, testNamespace,
				mcpv1alpha1.ConditionTypeAuthServerConfigValidated, "True", timeout, pollingInterval)
		})
	})
})
