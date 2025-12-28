package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var meshCmd = &cobra.Command{
	Use:   "mesh",
	Short: "Mesh operations",
	Long:  `Run tasks on the mesh and check status.`,
}

var meshRunCmd = &cobra.Command{
	Use:   "run <task>",
	Short: "Run a task on the mesh",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshRun,
}

var meshStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check mesh status",
	RunE:  runMeshStatus,
}

func init() {
	meshCmd.AddCommand(meshRunCmd)
	meshCmd.AddCommand(meshStatusCmd)
}

func runMeshRun(cmd *cobra.Command, args []string) error {
	input := args[0]

	reqBody, _ := json.Marshal(map[string]string{"input": input})
	resp, err := doRequest("POST", serverURL+"/mesh/run", bytes.NewReader(reqBody))
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

	if output == "json" {
		fmt.Println(string(body))
		return nil
	}

	// Parse and display as text
	var result struct {
		Output    string `json:"output"`
		Duration  string `json:"duration"`
		TokensIn  int    `json:"tokens_in"`
		TokensOut int    `json:"tokens_out"`
		TraceID   string `json:"trace_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Println(result.Output)
	fmt.Printf("\n[Duration: %s, Tokens: %d in / %d out, Trace: %s]\n",
		result.Duration, result.TokensIn, result.TokensOut, result.TraceID)

	return nil
}

func runMeshStatus(cmd *cobra.Command, args []string) error {
	// Check health
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		return fmt.Errorf("server not reachable: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy: %s", string(body))
	}

	fmt.Printf("Server: %s\n", serverURL)
	fmt.Println("Status: OK")

	// Get agent count
	agentResp, err := doRequest("GET", serverURL+"/agents", nil)
	if err != nil {
		return nil // Health check passed, auth might be missing
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

	return nil
}
