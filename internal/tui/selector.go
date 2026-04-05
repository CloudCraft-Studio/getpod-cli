package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/CloudCraft-Studio/getpod-cli/internal/config"
	"github.com/CloudCraft-Studio/getpod-cli/internal/state"
)

type Selector struct {
	config *config.Config
	state  *state.State
}

func NewSelector(cfg *config.Config) *Selector {
	return &Selector{config: cfg}
}

func (s *Selector) LoadState() error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	s.state = st
	return nil
}

func (s *Selector) HasActiveContext() bool {
	return s.state != nil &&
		s.state.ActiveClient != "" &&
		s.state.ActiveWorkspace != "" &&
		s.state.ActiveContext != ""
}

func (s *Selector) Run() error {
	if s.config == nil {
		return fmt.Errorf("no hay configuración cargada")
	}

	if s.state == nil {
		s.state = &state.State{}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n=== GetPod Context Selector ===\n")

	if err := s.selectClient(reader); err != nil {
		return err
	}
	if err := s.selectWorkspace(reader); err != nil {
		return err
	}
	if err := s.selectContext(reader); err != nil {
		return err
	}

	if err := s.state.Save(); err != nil {
		return err
	}

	fmt.Printf("\n✓ Contexto completo: %s / %s / %s\n",
		s.state.ActiveClient,
		s.state.ActiveWorkspace,
		s.state.ActiveContext,
	)
	return nil
}

func (s *Selector) selectClient(reader *bufio.Reader) error {
	if s.state.ActiveClient != "" {
		fmt.Printf("Cliente actual: %s\n", s.state.ActiveClient)
		return nil
	}

	clients := make([]string, 0, len(s.config.Clients))
	for name := range s.config.Clients {
		clients = append(clients, name)
	}

	if len(clients) == 0 {
		return fmt.Errorf("no hay clientes configurados")
	}

	if len(clients) == 1 {
		s.state.UseClient(clients[0])
		fmt.Printf("✓ Cliente seleccionado: %s\n", clients[0])
		return nil
	}

	fmt.Println("Selecciona un cliente:")
	for i, name := range clients {
		fmt.Printf("  %d) %s\n", i+1, name)
	}

	idx, err := s.readIndex(reader, len(clients))
	if err != nil {
		return err
	}

	s.state.UseClient(clients[idx-1])
	fmt.Printf("✓ Cliente seleccionado: %s\n", clients[idx-1])
	return nil
}

func (s *Selector) selectWorkspace(reader *bufio.Reader) error {
	if s.state.ActiveWorkspace != "" {
		fmt.Printf("Workspace actual: %s\n", s.state.ActiveWorkspace)
		return nil
	}

	client, ok := s.config.Clients[s.state.ActiveClient]
	if !ok {
		return fmt.Errorf("cliente %q no encontrado", s.state.ActiveClient)
	}

	workspaces := make([]string, 0, len(client.Workspaces))
	for name := range client.Workspaces {
		workspaces = append(workspaces, name)
	}

	if len(workspaces) == 0 {
		return fmt.Errorf("no hay workspaces para el cliente %s", s.state.ActiveClient)
	}

	if len(workspaces) == 1 {
		s.state.UseWorkspace(workspaces[0])
		fmt.Printf("✓ Workspace seleccionado: %s\n", workspaces[0])
		return nil
	}

	fmt.Println("\nSelecciona un workspace:")
	for i, name := range workspaces {
		fmt.Printf("  %d) %s\n", i+1, name)
	}

	idx, err := s.readIndex(reader, len(workspaces))
	if err != nil {
		return err
	}

	s.state.UseWorkspace(workspaces[idx-1])
	fmt.Printf("✓ Workspace seleccionado: %s\n", workspaces[idx-1])
	return nil
}

func (s *Selector) selectContext(reader *bufio.Reader) error {
	if s.state.ActiveContext != "" {
		fmt.Printf("Context actual: %s\n", s.state.ActiveContext)
		return nil
	}

	client, _ := s.config.Clients[s.state.ActiveClient]
	ws, ok := client.Workspaces[s.state.ActiveWorkspace]
	if !ok {
		return fmt.Errorf("workspace %q no encontrado", s.state.ActiveWorkspace)
	}

	contexts := make([]string, 0, len(ws.Contexts))
	for name := range ws.Contexts {
		contexts = append(contexts, name)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no hay contextos para el workspace %s", s.state.ActiveWorkspace)
	}

	if len(contexts) == 1 {
		s.state.UseContext(contexts[0])
		fmt.Printf("✓ Context seleccionado: %s\n", contexts[0])
		return nil
	}

	fmt.Println("\nSelecciona un contexto:")
	for i, name := range contexts {
		fmt.Printf("  %d) %s\n", i+1, name)
	}

	idx, err := s.readIndex(reader, len(contexts))
	if err != nil {
		return err
	}

	s.state.UseContext(contexts[idx-1])
	fmt.Printf("✓ Context seleccionado: %s\n", contexts[idx-1])
	return nil
}

func (s *Selector) readIndex(reader *bufio.Reader, max int) (int, error) {
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > max {
			fmt.Printf("Entrada inválida. Ingresa un número entre 1 y %d.\n", max)
			continue
		}
		return idx, nil
	}
}
