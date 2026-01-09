package github

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubActionsEnvironmentVariable() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubActionsEnvironmentVariableCreate,
		ReadContext:   resourceGithubActionsEnvironmentVariableRead,
		UpdateContext: resourceGithubActionsEnvironmentVariableUpdate,
		DeleteContext: resourceGithubActionsEnvironmentVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the repository.",
			},
			"environment": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the environment.",
			},
			"variable_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Name of the variable.",
				ValidateDiagFunc: validateSecretNameFunc,
			},
			"value": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Value of the variable.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_variable' creation.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_variable' update.",
			},
		},
	}
}

func resourceGithubActionsEnvironmentVariableCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName := d.Get("repository").(string)
	envName := d.Get("environment").(string)
	name := d.Get("variable_name").(string)

	escapedEnvName := url.PathEscape(envName)

	_, err := client.Actions.CreateEnvVariable(ctx, owner, repoName, escapedEnvName, &github.ActionsVariable{
		Name:  name,
		Value: d.Get("value").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildID(repoName, envName, name))

	// GitHub API does not return on create so we have to lookup the variable to get timestamps
	if variable, _, err := client.Actions.GetEnvVariable(ctx, owner, repoName, escapedEnvName, name); err == nil {
		if err := d.Set("created_at", variable.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("updated_at", variable.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubActionsEnvironmentVariableRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, envName, name, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	escapedEnvName := url.PathEscape(envName)

	variable, _, err := client.Actions.GetEnvVariable(ctx, owner, repoName, escapedEnvName, name)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing actions variable %s from state because it no longer exists in GitHub",
					d.Id())
				d.SetId("")
				return nil
			}
		}
		return diag.FromErr(err)
	}

	if err = d.Set("value", variable.Value); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("created_at", variable.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("updated_at", variable.UpdatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubActionsEnvironmentVariableUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, envName, name, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	escapedEnvName := url.PathEscape(envName)

	_, err = client.Actions.UpdateEnvVariable(ctx, owner, repoName, escapedEnvName, &github.ActionsVariable{
		Name:  name,
		Value: d.Get("value").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// GitHub API does not return on create so we have to lookup the variable to get timestamps
	if variable, _, err := client.Actions.GetEnvVariable(ctx, owner, repoName, escapedEnvName, name); err == nil {
		if err := d.Set("created_at", variable.CreatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("updated_at", variable.UpdatedAt.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubActionsEnvironmentVariableDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta := m.(*Owner)
	client := meta.v3client
	owner := meta.name

	repoName, envName, name, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	escapedEnvName := url.PathEscape(envName)

	_, err = client.Actions.DeleteEnvVariable(ctx, owner, repoName, escapedEnvName, name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
