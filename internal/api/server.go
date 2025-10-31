package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/agentregistry-dev/agentregistry/internal/database"
	"github.com/gin-gonic/gin"
)

//go:embed ui/dist/*
var embeddedUI embed.FS

// StartServer starts the API server with embedded UI
func StartServer(port string) error {
	// Initialize database
	if err := database.Initialize(); err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	router := gin.Default()

	// CORS middleware for development
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	api := router.Group("/api")
	{
		api.GET("/registries", getRegistries)
		api.POST("/registries", addRegistry)
		api.DELETE("/registries/:id", removeRegistry)
		api.GET("/servers", getServers)
		api.POST("/servers/:id/install", installServer)
		api.DELETE("/servers/:id/uninstall", uninstallServer)
		api.GET("/skills", getSkills)
		api.GET("/agents", getAgents)
		api.GET("/installations", getInstallations)
		api.GET("/health", healthCheck)
	}

	// Serve embedded UI
	// Try to serve from embedded filesystem
	uiFS, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		// If embedded UI doesn't exist yet (during development), serve a simple message
		router.NoRoute(func(c *gin.Context) {
			c.String(http.StatusOK, "UI not built yet. Run 'make build-ui' to build the Next.js app.")
		})
	} else {
		// Serve static files using http.FileServer for proper Next.js static export handling
		fileServer := http.FileServer(http.FS(uiFS))
		router.NoRoute(func(c *gin.Context) {
			// Let the file server handle the request directly
			// This properly handles index.html, trailing slashes, and static assets
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	return router.Run(":" + port)
}

// API handlers

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "arctl API is running",
	})
}

func getRegistries(c *gin.Context) {
	registries, err := database.GetRegistries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, registries)
}

func getServers(c *gin.Context) {
	servers, err := database.GetServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

func getSkills(c *gin.Context) {
	skills, err := database.GetSkills()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, skills)
}

func getAgents(c *gin.Context) {
	agents, err := database.GetAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agents)
}

func getInstallations(c *gin.Context) {
	installations, err := database.GetInstallations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, installations)
}

func addRegistry(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		URL  string `json:"url" binding:"required"`
		Type string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.AddRegistry(req.Name, req.URL, req.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Registry added successfully"})
}

func removeRegistry(c *gin.Context) {
	id := c.Param("id")

	if err := database.RemoveRegistryByID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registry removed successfully"})
}

func installServer(c *gin.Context) {
	serverID := c.Param("id")

	var req struct {
		Config map[string]string `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.InstallServer(serverID, req.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server installed successfully"})
}

func uninstallServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := database.UninstallServer(serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server uninstalled successfully"})
}
