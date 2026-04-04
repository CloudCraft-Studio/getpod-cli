# GetPod CLI — Unified Developer Workbench

GetPod es una herramienta diseñada específicamente para desarrolladores que trabajan como autónomos o consultores para **múltiples clientes/empresas**, centralizando su flujo de trabajo en una sola terminal.

## 🏛 Arquitectura de Configuración

GetPod utiliza una jerarquía de 3 niveles para organizar el trabajo y las herramientas de cada cliente de forma aislada.

```text
Cliente/Empresa (e.g. Acme Corp)
  ├── Plugins Activos (Jira, GitHub, Slack...)
  └── Credenciales Globales (Tokens, API Keys, URLs base)
  └── Workspace (e.g. Core Services, Mobile App)
        └── Context/Environment (e.g. dev, staging, prod)
              └── Configuración específica (Project keys, repo IDs, branches)
```

### El modelo de datos

1.  **Client**: Representa la entidad legal o cliente final. Aquí se configuran los plugins que el cliente utiliza oficialmente y las credenciales que el desarrollador tiene para acceder a sus sistemas.
2.  **Workspace**: Proyectos o áreas de trabajo dentro de una misma empresa (por ejemplo, el equipo de "SRE" y el equipo de "Frontend").
3.  **Context**: Entornos de ejecución específicos dentro de un workspace. Los plugins del cliente heredan sus credenciales y se fusionan con los parámetros específicos del contexto (por ejemplo, el `board_id` de Jira puede ser distinto en `dev` que en `prod`).

---

## 🛠 Instalación y Configuración

### 1. Instalación local

```bash
make install
```

### 2. Inicializar configuración

```bash
getpod config init
```
Esto crea `~/.getpod/config.yaml`. Edita este archivo para añadir tus clientes.

---

## 🚀 Flujo de Trabajo (Uso de Contextos)

GetPod funciona de forma similar a `kubectl context`, permitiéndote saltar entre clientes y proyectos manteniendo el estado de la sesión.

```bash
# 1. Ver y seleccionar el cliente activo
getpod client list
getpod client use acme-corp

# 2. Seleccionar el área de trabajo
getpod workspace use backend-services

# 3. Seleccionar el entorno
getpod context use dev

# 4. Validar configuración de plugins para este contexto
getpod plugins validate
```

Una vez configurado el contexto, todos los comandos de los plugins cargados funcionarán automáticamente con los tokens y parámetros correctos.

---

## 🔌 Sistema de Plugins (Registry)

GetPod utiliza un registro de plugins **compiled-in**. Cada plugin implementa la interface `Plugin` definida en `/internal/plugin/interface.go`.

**Ciclo de vida al ejecutar un comando:**
1.  Se carga `config.yaml`.
2.  Se resuelve el contexto activo desde `state.yaml`.
3.  Se inyectan las credenciales (merge cliente + contexto).
4.  El Registry inicializa solo los plugins activos para ese cliente.
5.  Los subcomandos de los plugins se inyectan dinámicamente en el comando raíz.

---

## 🤖 Desarrollo

```bash
make build   # Compila el binario en ./bin/getpod
make test    # Ejecuta unit tests (usando mock plugins)
make install # Instala globalmente en $GOPATH/bin
```