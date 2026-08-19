package controllers

import (
	"net/http"
	"novaflow/core"
)

type DocsController struct{}

func NewDocsController() *DocsController {
	return &DocsController{}
}

// ShowDocs renders the interactive Swagger UI page.
func (ctl *DocsController) ShowDocs(c *core.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>NovaFlow API Documentation 🚀</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    html {
      box-sizing: border-box;
      overflow: -grow-y;
    }
    *, *:before, *:after {
      box-sizing: inherit;
    }
    body {
      margin: 0;
      background: #0f172a;
    }
    /* Dark mode inversion filter for default white Swagger UI theme */
    .swagger-ui {
      filter: invert(0.9) hue-rotate(180deg);
      background-color: #fafafa; /* Will be inverted to dark */
      padding: 24px;
      font-family: sans-serif;
    }
    .swagger-ui .topbar {
      display: none;
    }
    .swagger-ui .info .title {
      color: #030712 !important; /* Will be inverted to light */
    }
    .swagger-ui .info p, .swagger-ui .info li, .swagger-ui .info td {
      color: #374151 !important; /* Will be inverted to muted */
    }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" charset="UTF-8"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.presets.standalone
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
	c.HTML(http.StatusOK, html)
}

// OpenAPISpec returns the OpenAPI 3.0.0 JSON specification mapping all endpoints.
func (ctl *DocsController) OpenAPISpec(c *core.Context) {
	spec := `{
  "openapi": "3.0.0",
  "info": {
    "title": "NovaFlow Go API Documentation",
    "version": "1.0.0",
    "description": "Interactive API Documentation for NovaFlow Go MVC Framework. Includes JWT Bearer authentication and full CRUD resource actions."
  },
  "servers": [
    {
      "url": "http://localhost:8080",
      "description": "Local development server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Enter your JWT token obtained from /api/v1/login"
      }
    }
  },
  "security": [
    {
      "BearerAuth": []
    }
  ],
  "paths": {
    "/api/v1/register": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Register a new user",
        "security": [],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name", "email", "password"],
                "properties": {
                  "name": { "type": "string", "example": "John Doe" },
                  "email": { "type": "string", "example": "john@example.com" },
                  "password": { "type": "string", "example": "secret123" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "User registered successfully"
          }
        }
      }
    },
    "/api/v1/login": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Login to obtain a JWT token",
        "security": [],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "example": "john@example.com" },
                  "password": { "type": "string", "example": "secret123" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Login successful, returns Bearer token"
          }
        }
      }
    },
    "/api/v1/me": {
      "get": {
        "tags": ["User Profile"],
        "summary": "Get authenticated user details",
        "responses": {
          "200": {
            "description": "User details"
          }
        }
      }
    },
    "/api/v1/products": {
      "get": {
        "tags": ["Products CRUD"],
        "summary": "Get list of all products",
        "responses": {
          "200": {
            "description": "List of products"
          }
        }
      },
      "post": {
        "tags": ["Products CRUD"],
        "summary": "Create a new product",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name", "price"],
                "properties": {
                  "name": { "type": "string", "example": "Smartphone" },
                  "price": { "type": "number", "example": 699.99 }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Product created successfully"
          }
        }
      }
    },
    "/api/v1/products/{id}": {
      "get": {
        "tags": ["Products CRUD"],
        "summary": "Get a specific product by ID",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": { "type": "integer" }
          }
        ],
        "responses": {
          "200": {
            "description": "Product details"
          }
        }
      },
      "put": {
        "tags": ["Products CRUD"],
        "summary": "Update an existing product by ID",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": { "type": "integer" }
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": { "type": "string", "example": "Updated Smartphone" },
                  "price": { "type": "number", "example": 649.99 }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Product updated successfully"
          }
        }
      },
      "delete": {
        "tags": ["Products CRUD"],
        "summary": "Delete a product by ID",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": { "type": "integer" }
          }
        ],
        "responses": {
          "200": {
            "description": "Product deleted successfully"
          }
        }
      }
    }
  }
}`
	c.Writer.Header().Set("Content-Type", "application/json")
	c.String(http.StatusOK, spec)
}
