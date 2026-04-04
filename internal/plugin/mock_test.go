package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// mockPlugin es una implementación de Plugin para usar en tests.
type mockPlugin struct {
	name      string
	setupCfg  map[string]string // guarda lo que recibió en Setup
	setupErr  error             // error a retornar en Setup
	metrics   []Metric
	metricErr error
	skills    []Skill
	skillErr  error
	commands  []*cobra.Command
}

func newMockPlugin(name string) *mockPlugin {
	return &mockPlugin{
		name: name,
		commands: []*cobra.Command{
			{Use: fmt.Sprintf("%s-cmd", name), Short: fmt.Sprintf("comando de %s", name)},
		},
	}
}

func (m *mockPlugin) Name() string    { return m.name }
func (m *mockPlugin) Version() string { return "0.0.1-mock" }

func (m *mockPlugin) Setup(cfg map[string]string) error {
	m.setupCfg = cfg
	return m.setupErr
}

func (m *mockPlugin) Validate() error { return nil }

func (m *mockPlugin) CollectMetrics(_ context.Context, _ time.Time) ([]Metric, error) {
	if m.metricErr != nil {
		return nil, m.metricErr
	}
	return m.metrics, nil
}

func (m *mockPlugin) DeriveSkills(_ context.Context) ([]Skill, error) {
	if m.skillErr != nil {
		return nil, m.skillErr
	}
	return m.skills, nil
}

func (m *mockPlugin) Commands() []*cobra.Command { return m.commands }

// errorPlugin es un plugin que siempre falla en métricas y skills.
var _ Plugin = (*mockPlugin)(nil)

func mockWithMetricError(name string, err error) *mockPlugin {
	p := newMockPlugin(name)
	p.metricErr = err
	return p
}

func mockWithSkillError(name string, err error) *mockPlugin {
	p := newMockPlugin(name)
	p.skillErr = err
	return p
}

// sentinel errors para tests
var errMockMetric = errors.New("mock metric error")
var errMockSkill = errors.New("mock skill error")
