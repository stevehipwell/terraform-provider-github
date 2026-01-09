package github

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"testing"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGithubActionsEnvironmentVariable(t *testing.T) {
	t.Run("create_update", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		envName := "environment / test"
		value := "my_variable_value"
		updatedValue := "my_updated_variable_value"

		config := `
resource "github_repository" "test" {
	name = "tf-acc-test-%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

resource "github_actions_environment_variable" "variable" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	variable_name    = "test_variable"
	value  = "%s"
}
`

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(config, randomID, envName, value),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "environment", envName),
						resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "value", value),
						resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "updated_at"),
					),
				},
				{
					Config: fmt.Sprintf(config, randomID, envName, updatedValue),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "environment", envName),
						resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "value", updatedValue),
						resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "created_at"),
						resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "updated_at"),
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

resource "github_actions_environment_variable" "variable" {
	repository 		= github_repository.test.name
	environment     = github_repository_environment.test.environment
	variable_name	= "test_variable"
	value 			= "my_variable_value"
}
`, randomID)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					Config:  config,
					Destroy: true,
				},
			},
		})
	})

	t.Run("import", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		envName := "environment / test"
		varName := "test_variable"
		value := "my_variable_value"

		config := fmt.Sprintf(`
resource "github_repository" "test" {
	name = "tf-acc-test-%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}

resource "github_actions_environment_variable" "variable" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	variable_name    = "%s"
	value  = "%s"
}
`, randomID, envName, varName, value)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					ResourceName:            "github_actions_environment_variable.variable",
					ImportStateId:           fmt.Sprintf(`tf-acc-test-%s:%s:%s`, randomID, envName, varName),
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"environment", "repository", "variable_name"},
				},
			},
		})
	})

	t.Run("error_on_existing", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		repoName := fmt.Sprintf("tf-acc-test-%s", randomID)
		envName := "environment / test"
		varName := "test_variable"

		baseConfig := fmt.Sprintf(`
resource "github_repository" "test" {
	name = "%s"
}

resource "github_repository_environment" "test" {
	repository       = github_repository.test.name
	environment      = "%s"
}
`, repoName, envName)

		config := fmt.Sprintf(`
%s

resource "github_actions_environment_variable" "variable" {
	repository       = github_repository.test.name
	environment      = github_repository_environment.test.environment
	variable_name    = "%s"
	value            = "test"
}
`, baseConfig, varName)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnauthenticated(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: baseConfig,
					Check: func(*terraform.State) error {
						meta, err := getTestMeta()
						if err != nil {
							return err
						}
						client := meta.v3client
						owner := meta.name
						ctx := context.Background()

						escapedEnvName := url.PathEscape(envName)

						_, err = client.Actions.CreateEnvVariable(ctx, owner, repoName, escapedEnvName, &github.ActionsVariable{
							Name:  varName,
							Value: "test",
						})
						return err
					},
				},
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Variable already exists`),
					Check: func(*terraform.State) error {
						meta, err := getTestMeta()
						if err != nil {
							return err
						}
						client := meta.v3client
						owner := meta.name
						ctx := context.Background()

						escapedEnvName := url.PathEscape(envName)

						_, err = client.Actions.DeleteEnvVariable(ctx, owner, repoName, escapedEnvName, varName)
						return err
					},
				},
			},
		})
	})
}
