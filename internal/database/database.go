package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentregistry-dev/agentregistry/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// Initialize sets up the SQLite database
func Initialize() error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .arctl directory if it doesn't exist
	arctlDir := filepath.Join(homeDir, ".arctl")
	if err := os.MkdirAll(arctlDir, 0755); err != nil {
		return fmt.Errorf("failed to create .arctl directory: %w", err)
	}

	// Open database connection
	dbPath := filepath.Join(arctlDir, "arctl.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	DB = db

	// Enable foreign key constraints (disabled by default in SQLite)
	if _, err := DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS registries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		url TEXT NOT NULL,
		type TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		registry_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		title TEXT,
		description TEXT NOT NULL,
		version TEXT NOT NULL,
		website_url TEXT,
		installed BOOLEAN DEFAULT 0,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (registry_id) REFERENCES registries(id) ON DELETE CASCADE,
		UNIQUE(registry_id, name, version)
	);

	CREATE TABLE IF NOT EXISTS skills (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		registry_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		title TEXT,
		description TEXT NOT NULL,
		version TEXT NOT NULL,
		category TEXT,
		installed BOOLEAN DEFAULT 0,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (registry_id) REFERENCES registries(id) ON DELETE CASCADE,
		UNIQUE(registry_id, name, version)
	);

	CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		registry_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		title TEXT,
		description TEXT NOT NULL,
		version TEXT NOT NULL,
		model TEXT,
		specialty TEXT,
		installed BOOLEAN DEFAULT 0,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (registry_id) REFERENCES registries(id) ON DELETE CASCADE,
		UNIQUE(registry_id, name, version)
	);

	CREATE TABLE IF NOT EXISTS installations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		resource_type TEXT NOT NULL,
		resource_id INTEGER NOT NULL,
		resource_name TEXT NOT NULL,
		version TEXT NOT NULL,
		config TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(resource_type, resource_name)
	);

	CREATE INDEX IF NOT EXISTS idx_servers_registry ON servers(registry_id);
	CREATE INDEX IF NOT EXISTS idx_skills_registry ON skills(registry_id);
	CREATE INDEX IF NOT EXISTS idx_agents_registry ON agents(registry_id);
	CREATE INDEX IF NOT EXISTS idx_servers_installed ON servers(installed);
	CREATE INDEX IF NOT EXISTS idx_skills_installed ON skills(installed);
	CREATE INDEX IF NOT EXISTS idx_agents_installed ON agents(installed);
	`

	_, err := DB.Exec(schema)
	return err
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// GetRegistries returns all connected registries
func GetRegistries() ([]models.Registry, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT id, name, url, type, created_at, updated_at 
		FROM registries 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query registries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var registries []models.Registry
	for rows.Next() {
		var r models.Registry
		if err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.Type, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan registry: %w", err)
		}
		registries = append(registries, r)
	}

	// Return empty array instead of nil if no registries found
	if registries == nil {
		return []models.Registry{}, nil
	}

	return registries, nil
}

// GetServers returns all MCP servers from connected registries
func GetServers() ([]models.ServerDetail, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT s.id, s.registry_id, r.name, s.name, s.title, s.description, s.version, s.website_url, s.installed, s.data, s.created_at, s.updated_at 
		FROM servers s
		JOIN registries r ON s.registry_id = r.id
		ORDER BY s.name, s.version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query servers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []models.ServerDetail
	for rows.Next() {
		var s models.ServerDetail
		if err := rows.Scan(&s.ID, &s.RegistryID, &s.RegistryName, &s.Name, &s.Title, &s.Description, &s.Version, &s.WebsiteURL, &s.Installed, &s.Data, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, s)
	}

	// Return empty array instead of nil if no servers found
	if servers == nil {
		return []models.ServerDetail{}, nil
	}

	return servers, nil
}

// GetServerByName returns a server by name
func GetServerByName(name string) (*models.ServerDetail, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var s models.ServerDetail
	err := DB.QueryRow(`
		SELECT s.id, s.registry_id, r.name, s.name, s.title, s.description, s.version, s.website_url, s.installed, s.data, s.created_at, s.updated_at 
		FROM servers s
		JOIN registries r ON s.registry_id = r.id
		WHERE s.name = ?
		ORDER BY s.version DESC
		LIMIT 1
	`, name).Scan(&s.ID, &s.RegistryID, &s.RegistryName, &s.Name, &s.Title, &s.Description, &s.Version, &s.WebsiteURL, &s.Installed, &s.Data, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query server: %w", err)
	}

	return &s, nil
}

// GetSkills returns all skills from connected registries
func GetSkills() ([]models.Skill, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT sk.id, sk.registry_id, r.name, sk.name, COALESCE(sk.title, ''), sk.description, sk.version, COALESCE(sk.category, ''), sk.installed, sk.data, sk.created_at, sk.updated_at 
		FROM skills sk
		JOIN registries r ON sk.registry_id = r.id
		ORDER BY sk.name, sk.version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query skills: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var skills []models.Skill
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.RegistryID, &s.RegistryName, &s.Name, &s.Title, &s.Description, &s.Version, &s.Category, &s.Installed, &s.Data, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, s)
	}

	// Return empty array instead of nil if no skills found
	if skills == nil {
		return []models.Skill{}, nil
	}

	return skills, nil
}

// GetSkillByName returns a skill by name
func GetSkillByName(name string) (*models.Skill, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var s models.Skill
	err := DB.QueryRow(`
		SELECT sk.id, sk.registry_id, r.name, sk.name, COALESCE(sk.title, ''), sk.description, sk.version, COALESCE(sk.category, ''), sk.installed, sk.data, sk.created_at, sk.updated_at 
		FROM skills sk
		JOIN registries r ON sk.registry_id = r.id
		WHERE sk.name = ?
		ORDER BY sk.version DESC
		LIMIT 1
	`, name).Scan(&s.ID, &s.RegistryID, &s.RegistryName, &s.Name, &s.Title, &s.Description, &s.Version, &s.Category, &s.Installed, &s.Data, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query skill: %w", err)
	}

	return &s, nil
}

// GetAgents returns all agents from connected registries
func GetAgents() ([]models.Agent, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT id, registry_id, name, COALESCE(title, ''), description, version, COALESCE(model, ''), COALESCE(specialty, ''), installed, data, created_at, updated_at 
		FROM agents 
		ORDER BY name, version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.RegistryID, &a.Name, &a.Title, &a.Description, &a.Version, &a.Model, &a.Specialty, &a.Installed, &a.Data, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, a)
	}

	// Return empty array instead of nil if no agents found
	if agents == nil {
		return []models.Agent{}, nil
	}

	return agents, nil
}

// GetInstallations returns all installed resources
func GetInstallations() ([]models.Installation, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT id, resource_type, resource_id, resource_name, version, config, created_at, updated_at 
		FROM installations 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query installations: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var installations []models.Installation
	for rows.Next() {
		var i models.Installation
		if err := rows.Scan(&i.ID, &i.ResourceType, &i.ResourceID, &i.ResourceName, &i.Version, &i.Config, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan installation: %w", err)
		}
		installations = append(installations, i)
	}

	// Return empty array instead of nil if no installations found
	if installations == nil {
		return []models.Installation{}, nil
	}

	return installations, nil
}

// AddRegistry adds a new registry
func AddRegistry(name, url, registryType string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := DB.Exec(`
		INSERT INTO registries (name, url, type) 
		VALUES (?, ?, ?)
	`, name, url, registryType)
	if err != nil {
		return fmt.Errorf("failed to add registry: %w", err)
	}

	return nil
}

// GetRegistryByName returns a registry by name
func GetRegistryByName(name string) (*models.Registry, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var r models.Registry
	err := DB.QueryRow(`
		SELECT id, name, url, type, created_at, updated_at 
		FROM registries 
		WHERE name = ?
	`, name).Scan(&r.ID, &r.Name, &r.URL, &r.Type, &r.CreatedAt, &r.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query registry: %w", err)
	}

	return &r, nil
}

// RemoveRegistry removes a registry
func RemoveRegistry(name string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	result, err := DB.Exec(`DELETE FROM registries WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to remove registry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("registry not found: %s", name)
	}

	return nil
}

