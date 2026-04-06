package jira

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

type Plugin struct {
	client *Client
}

func (p *Plugin) Name() string    { return "jira" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Setup(cfg map[string]string) error {
	baseURL, ok := cfg["base_url"]
	if !ok || baseURL == "" {
		return fmt.Errorf("missing required config: base_url")
	}

	email, ok := cfg["email"]
	if !ok || email == "" {
		return fmt.Errorf("missing required config: email")
	}

	apiToken, ok := cfg["api_token"]
	if !ok || apiToken == "" {
		return fmt.Errorf("missing required config: api_token")
	}

	p.client = NewClient(baseURL, email, apiToken)
	return nil
}

func (p *Plugin) Validate() error {
	if p.client == nil {
		return fmt.Errorf("plugin not initialized: call Setup() first")
	}

	ctx := context.Background()
	var user JiraUser
	if err := p.client.get(ctx, "/rest/api/3/myself", &user); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) CollectMetrics(ctx context.Context, since time.Time) ([]plugin.Metric, error) {
	return nil, nil
}

func (p *Plugin) DeriveSkills(ctx context.Context) ([]plugin.Skill, error) {
	return nil, nil
}

func (p *Plugin) Commands() []*cobra.Command {
	return nil
}
