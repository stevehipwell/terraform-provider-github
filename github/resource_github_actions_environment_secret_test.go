package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"testing"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGithubActionsEnvironmentSecret(t *testing.T) {
	t.Run("create_update_plaintext", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		envName := "environment / test"
		value := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := `
resource "github_repository" "test" {
	name = "tf-acc-test-%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

resource "github_actions_environment_secret" "test" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	secret_name      = "test_plaintext_secret_name"
	plaintext_value  = "%s"
}
`

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, randomID, envName, value),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "plaintext_value", value),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "encrypted_value"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
				{
					Config: fmt.Sprintf(config, randomID, envName, updatedValue),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "plaintext_value", updatedValue),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "encrypted_value"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
			},
		})
	})

	t.Run("create_update_encrypted", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		envName := "environment / test"
		value := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := `
resource "github_repository" "test" {
	name = "tf-acc-test-%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

resource "github_actions_environment_secret" "test" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	secret_name      = "test_encrypted_secret_name"
	encrypted_value  = "%s"
}
`

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, randomID, envName, value),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "plaintext_value"),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "encrypted_value", value),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
				{
					Config: fmt.Sprintf(config, randomID, envName, updatedValue),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "plaintext_value"),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "encrypted_value", updatedValue),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
			},
		})
	})

	t.Run("create_update_encrypted_with_key", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		envName := "environment / test"
		value := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := `
resource "github_repository" "test" {
	name = "tf-acc-test-%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

data "github_actions_environment_public_key" "test" {
	repository = github_repository.test.name
	environment = github_repository_environment.test.environment
}

resource "github_actions_environment_secret" "test" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	key_id           = data.github_actions_environment_public_key.test.key_id
	secret_name      = "test_encrypted_secret_name"
	encrypted_value  = "%s"
}
`

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, randomID, envName, value),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "plaintext_value"),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "encrypted_value", value),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
				{
					Config: fmt.Sprintf(config, randomID, envName, updatedValue),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", envName),
						resource.TestCheckNoResourceAttr("github_actions_environment_secret.test", "plaintext_value"),
						resource.TestCheckResourceAttr("github_actions_environment_secret.test", "encrypted_value", updatedValue),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "key_id"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
					),
				},
			},
		})
	})

	t.Run("recreate_on_drift", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		repoName := fmt.Sprintf("tf-acc-test-%s", randomID)
		envName := "environment / test"
		secretName := "test_plaintext_secret_name"

		config := fmt.Sprintf(`
resource "github_repository" "test" {
	name = "%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

resource "github_actions_environment_secret" "test" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	secret_name      = "%s"
	plaintext_value  = "test"
}
`, repoName, envName, secretName)

		var beforeCreatedAt string
		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						func(s *terraform.State) error {
							beforeCreatedAt = s.RootModule().Resources["github_actions_environment_secret.test"].Primary.Attributes["created_at"]
							return nil
						},
					),
				},
				{
					PreConfig: func() {
						meta, err := getTestMeta()
						if err != nil {
							t.Fatal(err.Error())
						}
						client := meta.v3client
						owner := meta.name
						ctx := context.Background()

						escapedEnvName := url.PathEscape(envName)

						repo, _, err := client.Repositories.Get(ctx, owner, repoName)
						if err != nil {
							t.Fatal(err.Error())
						}
						repoID := int(repo.GetID())

						keyID, _, err := getEnvironmentPublicKeyDetails(ctx, meta, repoID, escapedEnvName)
						if err != nil {
							t.Fatal(err.Error())
						}

						_, err = client.Actions.CreateOrUpdateEnvSecret(ctx, repoID, escapedEnvName, &github.EncryptedSecret{
							Name:           secretName,
							EncryptedValue: base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value")),
							KeyID:          keyID,
						})
						if err != nil {
							t.Fatal(err.Error())
						}
					},
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						func(s *terraform.State) error {
							afterCreatedAt := s.RootModule().Resources["github_actions_environment_secret.test"].Primary.Attributes["created_at"]

							if beforeCreatedAt == afterCreatedAt {
								return fmt.Errorf("expected resource to be recreated, but created_at remained the same: %s", beforeCreatedAt)
							}
							return nil
						},
					),
				},
			},
		})
	})

	t.Run("destroy", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
	resource "github_repository" "test" {
		name = "tf-acc-test-%s"
	}

	resource "github_repository_environment" "test" {
		repository       = github_repository.test.name
		environment      = "environment / test"
	}

	resource "github_actions_environment_secret" "test" {
		repository       = github_repository.test.name
		environment      = github_repository_environment.test.environment
		secret_name      = "test_plaintext_secret_name"
		plaintext_value  = "test"
	}
`, randomID)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config:  config,
					Destroy: true,
				},
			},
		})
	})
}