// AddOrUpdateServer adds or updates a server in the database
func AddOrUpdateServer(registryID int, name, title, description, version, websiteURL, data string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Use INSERT OR REPLACE to handle duplicates
	_, err := DB.Exec(`
		INSERT INTO servers (registry_id, name, title, description, version, website_url, data, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(registry_id, name, version) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			website_url = excluded.website_url,
			data = excluded.data,
			updated_at = CURRENT_TIMESTAMP
	`, registryID, name, title, description, version, websiteURL, data)
	if err != nil {
		return fmt.Errorf("failed to add/update server: %w", err)
	}

	return nil
}

// ClearRegistryServers removes all servers for a specific registry
func ClearRegistryServers(registryID int) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := DB.Exec(`DELETE FROM servers WHERE registry_id = ?`, registryID)
	if err != nil {
		return fmt.Errorf("failed to clear servers: %w", err)
	}

	return nil
}

// RemoveRegistryByID removes a registry by ID
func RemoveRegistryByID(id string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	result, err := DB.Exec(`DELETE FROM registries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to remove registry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("registry not found with id: %s", id)
	}

	return nil
}

// InstallServer marks a server as installed and stores its configuration
func InstallServer(serverID string, config map[string]string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Start a transaction
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Get server details
	var server models.ServerDetail
	err = tx.QueryRow(`
		SELECT id, name, version, data 
		FROM servers 
		WHERE id = ?
	`, serverID).Scan(&server.ID, &server.Name, &server.Version, &server.Data)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Mark server as installed
	_, err = tx.Exec(`UPDATE servers SET installed = 1 WHERE id = ?`, serverID)
	if err != nil {
		return fmt.Errorf("failed to mark server as installed: %w", err)
	}

	// Marshal config to JSON
	configJSON := "{}"
	if len(config) > 0 {
		configBytes, err := marshalConfig(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		configJSON = string(configBytes)
	}

	// Add installation record
	_, err = tx.Exec(`
		INSERT INTO installations (resource_type, resource_id, resource_name, version, config)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(resource_type, resource_name) DO UPDATE SET
			version = excluded.version,
			config = excluded.config,
			updated_at = CURRENT_TIMESTAMP
	`, "mcp", server.ID, server.Name, server.Version, configJSON)
	if err != nil {
		return fmt.Errorf("failed to add installation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UninstallServer marks a server as uninstalled and removes its installation record
func UninstallServer(serverID string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Get server name
	var serverName string
	err = tx.QueryRow(`SELECT name FROM servers WHERE id = ?`, serverID).Scan(&serverName)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Mark server as not installed
	_, err = tx.Exec(`UPDATE servers SET installed = 0 WHERE id = ?`, serverID)
	if err != nil {
		return fmt.Errorf("failed to mark server as uninstalled: %w", err)
	}

	// Remove installation record
	_, err = tx.Exec(`DELETE FROM installations WHERE resource_type = ? AND resource_name = ?`, "mcp", serverName)
	if err != nil {
		return fmt.Errorf("failed to remove installation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func marshalConfig(config map[string]string) ([]byte, error) {
	// Simple JSON marshaling
	result := "{"
	first := true
	for k, v := range config {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%s"`, k, v)
		first = false
	}
	result += "}"
	return []byte(result), nil
}

