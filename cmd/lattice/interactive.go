package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Start interactive REPL mode",
	Long: `Start an interactive session with the mesh.

Commands in interactive mode:
  /help     - Show help
  /agents   - List available agents
  /status   - Show mesh status
  /agent ID - Switch to specific agent
  /mesh     - Switch back to mesh mode
  /quit     - Exit interactive mode

Any other input will be sent to the mesh (or selected agent).`,
	RunE: runInteractive,
}

func runInteractive(cmd *cobra.Command, args []string) error {
	// Check server connectivity
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		return fmt.Errorf("cannot connect to server at %s: %w", serverURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy at %s", serverURL)
	}

	fmt.Println("Lattice Interactive Mode")
	fmt.Printf("Connected to: %s\n", serverURL)
	fmt.Println("Type /help for commands, /quit to exit")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	currentAgent := "" // Empty means mesh mode

	for {
		// Show prompt
		if currentAgent != "" {
			fmt.Printf("[%s]> ", currentAgent)
		} else {
			fmt.Print("[mesh]> ")
		}

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return nil
			}
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if handleCommand(input, &currentAgent) {
				continue
			}
			// /quit returns false
			break
		}

		// Send to mesh or agent
		if err := sendRequest(input, currentAgent); err != nil {
			fmt.Printf("Error: %v\n\n", err)
		}
	}

	fmt.Println("Goodbye!")
	return nil
}

func handleCommand(input string, currentAgent *string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		printHelp()

	case "/agents":
		listAgents()

	case "/status":
		showStatus()

	case "/agent":
		if len(parts) < 2 {
			fmt.Println("Usage: /agent <agent-id>")
		} else {
			*currentAgent = parts[1]
			fmt.Printf("Switched to agent: %s\n", *currentAgent)
		}

	case "/mesh":
		*currentAgent = ""
		fmt.Println("Switched to mesh mode")

	case "/quit", "/exit", "/q":
		return false

	default:
		fmt.Printf("Unknown command: %s (type /help for commands)\n", cmd)
	}

	fmt.Println()
	return true
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help     - Show this help")
	fmt.Println("  /agents   - List available agents")
	fmt.Println("  /status   - Show mesh status")
	fmt.Println("  /agent ID - Switch to specific agent")
	fmt.Println("  /mesh     - Switch back to mesh mode")
	fmt.Println("  /quit     - Exit interactive mode")
	fmt.Println()
	fmt.Println("Any other input will be sent to the mesh or selected agent.")
}

func listAgents() {
	resp, err := doRequest("GET", serverURL+"/agents", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", string(body))
		return
	}

	var result struct {
		Agents []struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Provides []string `json:"provides"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return
	}

	if len(result.Agents) == 0 {
		fmt.Println("No agents registered")
		return
	}

	fmt.Println("Available agents:")
	for _, a := range result.Agents {
		caps := strings.Join(a.Provides, ", ")
		fmt.Printf("  %s (%s) - [%s]\n", a.ID, a.Name, caps)
	}
}

func showStatus() {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		fmt.Printf("Server not reachable: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server unhealthy (status: %d)\n", resp.StatusCode)
		return
	}

	fmt.Printf("Server: %s\n", serverURL)
	fmt.Println("Status: OK")

	// Get agent count
	agentResp, err := doRequest("GET", serverURL+"/agents", nil)
	if err != nil {
		return
	}
	defer agentResp.Body.Close()

	if agentResp.StatusCode == http.StatusOK {
		agentBody, _ := io.ReadAll(agentResp.Body)
		var result struct {
			Agents []any `json:"agents"`
		}
		json.Unmarshal(agentBody, &result)
		fmt.Printf("Agents: %d registered\n", len(result.Agents))
	}
}

func sendRequest(input, agentID string) error {
	var url string
	if agentID != "" {
		url = serverURL + "/agents/" + agentID + "/run"
	} else {
		url = serverURL + "/mesh/run"
	}

	reqBody, _ := json.Marshal(map[string]string{"input": input})
	resp, err := doRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", string(body))
	}

	// Parse response
	var result struct {
		Output    string `json:"output"`
		Duration  string `json:"duration"`
		TokensIn  int    `json:"tokens_in"`
		TokensOut int    `json:"tokens_out"`
		TraceID   string `json:"trace_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// Just print raw output if parsing fails
		fmt.Println(string(body))
		return nil
	}

	fmt.Println()
	fmt.Println(result.Output)
	fmt.Println()

	// Show stats if available
	if result.Duration != "" {
		stats := fmt.Sprintf("[Duration: %s", result.Duration)
		if result.TokensIn > 0 || result.TokensOut > 0 {
			stats += fmt.Sprintf(", Tokens: %d/%d", result.TokensIn, result.TokensOut)
		}
		if result.TraceID != "" {
			stats += fmt.Sprintf(", Trace: %s", result.TraceID)
		}
		stats += "]"
		fmt.Println(stats)
	}

	return nil
}
