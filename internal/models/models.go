package models

import "time"

// Registry represents a connected registry
type Registry struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Type      string    `json:"type"` // public, private
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServerDetail represents an MCP server from the registry
// Based on the MCP server.json schema
type ServerDetail struct {
	ID          int       `json:"id"`
	RegistryID  int       `json:"registry_id"`
	Name        string    `json:"name"`        // e.g., "io.github.user/weather"
	Title       string    `json:"title"`       // Optional display name
	Description string    `json:"description"` // Clear explanation of functionality
	Version     string    `json:"version"`     // Semantic version
	WebsiteURL  string    `json:"website_url"` // Optional homepage
	Installed   bool      `json:"installed"`
	Data        string    `json:"data"` // JSON blob of full server.json
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Skill represents a skill from the registry
type Skill struct {
	ID          int       `json:"id"`
	RegistryID  int       `json:"registry_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Installed   bool      `json:"installed"`
	Data        string    `json:"data"` // JSON blob
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Installation represents an installed resource (MCP server or skill)
type Installation struct {
	ID           int       `json:"id"`
	ResourceType string    `json:"resource_type"` // mcp, skill
	ResourceID   int       `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	Version      string    `json:"version"`
	Config       string    `json:"config"` // JSON blob for configuration
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
