# GPOD-117: TUI Issue List + Detail View + Work Context

**Date:** 2026-04-05  
**Issue:** [GPOD-117](https://linear.app/cloudcraftstudio/issue/GPOD-117)  
**Status:** Design approved

---

## Objetivo

Implementar la vista de lista de issues y la vista de detalle con Work Context (repos + workspace + environment) en la TUI de GetPod. Los issues pertenecen al client activo. El workspace y environment se eligen al trabajar cada ticket. El work context persiste por issue en SQLite.

SQLite no es solo cache de performance — es el knowledge store local que ancla metadata por issue (notas, tiempo de finalización, decisiones) que se construirá en tickets futuros.

---

## Decisiones de diseño

| Decisión | Elección | Razón |
|---|---|---|
| Arquitectura TUI | Sub-modelos Bubbletea | Separación clara, testeable, escala a GPOD-118 |
| Repos origen | Plugin API (GitHub/Bitbucket) | Fetch dinámico, no declaración estática en config |
| Cache issues | SQLite + refresh manual `[r]` | Knowledge store, no solo performance |
| Work context persistencia | Por issue en SQLite | Permite retomar trabajo entre sesiones |
| Plugin interface | Interfaces secundarias opcionales | Base Plugin sin cambios, PlanningPlugin y RepoPlugin opcionales |
| Fetch scope | Solo issues no cerrados | Evita cargar historia de 5 años de tickets |
| Protección knowledge store | Nunca borrar filas con `notes != ''` o `started_at IS NOT NULL` | Preservar metadata local de issues cerrados |

---

## 1. Modelo de datos (SQLite)

### Migración: tabla `issues`

```sql
CREATE TABLE IF NOT EXISTS issues (
    id          TEXT PRIMARY KEY,
    client      TEXT NOT NULL,
    key         TEXT NOT NULL,
    title       TEXT NOT NULL,
    status      TEXT NOT NULL,
    priority    TEXT,                   -- nullable (no todos los plugins lo tienen)
    description TEXT,
    labels      TEXT,                   -- JSON array: ["backend", "infra"]
    raw_data    TEXT,                   -- JSON snapshot completo del payload del plugin
    fetched_at  DATETIME NOT NULL,
    -- Work context (persiste por issue)
    repos       TEXT,                   -- JSON array: ["repo-a", "repo-b"]
    workspace   TEXT,
    environment TEXT,
    -- Knowledge store (para GPOD-118+)
    notes       TEXT,
    started_at  DATETIME,
    finished_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_issues_client ON issues(client);
```

### Regla de protección

Al hacer cleanup o refresh completo, nunca eliminar filas donde:
```go
func shouldPreserveIssue(i Issue) bool {
    return i.Notes != "" || i.StartedAt != nil
}
```

### Estrategia de fetch

- Al abrir la vista de issues de un client sin cache → auto-fetch
- `[r]` → fetch manual: obtiene issues con estado no cerrado del planning plugin
- El plugin decide qué estados son "no cerrados" (Done, Closed, Cancelled se excluyen)
- Cada fetch hace upsert (INSERT OR REPLACE) pero preserva columnas del knowledge store

---

## 2. Plugin interfaces

La interface `Plugin` base (`internal/plugin/interface.go`) no cambia.

### Interfaces secundarias opcionales

```go
// PlanningPlugin: para plugins que gestionan issues (Jira, Linear, etc.)
type PlanningPlugin interface {
    ListIssues(ctx context.Context) ([]Issue, error)
    GetIssue(ctx context.Context, key string) (*Issue, error)
    AddComment(ctx context.Context, key, body string) error
    ChangeStatus(ctx context.Context, key, status string) error
    ListStatuses(ctx context.Context) ([]string, error)
}

// RepoPlugin: para plugins que gestionan repositorios (GitHub, Bitbucket)
type RepoPlugin interface {
    ListRepos(ctx context.Context) ([]Repo, error)
}

// Issue representa un ticket de planificación normalizado.
type Issue struct {
    ID          string          // clave única global (ej: "linear:LULO-1234")
    Key         string          // display key (ej: "LULO-1234")
    Title       string
    Status      string
    Priority    string          // vacío si el plugin no lo soporta
    Description string
    Labels      []string
    RawData     json.RawMessage // payload completo del plugin
}

// Repo representa un repositorio de código.
type Repo struct {
    Name      string
    Source    string    // "github" | "bitbucket"
    Language  string
    CloneURL  string
    UpdatedAt time.Time
}
```

La TUI hace type assertion para detectar capacidades:
```go
if pp, ok := p.(plugin.PlanningPlugin); ok {
    issues, _ = pp.ListIssues(ctx)
}
```

---

## 3. Arquitectura TUI

### Estructura de archivos

```
internal/tui/
  app.go                        ← orquestador (ya existe, se extiende)
  styles.go                     ← ya existe
  selector.go                   ← ya existe
  msgs.go                       ← mensajes entre modelos (nuevo)
  views/
    issue_list.go               ← IssueListModel
    issue_detail.go             ← IssueDetailModel
  modals/
    repo_picker.go              ← RepoPickerModal
    workspace_picker.go         ← WorkspacePickerModal
    env_picker.go               ← EnvPickerModal
```

### Interface `Modal`

```go
// internal/tui/modals/modal.go
type Modal interface {
    tea.Model
    Title() string
}
```

### Mensajes (`msgs.go`)

```go
package tui

import "github.com/CloudCraft-Studio/getpod-cli/internal/plugin"

type IssueSelectedMsg     struct{ Issue store.Issue }
type NavigateBackMsg      struct{}
type ReposSelectedMsg     struct{ Repos []string }
type WorkspaceSelectedMsg struct{ Workspace string }
type EnvSelectedMsg       struct{ Env string }
type ModalClosedMsg       struct{}

type IssuesFetchedMsg struct {
    Client string          // necesario para routing en multi-client
    Issues []store.Issue
    Err    error
}

type ReposFetchedMsg struct {
    Repos []plugin.Repo
    Err   error
}
```

### Estado `App`

```go
type appView int

const (
    issueListView appView = iota
    issueDetailView
)

type App struct {
    // ... campos existentes (cfg, reg, st, styles, width, height, clientIdx, focus) ...
    activeView  appView
    issueList   *views.IssueListModel
    issueDetail *views.IssueDetailModel
    activeModal modals.Modal   // tipado con interface, no tea.Model
    hasModal    bool
}
```

### Routing de `[Esc]` en `App.Update`

```
App.Update recibe [Esc]:
  if hasModal → cierra modal (hasModal=false, activeModal=nil)
  else if activeView == issueDetailView → NavigateBackMsg → activeView = issueListView
  else → no-op (ya en lista)
```

Los sub-modelos nunca necesitan saber si hay un modal abierto.

### Flujo de navegación

```
App
 └── IssueListModel
      ├── [r]     → fetch async → IssuesFetchedMsg
      ├── [Enter] → IssueSelectedMsg → App transiciona a IssueDetailModel
      └── auto-fetch al init si sin cache para el client activo

 └── IssueDetailModel (recibe issue al construirse)
      ├── [w] → App abre RepoPickerModal   → ReposSelectedMsg → persiste en SQLite
      ├── [x] → App abre WorkspacePickerModal → WorkspaceSelectedMsg → persiste, resetea env
      ├── [e] → App abre EnvPickerModal (requiere workspace)  → EnvSelectedMsg → persiste
      └── [Esc] → NavigateBackMsg → App vuelve a IssueListModel
```

---

## 4. Componentes

### `IssueListModel`

**Estado:** `items []store.Issue`, `cursor int`, `filter string`, `filterActive bool`, `loading bool`, `err error`

**Render — columnas:**
```
● LULO-1234  Fix EKS ingress controller      ●qa   High  In Progress
○ LULO-1235  Update Terraform modules              Med   Todo
```
- status dot: `●` activo, `○` todo, `◐` en review, `✓` done
- env dot: solo si `issue.Environment != ""`
- priority: badge coloreado (High=rojo, Med=amarillo, Low=gris)
- status: text muted

**Teclas:** `↑↓` navegar, `/` activar filtro (busca en key+título+status), `Esc` desactivar filtro, `Enter` → `IssueSelectedMsg`, `[r]` refresh fetch

**Init:** si no hay items para el client activo → dispara fetch async

---

### `IssueDetailModel`

**Estado:** `issue store.Issue`, `descOffset int`, `focusedSection int`

**Secciones:**

1. **Header:** `KEY · ENV  Title`
2. **Metadata:** badges de status / priority / labels
3. **Description:** texto con scroll `↑↓` (desplaza `descOffset`)
4. **Work Context:**
   ```
   [w] Repositories  repo-a, repo-b (+1 more)
   [x] Workspace     Lulo X
   [e] Environment   qa · AWS 111111111111 · us-east-1
   
   ✓ Ready: 2 repos · Lulo X · qa · AWS 111111111111
   ```
   Si falta algo: `○ Not ready: missing workspace`
5. **Actions:** lista de acciones con estado habilitado/deshabilitado (stubs en GPOD-117, implementación en GPOD-118):
   ```
   [p] Plan with AI      (requires repos + workspace + env)
   [b] Create branch     (requires repos)
   [c] Commit + Push     ✓
   [r] Create PR         (requires repos)
   [m] Comment           ✓
   [s] Change status     ✓
   ```

**Persistencia:** al recibir `ReposSelectedMsg`, `WorkspaceSelectedMsg`, `EnvSelectedMsg` → inmediatamente hace upsert en SQLite (sin botón guardar)

---

### `RepoPickerModal`

**Estado:** `items []plugin.Repo`, `selected map[string]bool`, `cursor int`, `filter string`, `loading bool`

**Init:** fetch repos del RepoPlugin configurado para el client (async)

**Render:**
```
┌─ Select Repositories ──────────────────────────┐
│ /filter                                         │
│ [x] repo-backend      github  Go    2h ago      │
│ [ ] repo-frontend     github  TS    1d ago      │
│ [x] infra-terraform   github  HCL   3d ago      │
└─────────────────────── space toggle · enter ok ─┘
```

**Teclas:** `↑↓`, `space` toggle, `/` filtro, `Enter` → `ReposSelectedMsg`, `Esc` → `ModalClosedMsg`

---

### `WorkspacePickerModal`

**Estado:** workspaces del client (desde `cfg.Clients[client].Workspaces`), `cursor int`

**Sin fetch** — datos del config en memoria.

**Render:** lista de workspaces con display name. Al seleccionar → `WorkspaceSelectedMsg` + el env del issue se resetea a vacío.

---

### `EnvPickerModal`

**Estado:** environments del workspace activo (desde config), `cursor int`

**Sin fetch** — datos del config en memoria.

**Requiere** `issue.Workspace != ""` para abrirse (si no, App muestra mensaje de error inline).

**Render:**
```
┌─ Select Environment ───────────────────────────┐
│   qa    AWS 111111111111  us-east-1             │
│   stg   AWS 222222222222  us-east-1             │
│ ⚠ prod  AWS 333333333333  us-east-1             │
└────────────────────────────────── enter select ─┘
```

`prod` muestra `⚠` en amarillo. Al seleccionar → `EnvSelectedMsg`.

---

## 5. Configuración (conventions)

El config de workspace ya soporta `ContextConfig map[string]map[string]string`. Las keys convencionales que la TUI leerá para el env picker son:

```yaml
workspaces:
  lulo-x:
    display_name: "Lulo X"
    contexts:
      qa:
        aws_account: "111111111111"
        aws_region:  "us-east-1"
        jira:
          project_key: "LULO"
```

`aws_account` y `aws_region` son keys convencionales leídas directamente por el env picker. No requieren cambios al schema de config.

---

## 6. Acceptance Criteria (GPOD-117)

- [ ] Lista de issues del client activo con scroll y filtro `/`
- [ ] Auto-fetch al abrir vista sin cache; refresh manual con `[r]`
- [ ] Vista detalle con secciones: header, metadata, description (con scroll), work context, actions
- [ ] Repo picker modal: multi-select, filtro, fetch dinámico del RepoPlugin
- [ ] Workspace picker modal: lista desde config, selección única
- [ ] Environment picker modal: depende de workspace, muestra AWS account/region, warning prod
- [ ] Ready indicator: `✓ Ready: N repos · Workspace · env · AWS account`
- [ ] Work context persiste inmediatamente en SQLite al cambiar cualquier campo
- [ ] Actions se muestran habilitadas/deshabilitadas según work context (stubs — implementación en GPOD-118)
- [ ] Environment se puede cambiar sin resetear repos ni workspace
- [ ] Para clients con un solo workspace/env, se auto-selecciona
- [ ] `[Esc]` con modal abierto cierra modal; sin modal en detalle vuelve a lista
- [ ] Migración SQLite para tabla `issues` con índice por client
- [ ] Interfaces `PlanningPlugin` y `RepoPlugin` definidas en `internal/plugin/interface.go`
- [ ] Mensaje `IssuesFetchedMsg` incluye campo `Client`
