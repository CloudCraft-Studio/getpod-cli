package plugin

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

// Plugin es el contrato que todo plugin de GetPod debe implementar.
// Cada integración externa (Jira, Linear, GitHub, Bitbucket, etc.)
// implementa esta interface y se registra en el Registry al arrancar la CLI.
//
// Ciclo de vida:
//  1. Setup()   — recibe credenciales del cliente + config del context activo (merge)
//  2. Validate() — verifica conectividad / credenciales
//  3. CollectMetrics() / DeriveSkills() — recolección de datos
//  4. Commands() — expone subcomandos Cobra al root command
type Plugin interface {
	// Name retorna el identificador único del plugin (e.g. "jira", "linear").
	// Debe coincidir con la clave en clients.<nombre>.plugins del config.yaml.
	Name() string

	// Version retorna la versión del plugin.
	Version() string

	// Setup inicializa el plugin con la configuración efectiva del context activo.
	// El mapa recibido es el merge de:
	//   - clients.<cliente>.plugins.<plugin>  (credenciales: token, url...)
	//   - clients.<cliente>.workspaces.<ws>.contexts.<ctx>.<plugin>  (config: project_key, repo...)
	Setup(cfg map[string]string) error

	// Validate verifica que las credenciales y la conectividad sean correctas.
	Validate() error

	// CollectMetrics recolecta métricas desde la herramienta externa
	// a partir de la fecha `since`.
	CollectMetrics(ctx context.Context, since time.Time) ([]Metric, error)

	// DeriveSkills infiere habilidades del desarrollador a partir de la actividad
	// registrada en la herramienta externa.
	DeriveSkills(ctx context.Context) ([]Skill, error)

	// Commands expone subcomandos Cobra específicos del plugin.
	// Estos se registran automáticamente en el root command al arrancar.
	Commands() []*cobra.Command
}

// Metric representa un evento de actividad capturado desde un plugin.
type Metric struct {
	Plugin    string            `json:"plugin"`
	Event     string            `json:"event"`
	Timestamp time.Time         `json:"timestamp"`
	Meta      map[string]string `json:"meta"`
}

// Skill representa una habilidad inferida a partir de la actividad del desarrollador.
type Skill struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"` // 0.0 – 1.0
	Source     string  `json:"source"`     // nombre del plugin que la derivó
}
