package coder_bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coder/coder/v2/codersdk"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Config struct {
	URL                     string
	AccessToken             string
	TemplateID              string
	TemplateVersionID       string
	TemplateVersionPresetID string
	TemplateName            string
	TemplateExampleID       string
	TemplateExampleName     string
	UserEmail               string
	Username                string
	UserPassword            string
	UserFullName            string
	TokenName               string
	TokenLifetime           time.Duration
	TokenScope              codersdk.APIKeyScope
	WaitTimeout             time.Duration
	WaitInterval            time.Duration
	RequireTemplate         bool
}

type Result struct {
	AccessToken             string
	TemplateID              string
	TemplateVersionID       string
	TemplateVersionPresetID string
}

func Ensure(ctx context.Context, cfg Config) (Result, error) {
	cfg = withDefaults(cfg)
	if strings.TrimSpace(cfg.URL) == "" {
		return Result{}, errors.New("CODER_URL is required")
	}
	client, err := newClient(cfg.URL)
	if err != nil {
		return Result{}, err
	}
	if err := waitForCoder(ctx, client, cfg.WaitTimeout, cfg.WaitInterval); err != nil {
		return Result{}, err
	}

	accessToken, orgID, err := ensureAccessToken(ctx, client, cfg)
	if err != nil {
		return Result{}, err
	}
	client.SetSessionToken(accessToken)

	result := Result{
		AccessToken:             accessToken,
		TemplateID:              strings.TrimSpace(cfg.TemplateID),
		TemplateVersionID:       strings.TrimSpace(cfg.TemplateVersionID),
		TemplateVersionPresetID: strings.TrimSpace(cfg.TemplateVersionPresetID),
	}

	if cfg.RequireTemplate {
		templateResult, err := ensureTemplate(ctx, client, cfg, orgID)
		if err != nil {
			return Result{}, err
		}
		result.TemplateID = templateResult.TemplateID
		result.TemplateVersionID = templateResult.TemplateVersionID
		if result.TemplateVersionPresetID == "" {
			result.TemplateVersionPresetID = templateResult.TemplateVersionPresetID
		}
	}

	return result, nil
}

func withDefaults(cfg Config) Config {
	if cfg.TokenScope == "" {
		cfg.TokenScope = codersdk.APIKeyScopeAll
	}
	if cfg.TokenName == "" {
		cfg.TokenName = "coder-bootstrap"
	}
	if cfg.TokenLifetime == 0 {
		cfg.TokenLifetime = 720 * time.Hour
	}
	if cfg.WaitTimeout == 0 {
		cfg.WaitTimeout = 2 * time.Minute
	}
	if cfg.WaitInterval == 0 {
		cfg.WaitInterval = 2 * time.Second
	}
	if cfg.UserEmail == "" {
		cfg.UserEmail = "coder@example.com"
	}
	if cfg.Username == "" {
		cfg.Username = "coder"
	}
	if cfg.UserFullName == "" {
		cfg.UserFullName = "Coder Admin"
	}
	if cfg.UserPassword == "" {
		cfg.UserPassword = "coder"
	}
	if cfg.TemplateName == "" {
		cfg.TemplateName = "default-template"
	}
	if cfg.TemplateExampleID == "" && cfg.TemplateExampleName == "" {
		cfg.TemplateExampleName = "Docker"
	}
	return cfg
}

func newClient(rawURL string) (*codersdk.Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid CODER_URL: %w", err)
	}
	return codersdk.New(parsed), nil
}