// MarkServerInstalled marks a server as installed or uninstalled
func MarkServerInstalled(serverID int, installed bool) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	query := `UPDATE servers SET installed = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := DB.Exec(query, installed, serverID)
	return err
}

// MarkSkillInstalled marks a skill as installed or uninstalled
func MarkSkillInstalled(skillID int, installed bool) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	query := `UPDATE skills SET installed = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := DB.Exec(query, installed, skillID)
	return err
}

// GetInstalledServers returns all installed MCP servers
func GetInstalledServers() ([]models.ServerDetail, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query(`
		SELECT s.id, s.registry_id, r.name, s.name, s.title, s.description, s.version, s.website_url, s.installed, s.data, s.created_at, s.updated_at 
		FROM servers s
		JOIN registries r ON s.registry_id = r.id
		WHERE s.installed = 1
		ORDER BY s.name, s.version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query installed servers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []models.ServerDetail
	for rows.Next() {
		var s models.ServerDetail
		if err := rows.Scan(&s.ID, &s.RegistryID, &s.RegistryName, &s.Name, &s.Title, &s.Description, &s.Version, &s.WebsiteURL, &s.Installed, &s.Data, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, s)
	}

	// Return empty array instead of nil if no servers found
	if servers == nil {
		return []models.ServerDetail{}, nil
	}

	return servers, nil
}

// GetInstallationByName returns an installation record by resource name
func GetInstallationByName(resourceType, resourceName string) (*models.Installation, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var i models.Installation
	err := DB.QueryRow(`
		SELECT id, resource_type, resource_id, resource_name, version, config, created_at, updated_at 
		FROM installations 
		WHERE resource_type = ? AND resource_name = ?
	`, resourceType, resourceName).Scan(&i.ID, &i.ResourceType, &i.ResourceID, &i.ResourceName, &i.Version, &i.Config, &i.CreatedAt, &i.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query installation: %w", err)
	}

	return &i, nil
}
