package server

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"

	"sitewatch/internal/config"
	"sitewatch/internal/handlers"
	"sitewatch/internal/logger"
	"sitewatch/internal/middleware"
	"sitewatch/internal/models"
	"sitewatch/internal/services/auth"
)

// SetupFiberApp configures and returns the Fiber application
func SetupFiberApp(appState *config.AppState) *fiber.App {
	log := logger.Default().WithComponent("server")
	
	// Initialize authentication service
	authService := auth.NewService(&appState.Config.Auth)
	log.Info("Authentication service initialized", "enabled", authService.IsEnabled())
	// Initialize template engine
	engine := html.New("./web/templates", ".html")
	engine.Reload(true) // Enable auto-reload in development
	engine.Layout("embed") // Use embedded layout system
	
	// Add custom template functions
	engine.AddFunc("printf", fmt.Sprintf)
	engine.AddFunc("formatLatency", func(latency *float64) string {
		if latency == nil {
			return ""
		}
		return fmt.Sprintf("%.1f", *latency)
	})
	engine.AddFunc("dict", func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	})
	engine.AddFunc("until", func(count int) []int {
		result := make([]int, count)
		for i := range result {
			result[i] = i
		}
		return result
	})

	fiberApp := fiber.New(fiber.Config{
		Views:        engine,
		ReadTimeout:  appState.Config.Server.ReadTimeout,
		WriteTimeout: appState.Config.Server.WriteTimeout,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			
			// Log error with structured logging
			requestLog := log.WithRequest(c.Method(), c.Path())
			requestLog.Error("Request error", "error", err, "status_code", code, "user_agent", c.Get("User-Agent"))
			
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// Middleware
	fiberApp.Use(recover.New())
	
	// Performance metrics middleware
	fiberApp.Use(middleware.MetricsMiddleware())
	
	// Custom structured logging middleware
	fiberApp.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Continue to next middleware
		err := c.Next()
		
		// Log request
		duration := time.Since(start)
		requestLog := log.WithRequest(c.Method(), c.Path())
		
		if err != nil {
			requestLog.Error("Request completed with error", 
				"status", c.Response().StatusCode(),
				"duration_ms", duration.Milliseconds(),
				"user_agent", c.Get("User-Agent"),
				"remote_addr", c.IP(),
				"error", err)
		} else {
			requestLog.Info("Request completed", 
				"status", c.Response().StatusCode(),
				"duration_ms", duration.Milliseconds(),
				"user_agent", c.Get("User-Agent"),
				"remote_addr", c.IP())
		}
		
		return err
	})
	
	fiberApp.Use(cors.New())

	// Health check endpoint - accessible with metrics permission
	fiberApp.Get("/health", 
		middleware.APIAuthMiddleware(authService, models.PermissionMetrics), 
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"status":  "ok",
				"uptime":  time.Since(appState.StartTime).Seconds(),
				"version": "1.0.0",
			})
		})

	// Static files
	fiberApp.Static("/static", "./web/static")
	
	// UI Routes (Public - with session management)
	fiberApp.Get("/", func(c *fiber.Ctx) error {
		// Set UI session cookie if auth is enabled
		if authService.IsEnabled() {
			sessionName := authService.GetUISessionName()
			if c.Cookies(sessionName) == "" {
				expiry := authService.GetUISessionExpiry()
				c.Cookie(&fiber.Cookie{
					Name:     sessionName,
					Value:    appState.Config.Auth.UI.Secret,
					Expires:  time.Now().Add(expiry),
					HTTPOnly: true,
					SameSite: "Strict",
					Secure:   false, // Set to true in production with HTTPS
				})
			}
		}
		return handlers.HandleDashboard(c)
	})
	fiberApp.Get("/dashboard", func(c *fiber.Ctx) error {
		// Set UI session cookie if auth is enabled
		if authService.IsEnabled() {
			sessionName := authService.GetUISessionName()
			if c.Cookies(sessionName) == "" {
				expiry := authService.GetUISessionExpiry()
				c.Cookie(&fiber.Cookie{
					Name:     sessionName,
					Value:    appState.Config.Auth.UI.Secret,
					Expires:  time.Now().Add(expiry),
					HTTPOnly: true,
					SameSite: "Strict",
					Secure:   false, // Set to true in production with HTTPS
				})
			}
		}
		return handlers.HandleDashboard(c)
	})

	// UI Fragment Routes (for HTMX) - Protected with UI session
	ui := fiberApp.Group("/ui", middleware.UIAuthMiddleware(authService))
	ui.Get("/overview", handlers.HandleUIOverview)
	ui.Get("/sites", handlers.HandleUISites)
	ui.Get("/details/:siteId", handlers.HandleUIDetails)
	ui.Get("/enhanced-fragment/:siteId", handlers.HandleUIEnhancedFragment)
	ui.Get("/chart-data/:siteId/:chartType/:range", handlers.HandleUIChartData)
	ui.Get("/logs", handlers.HandleUILogs)
	ui.Get("/logs-table", handlers.HandleUILogsTable)
	ui.Post("/test/:siteId", handlers.HandleSiteTest)

	// API Routes - Protected with API tokens
	api := fiberApp.Group("/api")

	// Sites endpoints (read permission required)
	apiRead := api.Group("", middleware.APIAuthMiddleware(authService, models.PermissionRead))
	apiRead.Get("/sites", handlers.HandleGetSites)
	apiRead.Get("/sites/:siteId/status", handlers.HandleGetSiteStatus)
	apiRead.Get("/sites/:siteId/details", handlers.HandleGetSiteDetails)
	apiRead.Get("/sites/:siteId/statistics", handlers.HandleGetSiteStatistics)
	apiRead.Get("/sites/:siteId/charts", handlers.HandleGetSiteChartData)
	apiRead.Get("/logs", handlers.HandleGetLogs)
	// Health endpoint also available for read tokens
	apiRead.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"uptime":  time.Since(appState.StartTime).Seconds(),
			"version": "1.0.0",
		})
	})
	
	// Test endpoints (test permission required)
	apiTest := api.Group("", middleware.APIAuthMiddleware(authService, models.PermissionTest))
	apiTest.Post("/sites/:siteId/test", handlers.HandleSiteTest)

	// Metrics endpoint (Prometheus format) - Protected with metrics permission
	if appState.Config.Metrics.Enabled {
		fiberApp.Get(appState.Config.Metrics.Path, 
			middleware.APIAuthMiddleware(authService, models.PermissionMetrics), 
			handlers.HandlePrometheusMetrics)
	}

	return fiberApp
}