func waitForCoder(ctx context.Context, client *codersdk.Client, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		_, err := client.HasFirstUser(ctx)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("coder not ready: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func ensureAccessToken(ctx context.Context, client *codersdk.Client, cfg Config) (string, uuid.UUID, error) {
	token := strings.TrimSpace(cfg.AccessToken)
	if token != "" && !isPlaceholderToken(token) {
		client.SetSessionToken(token)
		if _, err := client.User(ctx, codersdk.Me); err == nil {
			return token, uuid.Nil, nil
		}
	}

	hasUser, err := client.HasFirstUser(ctx)
	if err != nil {
		return "", uuid.Nil, err
	}

	var orgID uuid.UUID
	if !hasUser {
		resp, err := client.CreateFirstUser(ctx, codersdk.CreateFirstUserRequest{
			Email:    cfg.UserEmail,
			Username: cfg.Username,
			Name:     cfg.UserFullName,
			Password: cfg.UserPassword,
		})
		if err != nil {
			return "", uuid.Nil, err
		}
		orgID = resp.OrganizationID
		zap.S().Infow("coder first user created", "username", cfg.Username)
	}

	loginResp, err := client.LoginWithPassword(ctx, codersdk.LoginWithPasswordRequest{
		Email:    cfg.UserEmail,
		Password: cfg.UserPassword,
	})
	if err != nil {
		return "", uuid.Nil, err
	}
	client.SetSessionToken(loginResp.SessionToken)

	user, err := client.User(ctx, codersdk.Me)
	if err != nil {
		return "", uuid.Nil, err
	}
	if orgID == uuid.Nil {
		orgs, err := client.Organizations(ctx)
		if err != nil {
			return "", uuid.Nil, err
		}
		if len(orgs) == 0 {
			return "", uuid.Nil, errors.New("coder has no organizations")
		}
		orgID = orgs[0].ID
	}

	apiKey, err := client.CreateToken(ctx, user.ID.String(), codersdk.CreateTokenRequest{
		Lifetime:  cfg.TokenLifetime,
		Scope:     cfg.TokenScope,
		TokenName: cfg.TokenName,
	})
	if err != nil {
		zap.S().Warnw("failed to create coder api token, using session token", "error", err)
		return loginResp.SessionToken, orgID, nil
	}
	return apiKey.Key, orgID, nil
}

func ensureTemplate(ctx context.Context, client *codersdk.Client, cfg Config, orgID uuid.UUID) (Result, error) {
	if orgID == uuid.Nil {
		orgs, err := client.Organizations(ctx)
		if err != nil {
			return Result{}, err
		}
		if len(orgs) == 0 {
			return Result{}, errors.New("coder has no organizations")
		}
		orgID = orgs[0].ID
	}

	templateID := parseUUID(cfg.TemplateID)
	templateVersionID := parseUUID(cfg.TemplateVersionID)
	presetID := parseUUID(cfg.TemplateVersionPresetID)

	if templateVersionID == nil && templateID != nil {
		template, err := client.Template(ctx, *templateID)
		if err != nil {
			return Result{}, err
		}
		templateVersionID = &template.ActiveVersionID
	}
	if templateID == nil && templateVersionID != nil {
		version, err := client.TemplateVersion(ctx, *templateVersionID)
		if err != nil {
			return Result{}, err
		}
		templateID = version.TemplateID
	}

	if templateID == nil && templateVersionID == nil {
		found, err := findTemplateByName(ctx, client, orgID, cfg.TemplateName)
		if err != nil {
			return Result{}, err
		}
		if found != nil {
			templateID = &found.ID
			templateVersionID = &found.ActiveVersionID
		} else {
			created, versionID, err := createTemplateFromExample(ctx, client, cfg, orgID)
			if err != nil {
				return Result{}, err
			}
			templateID = &created.ID
			templateVersionID = &versionID
		}
	}

	result := Result{}
	if templateID != nil {
		result.TemplateID = templateID.String()
	}
	if templateVersionID != nil {
		result.TemplateVersionID = templateVersionID.String()
		if presetID == nil {
			presets, err := client.TemplateVersionPresets(ctx, *templateVersionID)
			if err != nil {
				return Result{}, err
			}
			if len(presets) > 0 {
				result.TemplateVersionPresetID = presets[0].ID.String()
			}
		} else {
			result.TemplateVersionPresetID = presetID.String()
		}
	}
	return result, nil
}

func findTemplateByName(ctx context.Context, client *codersdk.Client, orgID uuid.UUID, name string) (*codersdk.Template, error) {
	templates, err := client.Templates(ctx, codersdk.TemplateFilter{
		OrganizationID: orgID,
		ExactName:      name,
	})
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, nil
	}
	return &templates[0], nil
}

func createTemplateFromExample(ctx context.Context, client *codersdk.Client, cfg Config, orgID uuid.UUID) (codersdk.Template, uuid.UUID, error) {
	exampleID, err := resolveExampleID(ctx, client, cfg)
	if err != nil {
		return codersdk.Template{}, uuid.Nil, err
	}

	version, err := client.CreateTemplateVersion(ctx, orgID, codersdk.CreateTemplateVersionRequest{
		Name:          "bootstrap",
		Message:       "bootstrap template",
		StorageMethod: codersdk.ProvisionerStorageMethodFile,
		ExampleID:     exampleID,
		Provisioner:   codersdk.ProvisionerTypeTerraform,
	})
	if err != nil {
		return codersdk.Template{}, uuid.Nil, err
	}
	if err := waitForJob(ctx, client, orgID, version.Job.ID, cfg.WaitTimeout, cfg.WaitInterval); err != nil {
		return codersdk.Template{}, uuid.Nil, err
	}

	template, err := client.CreateTemplate(ctx, orgID, codersdk.CreateTemplateRequest{
		Name:        cfg.TemplateName,
		DisplayName: cfg.TemplateName,
		VersionID:   version.ID,
	})
	if err != nil {
		return codersdk.Template{}, uuid.Nil, err
	}
	return template, template.ActiveVersionID, nil
}

func resolveExampleID(ctx context.Context, client *codersdk.Client, cfg Config) (string, error) {
	if strings.TrimSpace(cfg.TemplateExampleID) != "" {
		return cfg.TemplateExampleID, nil
	}
	examples, err := client.StarterTemplates(ctx)
	if err != nil {
		return "", err
	}
	if len(examples) == 0 {
		return "", errors.New("coder has no starter templates")
	}
	if cfg.TemplateExampleName == "" {
		return examples[0].ID, nil
	}
	for _, example := range examples {
		if strings.EqualFold(example.Name, cfg.TemplateExampleName) {
			return example.ID, nil
		}
	}
	return examples[0].ID, nil
}

func waitForJob(ctx context.Context, client *codersdk.Client, orgID, jobID uuid.UUID, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		job, err := client.OrganizationProvisionerJob(ctx, orgID, jobID)
		if err != nil {
			return err
		}
		switch job.Status {
		case codersdk.ProvisionerJobSucceeded:
			return nil
		case codersdk.ProvisionerJobFailed:
			if job.Error != "" {
				return errors.New(job.Error)
			}
			return errors.New("template version job failed")
		}
		if time.Now().After(deadline) {
			return errors.New("template version job timed out")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func parseUUID(raw string) *uuid.UUID {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func isPlaceholderToken(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "changeme", "change-me", "replace-me", "replace_me":
		return true
	default:
		return false
	}
}
