package e2eencryption

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	oauthapiconfigobservercontroller "github.com/openshift/cluster-authentication-operator/pkg/operator/configobservation"
	operatorencryption "github.com/openshift/cluster-authentication-operator/test/library/encryption"
	library "github.com/openshift/library-go/test/library/encryption"
)

func TestEncryptionTypeIdentity(t *testing.T) {
	library.TestEncryptionTypeIdentity(t, library.BasicScenario{
		Namespace:                       "openshift-config-managed",
		LabelSelector:                   "encryption.apiserver.operator.openshift.io/component" + "=" + "openshift-oauth-apiserver",
		EncryptionConfigSecretName:      fmt.Sprintf("encryption-config-openshift-oauth-apiserver"),
		EncryptionConfigSecretNamespace: "openshift-config-managed",
		OperatorNamespace:               "openshift-authentication-operator",
		TargetGRs:                       operatorencryption.DefaultTargetGRs,
		AssertFunc:                      operatorencryption.AssertTokens,
	})
}

func TestEncryptionTypeUnset(t *testing.T) {
	library.TestEncryptionTypeUnset(t, library.BasicScenario{
		Namespace:                       "openshift-config-managed",
		LabelSelector:                   "encryption.apiserver.operator.openshift.io/component" + "=" + "openshift-oauth-apiserver",
		EncryptionConfigSecretName:      fmt.Sprintf("encryption-config-openshift-oauth-apiserver"),
		EncryptionConfigSecretNamespace: "openshift-config-managed",
		OperatorNamespace:               "openshift-authentication-operator",
		TargetGRs:                       operatorencryption.DefaultTargetGRs,
		AssertFunc:                      operatorencryption.AssertTokens,
	})
}

func TestEncryptionTurnOnAndOff(t *testing.T) {
	library.TestEncryptionTurnOnAndOff(t, library.OnOffScenario{
		BasicScenario: library.BasicScenario{
			Namespace:                       "openshift-config-managed",
			LabelSelector:                   "encryption.apiserver.operator.openshift.io/component" + "=" + "openshift-oauth-apiserver",
			EncryptionConfigSecretName:      fmt.Sprintf("encryption-config-openshift-oauth-apiserver"),
			EncryptionConfigSecretNamespace: "openshift-config-managed",
			OperatorNamespace:               "openshift-authentication-operator",
			TargetGRs:                       operatorencryption.DefaultTargetGRs,
			AssertFunc:                      operatorencryption.AssertTokens,
		},
		CreateResourceFunc: func(t testing.TB, _ library.ClientSet, namespace string) runtime.Object {
			return operatorencryption.CreateAndStoreTokenOfLife(context.TODO(), t, operatorencryption.GetClients(t))
		},
		AssertResourceEncryptedFunc:    operatorencryption.AssertTokenOfLifeEncrypted,
		AssertResourceNotEncryptedFunc: operatorencryption.AssertTokenOfLifeNotEncrypted,
		ResourceFunc:                   func(t testing.TB, _ string) runtime.Object { return operatorencryption.TokenOfLife(t) },
		ResourceName:                   "TokenOfLife",
	})
}

// TestEncryptionRotation first encrypts data with aescbc key
// then it forces a key rotation by setting the "encyrption.Reason" in the operator's configuration file
func TestEncryptionRotation(t *testing.T) {
	ctx := context.TODO()
	library.TestEncryptionRotation(t, library.RotationScenario{
		BasicScenario: library.BasicScenario{
			Namespace:                       "openshift-config-managed",
			LabelSelector:                   "encryption.apiserver.operator.openshift.io/component" + "=" + "openshift-oauth-apiserver",
			EncryptionConfigSecretName:      fmt.Sprintf("encryption-config-openshift-oauth-apiserver"),
			EncryptionConfigSecretNamespace: "openshift-config-managed",
			OperatorNamespace:               "openshift-authentication-operator",
			TargetGRs:                       operatorencryption.DefaultTargetGRs,
			AssertFunc:                      operatorencryption.AssertTokens,
		},
		CreateResourceFunc: func(t testing.TB, _ library.ClientSet, _ string) runtime.Object {
			return operatorencryption.CreateAndStoreTokenOfLife(ctx, t, operatorencryption.GetClients(t))
		},
		GetRawResourceFunc: func(t testing.TB, clientSet library.ClientSet, _ string) string {
			return operatorencryption.GetRawTokenOfLife(t, clientSet)
		},
		UnsupportedConfigFunc: func(rawUnsupportedEncryptionCfg []byte) error {
			cs := operatorencryption.GetClients(t)
			authOperator, err := cs.OperatorClient.Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}

			unsupportedConfigAsMap := map[string]interface{}{}
			if len(authOperator.Spec.UnsupportedConfigOverrides.Raw) > 0 {
				if err := json.Unmarshal(authOperator.Spec.UnsupportedConfigOverrides.Raw, &unsupportedConfigAsMap); err != nil {
					return err
				}
			}
			unsupportedEncryptionConfigAsMap := map[string]interface{}{}
			if err := json.Unmarshal(rawUnsupportedEncryptionCfg, &unsupportedEncryptionConfigAsMap); err != nil {
				return err
			}
			if err := unstructured.SetNestedMap(unsupportedConfigAsMap, unsupportedEncryptionConfigAsMap, oauthapiconfigobservercontroller.OAuthAPIServerConfigPrefix); err != nil {
				return err
			}
			rawUnsupportedCfg, err := json.Marshal(unsupportedConfigAsMap)
			if err != nil {
				return err
			}
			authOperator.Spec.UnsupportedConfigOverrides.Raw = rawUnsupportedCfg

			_, err = cs.OperatorClient.Update(ctx, authOperator, metav1.UpdateOptions{})
			return err
		},
	})
}
