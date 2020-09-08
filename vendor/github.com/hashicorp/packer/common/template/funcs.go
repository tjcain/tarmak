package template

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
)

// DeprecatedTemplateFunc wraps a template func to warn users that it's
// deprecated. The deprecation warning is called only once.
func DeprecatedTemplateFunc(funcName, useInstead string, deprecated func(string) string) func(string) string {
	once := sync.Once{}
	return func(in string) string {
		once.Do(func() {
			log.Printf("[WARN]: the `%s` template func is deprecated, please use %s instead",
				funcName, useInstead)
		})
		return deprecated(in)
	}
}

// Vault retrieves a secret from an HC vault KV store
func Vault(path string, key string) (string, error) {

	if token := os.Getenv("VAULT_TOKEN"); token == "" {
		return "", errors.New("Must set VAULT_TOKEN env var in order to use vault template function")
	}

	vaultConfig := vaultapi.DefaultConfig()
	cli, err := vaultapi.NewClient(vaultConfig)
	if err != nil {
		return "", fmt.Errorf("Error getting Vault client: %s", err)
	}
	secret, err := cli.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("Error reading vault secret: %s", err)
	}
	if secret == nil {
		return "", errors.New("Vault Secret does not exist at the given path")
	}

	data, ok := secret.Data["data"]
	if !ok {
		// maybe ths is v1, not v2 kv store
		value, ok := secret.Data[key]
		if ok {
			return value.(string), nil
		}

		// neither v1 nor v2 proudced a valid value
		return "", fmt.Errorf("Vault data was empty at the given path. Warnings: %s", strings.Join(secret.Warnings, "; "))
	}

	if val, ok := data.(map[string]interface{})[key]; ok {
		return val.(string), nil
	}
	return "", errors.New("Vault path does not contain the requested key")
}
