package http

import (
	"encoding/json"
	"net/http"
)

// OpenAPI 3.0 specification for 100y-saas API
var openAPISpec = map[string]interface{}{
	"openapi": "3.0.3",
	"info": map[string]interface{}{
		"title":       "100y-saas API",
		"description": "Maintenance-free SaaS platform API - designed to run for 100 years without updates",
		"version":     "1.0.0",
		"contact": map[string]interface{}{
			"name": "100y-saas",
			"url":  "https://github.com/dporkka/100y-saas",
		},
		"license": map[string]interface{}{
			"name": "MIT",
			"url":  "https://opensource.org/licenses/MIT",
		},
	},
	"servers": []map[string]interface{}{
		{
			"url":         "http://localhost:8080",
			"description": "Development server",
		},
		{
			"url":         "https://your-domain.com",
			"description": "Production server",
		},
	},
	"paths": map[string]interface{}{
		"/": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get application dashboard",
				"description": "Returns the main application interface",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "HTML dashboard page",
						"content": map[string]interface{}{
							"text/html": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
		"/healthz": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Health check endpoint",
				"description": "Returns application health status",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Health check response",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/HealthResponse",
								},
							},
						},
					},
				},
			},
		},
		"/auth/register": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Register new user",
				"description": "Create a new user account",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/RegisterRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"201": map[string]interface{}{
						"description": "User created successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UserResponse",
								},
							},
						},
					},
					"400": map[string]interface{}{
						"description": "Invalid request",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/auth/login": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "User login",
				"description": "Authenticate user and create session",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/LoginRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Login successful",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UserResponse",
								},
							},
						},
					},
					"401": map[string]interface{}{
						"description": "Invalid credentials",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/auth/logout": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "User logout",
				"description": "End user session",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Logout successful",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/MessageResponse",
								},
							},
						},
					},
				},
			},
		},
		"/tenants": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "List tenants",
				"description": "Get list of tenants for authenticated user",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"parameters": []map[string]interface{}{
					{
						"name":        "page",
						"in":          "query",
						"description": "Page number",
						"schema": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"default": 1,
						},
					},
					{
						"name":        "limit",
						"in":          "query",
						"description": "Items per page",
						"schema": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 20,
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "List of tenants",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/TenantsResponse",
								},
							},
						},
					},
				},
			},
			"post": map[string]interface{}{
				"summary":     "Create tenant",
				"description": "Create a new tenant",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/CreateTenantRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"201": map[string]interface{}{
						"description": "Tenant created successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/TenantResponse",
								},
							},
						},
					},
				},
			},
		},
		"/analytics": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get analytics data",
				"description": "Retrieve analytics and usage data",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"parameters": []map[string]interface{}{
					{
						"name":        "period",
						"in":          "query",
						"description": "Time period for analytics",
						"schema": map[string]interface{}{
							"type": "string",
							"enum": []string{"day", "week", "month", "year"},
							"default": "week",
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Analytics data",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AnalyticsResponse",
								},
							},
						},
					},
				},
			},
		},
		"/analytics/events": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Track event",
				"description": "Record an analytics event",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/TrackEventRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"201": map[string]interface{}{
						"description": "Event tracked successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/MessageResponse",
								},
							},
						},
					},
				},
			},
		},
		"/export": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Export user data",
				"description": "Export user's data in specified format",
				"security": []map[string]interface{}{
					{"sessionAuth": []string{}},
				},
				"parameters": []map[string]interface{}{
					{
						"name":        "format",
						"in":          "query",
						"description": "Export format",
						"schema": map[string]interface{}{
							"type": "string",
							"enum": []string{"json", "csv"},
							"default": "json",
						},
					},
					{
						"name":        "type",
						"in":          "query",
						"description": "Data type to export",
						"schema": map[string]interface{}{
							"type": "string",
							"enum": []string{"profile", "tenants", "analytics", "all"},
							"default": "all",
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Exported data",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ExportResponse",
								},
							},
							"text/csv": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
	},
	"components": map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"sessionAuth": map[string]interface{}{
				"type":        "apiKey",
				"in":          "cookie",
				"name":        "session_token",
				"description": "Session-based authentication using HTTP cookies",
			},
		},
		"schemas": map[string]interface{}{
			"HealthResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type": "string",
						"example": "healthy",
					},
					"timestamp": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"version": map[string]interface{}{
						"type": "string",
						"example": "1.0.0",
					},
				},
			},
			"RegisterRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"email", "password"},
				"properties": map[string]interface{}{
					"email": map[string]interface{}{
						"type": "string",
						"format": "email",
						"example": "user@example.com",
					},
					"password": map[string]interface{}{
						"type": "string",
						"minLength": 8,
						"example": "secure-password",
					},
					"name": map[string]interface{}{
						"type": "string",
						"example": "John Doe",
					},
				},
			},
			"LoginRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"email", "password"},
				"properties": map[string]interface{}{
					"email": map[string]interface{}{
						"type": "string",
						"format": "email",
						"example": "user@example.com",
					},
					"password": map[string]interface{}{
						"type": "string",
						"example": "secure-password",
					},
				},
			},
			"UserResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "string",
						"example": "user_123",
					},
					"email": map[string]interface{}{
						"type": "string",
						"format": "email",
						"example": "user@example.com",
					},
					"name": map[string]interface{}{
						"type": "string",
						"example": "John Doe",
					},
					"created_at": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
				},
			},
			"CreateTenantRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"name"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
						"example": "My Company",
					},
					"plan": map[string]interface{}{
						"type": "string",
						"enum": []string{"free", "pro", "enterprise"},
						"default": "free",
					},
				},
			},
			"TenantResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "string",
						"example": "tenant_123",
					},
					"name": map[string]interface{}{
						"type": "string",
						"example": "My Company",
					},
					"plan": map[string]interface{}{
						"type": "string",
						"example": "free",
					},
					"created_at": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"owner_id": map[string]interface{}{
						"type": "string",
						"example": "user_123",
					},
				},
			},
			"TenantsResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tenants": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"$ref": "#/components/schemas/TenantResponse",
						},
					},
					"total": map[string]interface{}{
						"type": "integer",
						"example": 10,
					},
					"page": map[string]interface{}{
						"type": "integer",
						"example": 1,
					},
					"limit": map[string]interface{}{
						"type": "integer",
						"example": 20,
					},
				},
			},
			"TrackEventRequest": map[string]interface{}{
				"type": "object",
				"required": []string{"event_type"},
				"properties": map[string]interface{}{
					"event_type": map[string]interface{}{
						"type": "string",
						"example": "page_view",
					},
					"properties": map[string]interface{}{
						"type": "object",
						"additionalProperties": true,
						"example": map[string]interface{}{
							"page": "/dashboard",
							"source": "web",
						},
					},
				},
			},
			"AnalyticsResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type": "string",
						"example": "week",
					},
					"total_events": map[string]interface{}{
						"type": "integer",
						"example": 1250,
					},
					"unique_users": map[string]interface{}{
						"type": "integer",
						"example": 85,
					},
					"top_events": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"event_type": map[string]interface{}{
									"type": "string",
								},
								"count": map[string]interface{}{
									"type": "integer",
								},
							},
						},
					},
				},
			},
			"ExportResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "object",
						"additionalProperties": true,
						"description": "Exported data in requested format",
					},
					"exported_at": map[string]interface{}{
						"type": "string",
						"format": "date-time",
					},
					"format": map[string]interface{}{
						"type": "string",
						"example": "json",
					},
				},
			},
			"MessageResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
						"example": "Operation completed successfully",
					},
				},
			},
			"ErrorResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error": map[string]interface{}{
						"type": "string",
						"example": "Invalid request parameters",
					},
					"code": map[string]interface{}{
						"type": "string",
						"example": "INVALID_REQUEST",
					},
				},
			},
		},
	},
}

// SwaggerUIHTML returns embedded Swagger UI HTML
func SwaggerUIHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>100y-saas API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            })
        }
    </script>
</body>
</html>`
}

// HandleSwagger serves the Swagger UI interface
func (h *Handlers) HandleSwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(SwaggerUIHTML()))
}

// HandleSwaggerJSON serves the OpenAPI JSON specification
func (h *Handlers) HandleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(openAPISpec)
}
