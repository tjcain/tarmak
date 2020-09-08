// Copyright Jetstack Ltd. See LICENSE for details.
package kubernetes

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	vault "github.com/hashicorp/vault/api"
)

type InitToken struct {
	Role          string
	Policies      []string
	kubernetes    *Kubernetes
	token         *string
	ExpectedToken string
}

func (i *InitToken) Ensure() error {
	var result error

	if err := i.ensureCreated(); err != nil {
		return err
	}

	// always ensure token role and init token policy is set (this is idempotent)
	if err := i.writeTokenRole(); err != nil {
		result = multierror.Append(result, fmt.Errorf("not able to write token role: %s", err))
	}
	if err := i.writeInitTokenPolicy(); err != nil {
		result = multierror.Append(result, fmt.Errorf("not able to write init token policy: %s", err))
	}
	if result != nil {
		return result
	}

	// make sure init token exists
	initToken, err := i.InitToken()
	if err != nil {
		return fmt.Errorf("not able to ensure init token: %s", err)
	}
	i.token = &initToken

	// Renew token
	_, err = i.kubernetes.vaultClient.Auth().Token().Renew(initToken, 0)
	if err != nil {
		return err
	}

	return nil
}

func (i *InitToken) ensureCreated() error {
	token, err := i.InitToken()
	if err != nil {
		return err
	}

	_, err = i.kubernetes.vaultClient.Auth().Token().Lookup(token)
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), "Code: 403.") ||
		!strings.Contains(err.Error(), "bad token") {
		i.kubernetes.Log.Errorf("GOT ERROR HERE!\n%s\n", err)
		return err
	}

	// token revoked or expired
	_, err = i.kubernetes.vaultClient.Auth().Token().CreateOrphan(&vault.TokenCreateRequest{
		ID:          token,
		DisplayName: fmt.Sprintf("%s/secrets/init_token_%s", i.kubernetes.Path(), i.Role),
		TTL:         fmt.Sprintf("%d", int(i.kubernetes.MaxValidityInitTokens.Seconds())),
		Period:      fmt.Sprintf("%d", int(i.kubernetes.MaxValidityInitTokens.Seconds())),
		Policies:    []string{"default", fmt.Sprintf("%s-creator", i.namePath())},
	})
	if err != nil {
		return err
	}

	return nil
}

func (i *InitToken) Delete() error {
	var result *multierror.Error

	if err := i.deleteInitTokenPolicy(); err != nil {
		result = multierror.Append(result, err)
	}

	if err := i.deleteTokenRole(); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func (i *InitToken) EnsureDryRun() (bool, error) {
	var result *multierror.Error

	expDate, err := i.ensureDryRunExpDate()
	if err != nil || expDate {
		return expDate, err
	}

	secret, err := i.readTokenRole()
	if err != nil {
		result = multierror.Append(result, err)
	} else if len(secret.Data) == 0 {
		return true, result.ErrorOrNil()
	}

	if !secretDataMatch(secret.Data, i.writeData()) {
		return true, result.ErrorOrNil()
	}

	policy, err := i.readInitTokenPolicy()
	if err != nil {
		result = multierror.Append(result, err)
	} else if policy != i.policy().Policy() {
		return true, result.ErrorOrNil()
	}

	// get init token from secrets backend
	token, err := i.secretsBackend().InitToken(i.Name(), i.Role, []string{fmt.Sprintf("%s-creator", i.namePath())}, i.ExpectedToken)
	if err != nil {
		result = multierror.Append(result, err)
	} else if token == "" {
		return true, result.ErrorOrNil()
	}

	return false, result.ErrorOrNil()
}

func (i *InitToken) ensureDryRunExpDate() (bool, error) {
	token, err := i.InitToken()
	if err != nil {
		return true, err
	}

	s, err := i.kubernetes.vaultClient.Auth().Token().Lookup(token)
	if err != nil {
		if strings.Contains(err.Error(), "Code: 403.") &&
			strings.Contains(err.Error(), "bad token") {
			return true, nil
		}

		return true, err
	}

	ttl, err := s.TokenTTL()
	if err != nil {
		return true, err
	}

	// less than a year
	if ttl.Hours() < 24*365 {
		return true, nil
	}

	return false, nil
}

// Get init token name
func (i *InitToken) Name() string {
	return fmt.Sprintf("%s-%s", i.kubernetes.clusterID, i.Role)
}

// Get name path suffix for token role
func (i *InitToken) namePath() string {
	return fmt.Sprintf("%s/%s", i.kubernetes.clusterID, i.Role)
}

// Construct file path for ../create
func (i *InitToken) createPath() string {
	return filepath.Join("auth/token/create", i.Name())
}

// Construct file path for ../auth
func (i *InitToken) Path() string {
	return filepath.Join("auth/token/roles", i.Name())
}

// Write token role to vault
func (i *InitToken) writeTokenRole() error {
	_, err := i.kubernetes.vaultClient.Logical().Write(i.Path(), i.writeData())
	if err != nil {
		return fmt.Errorf("error writing token role %s: %v", i.Path(), err)
	}

	return nil
}

func (i *InitToken) deleteTokenRole() error {
	_, err := i.kubernetes.vaultClient.Logical().Delete(i.Path())
	if err != nil {
		return fmt.Errorf("error deleting token role %s: %v", i.Path(), err)
	}

	return nil
}

func (i *InitToken) readTokenRole() (*vault.Secret, error) {
	secret, err := i.kubernetes.vaultClient.Logical().Read(i.Path())
	if err != nil {
		return nil, fmt.Errorf("error read token role %s: %v", i.Path(), err)
	}

	return secret, nil
}

func (i *InitToken) policy() *Policy {
	return &Policy{
		Name: fmt.Sprintf("%s-creator", i.namePath()),
		Policies: []*policyPath{
			&policyPath{
				path:         i.createPath(),
				capabilities: []string{"create", "read", "update"},
			},
		},
	}
}

// Construct policy and send to kubernetes to be written to vault
func (i *InitToken) writeInitTokenPolicy() error {
	return i.kubernetes.WritePolicy(i.policy())
}

func (i *InitToken) deleteInitTokenPolicy() error {
	return i.kubernetes.DeletePolicy(i.policy())
}

func (i *InitToken) readInitTokenPolicy() (string, error) {
	return i.kubernetes.ReadPolicy(i.policy())
}

// InitToken fetches the token from the secrets backend if it is not already set
func (i *InitToken) InitToken() (string, error) {
	if i.token != nil {
		return *i.token, nil
	}

	// get init token from the secrets backend
	token, err := i.secretsBackend().InitToken(i.Name(), i.Role, []string{fmt.Sprintf("%s-creator", i.namePath())}, i.ExpectedToken)
	if err != nil {
		return "", err
	}

	i.token = &token
	return token, nil
}

func (i *InitToken) secretsBackend() *GenericVaultBackend {
	return i.kubernetes.secretsBackend
}

func (i *InitToken) writeData() map[string]interface{} {
	return map[string]interface{}{
		"period":           fmt.Sprintf("%d", int(i.kubernetes.MaxValidityInitTokens.Seconds())),
		"orphan":           true,
		"allowed_policies": i.Policies,
		"path_suffix":      i.namePath(),
	}
}
