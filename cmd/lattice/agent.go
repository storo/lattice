package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
	Long:  `List, inspect, and run agents on the server.`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents",
	RunE:  runAgentList,
}

var agentInfoCmd = &cobra.Command{
	Use:   "info <agent-id>",
	Short: "Get agent information",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentInfo,
}

var agentRunCmd = &cobra.Command{
	Use:   "run <agent-id> <input>",
	Short: "Run an agent with input",
	Args:  cobra.ExactArgs(2),
	RunE:  runAgentRun,
}

func init() {
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentInfoCmd)
	agentCmd.AddCommand(agentRunCmd)
}

func runAgentList(cmd *cobra.Command, args []string) error {
	resp, err := doRequest("GET", serverURL+"/agents", nil)
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
		Agents []struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Provides []string `json:"provides"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Printf("%-36s  %-20s  %s\n", "ID", "NAME", "CAPABILITIES")
	fmt.Println("------------------------------------  --------------------  --------------------")
	for _, a := range result.Agents {
		caps := ""
		for i, c := range a.Provides {
			if i > 0 {
				caps += ", "
			}
			caps += c
		}
		fmt.Printf("%-36s  %-20s  %s\n", a.ID, a.Name, caps)
	}

	return nil
}

func runAgentInfo(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	resp, err := doRequest("GET", serverURL+"/agents/"+agentID, nil)
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
	var agent struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Provides    []string `json:"provides"`
		Needs       []string `json:"needs"`
	}
	if err := json.Unmarshal(body, &agent); err != nil {
		return err
	}

	fmt.Printf("ID:          %s\n", agent.ID)
	fmt.Printf("Name:        %s\n", agent.Name)
	fmt.Printf("Description: %s\n", agent.Description)
	fmt.Printf("Provides:    %v\n", agent.Provides)
	fmt.Printf("Needs:       %v\n", agent.Needs)

	return nil
}

func runAgentRun(cmd *cobra.Command, args []string) error {
	agentID := args[0]
	input := args[1]

	reqBody, _ := json.Marshal(map[string]string{"input": input})
	resp, err := doRequest("POST", serverURL+"/agents/"+agentID+"/run", bytes.NewReader(reqBody))
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
		Output   string `json:"output"`
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Println(result.Output)
	fmt.Printf("\n[Duration: %s]\n", result.Duration)

	return nil
}

func doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}
