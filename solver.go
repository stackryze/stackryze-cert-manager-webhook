package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	whapi "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
)

// stackryzeSolver implements the cert-manager DNS01 webhook Solver interface.
type stackryzeSolver struct {
	kube *kubernetes.Clientset
}

// secretKeyRef points at a key inside a Kubernetes Secret.
type secretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// stackryzeConfig is the per-issuer config block in the ACME solver.
type stackryzeConfig struct {
	APIURL            string       `json:"apiUrl"`
	APITokenSecretRef secretKeyRef `json:"apiTokenSecretRef"`
}

func (s *stackryzeSolver) Name() string { return "stackryze" }

func (s *stackryzeSolver) Initialize(kubeClientConfig *restclient.Config, _ <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	s.kube = cl
	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (stackryzeConfig, error) {
	cfg := stackryzeConfig{}
	if cfgJSON == nil {
		return cfg, fmt.Errorf("no solver config provided")
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %w", err)
	}
	if cfg.APITokenSecretRef.Name == "" || cfg.APITokenSecretRef.Key == "" {
		return cfg, fmt.Errorf("apiTokenSecretRef.name and .key are required")
	}
	return cfg, nil
}

func (s *stackryzeSolver) token(cfg stackryzeConfig, namespace string) (string, error) {
	secret, err := s.kube.CoreV1().Secrets(namespace).Get(context.TODO(), cfg.APITokenSecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to load secret %q/%q: %w", namespace, cfg.APITokenSecretRef.Name, err)
	}
	b, ok := secret.Data[cfg.APITokenSecretRef.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %q", cfg.APITokenSecretRef.Key, cfg.APITokenSecretRef.Name)
	}
	return strings.TrimSpace(string(b)), nil
}

// label converts the challenge FQDN into a record label relative to the zone.
func label(fqdn, zoneName string) string {
	f := strip(strings.ToLower(fqdn))
	z := strip(strings.ToLower(zoneName))
	if f == z {
		return "@"
	}
	return strings.TrimSuffix(f, "."+z)
}

func (s *stackryzeSolver) client(ch *whapi.ChallengeRequest) (*stackryzeClient, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}
	token, err := s.token(cfg, ch.ResourceNamespace)
	if err != nil {
		return nil, err
	}
	return newClient(cfg.APIURL, token), nil
}

func (s *stackryzeSolver) Present(ch *whapi.ChallengeRequest) error {
	c, err := s.client(ch)
	if err != nil {
		return err
	}
	z, err := c.getZoneByName(ch.ResolvedZone)
	if err != nil {
		return err
	}
	return c.addTXT(z.ID, label(ch.ResolvedFQDN, ch.ResolvedZone), fmt.Sprintf("%q", ch.Key))
}

func (s *stackryzeSolver) CleanUp(ch *whapi.ChallengeRequest) error {
	c, err := s.client(ch)
	if err != nil {
		return err
	}
	z, err := c.getZoneByName(ch.ResolvedZone)
	if err != nil {
		return err
	}
	return c.deleteTXT(z.ID, label(ch.ResolvedFQDN, ch.ResolvedZone), fmt.Sprintf("%q", ch.Key))
}
