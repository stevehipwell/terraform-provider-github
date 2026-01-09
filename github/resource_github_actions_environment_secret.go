package github

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubActionsEnvironmentSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubActionsEnvironmentSecretCreate,
		ReadContext:   resourceGithubActionsEnvironmentSecretRead,
		DeleteContext: resourceGithubActionsEnvironmentSecretDelete,

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, m any) error {
			if len(diff.Id()) == 0 {
				return nil
			}

			remoteUpdatedAt := diff.Get("remote_updated_at").(string)
			if len(remoteUpdatedAt) == 0 {
				return nil
			}

			updatedAt := diff.Get("updated_at").(string)
			if updatedAt != remoteUpdatedAt {
				err := diff.SetNew("updated_at", remoteUpdatedAt)
				if err != nil {
					return err
				}

				if len(updatedAt) != 0 {
					return diff.ForceNew("updated_at")
				}
			}

			return nil
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the repository.",
			},
			"environment": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the environment.",
			},
			"secret_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Name of the secret.",
				ValidateDiagFunc: validateSecretNameFunc,
			},
			"key_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				Description:   "ID of the public key used to encrypt the secret.",
				ConflictsWith: []string{"plaintext_value"},
			},
			"encrypted_value": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Sensitive:        true,
				Description:      "Encrypted value of the secret using the GitHub public key in Base64 format.",
				ConflictsWith:    []string{"plaintext_value"},
				ValidateDiagFunc: toDiagFunc(validation.StringIsBase64, "encrypted_value"),
			},
			"plaintext_value": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Sensitive:     true,
				Description:   "Plaintext value of the secret to be encrypted.",
				ConflictsWith: []string{"encrypted_value"},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_environment_secret' creation.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_environment_secret' update.",
			},
			"remote_updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of remote 'actions_environment_secret' update.",
			},
		},
	}
}

func resourceGithubActionsEnvironmentSecretCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName := d.Get("repository").(string)
	envName := d.Get("environment").(string)
	name := d.Get("secret_name").(string)
	keyID := d.Get("key_id").(string)
	encryptedValue := d.Get("encrypted_value").(string)

	escapedEnvName := url.PathEscape(envName)

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return diag.FromErr(err)
	}
	repoID := int(repo.GetID())

	var publicKey string
	if len(keyID) == 0 || len(encryptedValue) == 0 {
		keyID, publicKey, err = getEnvironmentPublicKeyDetails(ctx, meta, repoID, escapedEnvName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(encryptedValue) == 0 {
		plaintextValue := d.Get("plaintext_value").(string)

		encryptedBytes, err := encryptPlaintext(plaintextValue, publicKey)
		if err != nil {
			return diag.FromErr(err)
		}
		encryptedValue = base64.StdEncoding.EncodeToString(encryptedBytes)
	}

	_, err = client.Actions.CreateOrUpdateEnvSecret(ctx, repoID, escapedEnvName, &github.EncryptedSecret{
		Name:           name,
		KeyID:          keyID,
		EncryptedValue: encryptedValue,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildID(repoName, envName, name))
	if err := d.Set("key_id", keyID); err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on create so we have to lookup the secret to get timestamps
	if secret, _, err := client.Actions.GetEnvSecret(ctx, repoID, escapedEnvName, name); err == nil {
		if err := d.Set("created_at", secret.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubActionsEnvironmentSecretRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, envName, name, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	escapedEnvName := url.PathEscape(envName)

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing environment secret %s from state because it no longer exists in GitHub", d.Id())
				d.SetId("")
				return nil
			}
		}
		return diag.FromErr(err)
	}

	secret, _, err := client.Actions.GetEnvSecret(ctx, int(repo.GetID()), escapedEnvName, name)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing environment secret %s from state because it no longer exists in GitHub", d.Id())
				d.SetId("")
				return nil
			}
		}
		return diag.FromErr(err)
	}

	if len(d.Get("created_at").(string)) == 0 {
		if err = d.Set("created_at", secret.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if len(d.Get("updated_at").(string)) == 0 {
		if err = d.Set("updated_at", secret.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err = d.Set("remote_updated_at", secret.UpdatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubActionsEnvironmentSecretDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, envName, name, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	escapedEnvName := url.PathEscape(envName)

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting environment secret: %s", d.Id())
	_, err = client.Actions.DeleteEnvSecret(ctx, int(repo.GetID()), escapedEnvName, name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getEnvironmentPublicKeyDetails(ctx context.Context, meta *Owner, repoID int, envName string) (string, string, error) {
	client := meta.v3client

	publicKey, _, err := client.Actions.GetEnvPublicKey(ctx, repoID, envName)
	if err != nil {
		return "", "", err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), nil
}
