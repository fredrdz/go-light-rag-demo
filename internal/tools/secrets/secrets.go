// Package secrets provides functionality to load secrets from Bitwarden
// and set them as environment variables.
package secrets

import (
	"errors"
	"fmt"
	"os"

	bws "github.com/bitwarden/sdk-go"
	"github.com/google/uuid"
)

// Load orchestrates the fetching and setting of secrets.
func Load() error {
	accessToken, organizationID, err := LoadConfig()
	if err != nil {
		return err
	}

	client, err := NewBitwardenClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if aerr := Authenticate(client, accessToken); aerr != nil {
		return aerr
	}

	secrets, err := FetchSecrets(client, organizationID)
	if err != nil {
		return err
	}

	if err := SetEnvironmentVariables(secrets); err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------

// LoadConfig retrieves required configuration from environment variables.
func LoadConfig() (string, string, error) {
	accessToken := os.Getenv("ACCESS_TOKEN")
	organizationID := os.Getenv("ORGANIZATION_ID")

	if accessToken == "" || organizationID == "" {
		return "", "", errors.New("ACCESS_TOKEN and ORGANIZATION_ID must be set")
	}

	if _, err := uuid.Parse(organizationID); err != nil {
		return "", "", fmt.Errorf("invalid uuid: %w", err)
	}

	return accessToken, organizationID, nil
}

// NewBitwardenClient initializes and returns a Bitwarden client.
func NewBitwardenClient() (bws.BitwardenClientInterface, error) {
	client, err := bws.NewBitwardenClient(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating bitwarden client: %w", err)
	}

	return client, nil
}

// Authenticate logs in to Bitwarden using an access token.
func Authenticate(client bws.BitwardenClientInterface, accessToken string) error {
	if err := client.AccessTokenLogin(accessToken, nil); err != nil {
		return fmt.Errorf("error logging in with access token: %w", err)
	}

	return nil
}

// FetchSecrets retrieves all secrets for the given organization.
func FetchSecrets(client bws.BitwardenClientInterface, organizationID string) (*bws.SecretsResponse, error) {
	secretIdentifiers, err := client.Secrets().List(organizationID)
	if err != nil {
		return nil, fmt.Errorf("error listing secrets: %w", err)
	}

	var secretIDs []string
	for _, identifier := range secretIdentifiers.Data {
		secretIDs = append(secretIDs, identifier.ID)
	}

	secrets, err := client.Secrets().GetByIDS(secretIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving secrets: %w", err)
	}

	return secrets, nil
}

// SetEnvironmentVariables sets retrieved secrets as environment variables.
func SetEnvironmentVariables(secrets *bws.SecretsResponse) error {
	for _, secret := range secrets.Data {
		if secret.Key == "" {
			continue
		}

		if err := os.Setenv(secret.Key, secret.Value); err != nil {
			return fmt.Errorf("error setting env var for key %s: %w", secret.Key, err)
		}
	}

	return nil
}
