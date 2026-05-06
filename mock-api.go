package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Mock data structures
type AgentRegistry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Type        string `json:"type"`
	Identifier  string `json:"identifier"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type GitHubSubmission struct {
	Repository   string `json:"repository"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Manifest     string `json:"manifest"`
}

var mockRegistries = []AgentRegistry{
	{
		ID:          "example-agent-1",
		Name:        "Example AI Agent",
		Namespace:   "default",
		Type:        "ai-agent",
		Identifier:  "example-agent",
		Category:    "productivity",
		Description: "An example AI agent for demonstration",
		Status:      "active",
	},
	{
		ID:          "data-analyzer-2",
		Name:        "Data Analyzer",
		Namespace:   "analytics",
		Type:        "data-processor",
		Identifier:  "data-analyzer",
		Category:    "analytics",
		Description: "Analyzes large datasets and provides insights",
		Status:      "active",
	},
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func registriesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRegistries)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method == "POST" {
		var submission GitHubSubmission
		if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Mock response
		response := map[string]interface{}{
			"status":  "success",
			"message": "Resource submitted successfully to GitHub repository",
			"pr_url":  fmt.Sprintf("https://github.com/%s/pull/123", submission.Repository),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	http.HandleFunc("/api/registries", registriesHandler)
	http.HandleFunc("/api/submit", submitHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/readyz", healthHandler)
	
	fmt.Println("Mock Agent Registry API server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}