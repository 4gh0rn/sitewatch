package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"sitewatch/internal/services/auth"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "generate":
		generateToken()
	case "list":
		listTokens()
	case "ui-secret":
		generateUISecret()
	case "example":
		showExample()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SiteWatch Token Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run tools/token-gen/main.go <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  generate    Generate a new API token")
	fmt.Println("  list        List configured tokens (from config)")
	fmt.Println("  ui-secret   Generate a new UI secret")
	fmt.Println("  example     Show authentication configuration example")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run tools/token-gen/main.go generate --name=\"Telegraf\" --permissions=\"metrics\"")
	fmt.Println("  go run tools/token-gen/main.go ui-secret")
}

func generateToken() {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	name := fs.String("name", "", "Token name/description (required)")
	permissions := fs.String("permissions", "metrics", "Comma-separated permissions (metrics,read,test,admin)")
	expires := fs.String("expires", "", "Expiration date (YYYY-MM-DD format, optional)")
	prefix := fs.String("prefix", "sw", "Token prefix")

	fs.Parse(os.Args[2:])

	if *name == "" {
		fmt.Println("Error: --name is required")
		fs.Usage()
		os.Exit(1)
	}

	// Generate token
	token, err := auth.GenerateToken(*prefix)
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		os.Exit(1)
	}

	// Parse permissions
	permList := strings.Split(*permissions, ",")
	for i, perm := range permList {
		permList[i] = strings.TrimSpace(perm)
	}

	// Output YAML format
	fmt.Printf("# Add this to your configs/config.yaml under auth.api.tokens:\n")
	fmt.Printf("- token: \"%s\"\n", token)
	fmt.Printf("  name: \"%s\"\n", *name)
	fmt.Printf("  permissions: [%s]\n", strings.Join(permList, ", "))
	if *expires != "" {
		// Validate date format
		if _, err := time.Parse("2006-01-02", *expires); err != nil {
			fmt.Printf("Error: Invalid date format. Use YYYY-MM-DD\n")
			os.Exit(1)
		}
		fmt.Printf("  expires: \"%s\"\n", *expires)
	}
	fmt.Printf("  created: \"%s\"\n", time.Now().Format("2006-01-02T15:04:05Z07:00"))
	fmt.Println()
	fmt.Println("Token Details:")
	fmt.Printf("  Token: %s\n", token)
	fmt.Printf("  Name: %s\n", *name)
	fmt.Printf("  Permissions: %s\n", strings.Join(permList, ", "))
	if *expires != "" {
		fmt.Printf("  Expires: %s\n", *expires)
	} else {
		fmt.Printf("  Expires: Never\n")
	}
	fmt.Println()
	fmt.Println("Usage example:")
	fmt.Printf("  curl -H \"Authorization: Bearer %s\" http://localhost:8080/api/sites\n", token)
}

func generateUISecret() {
	secret, err := auth.GenerateUISecret()
	if err != nil {
		fmt.Printf("Error generating UI secret: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("# Add this to your configs/config.yaml under auth.ui:\n")
	fmt.Printf("ui:\n")
	fmt.Printf("  secret: \"%s\"\n", secret)
	fmt.Printf("  session_name: \"sitewatch_session\"\n")
	fmt.Printf("  expires_hours: 24\n")
	fmt.Println()
	fmt.Println("UI Secret Details:")
	fmt.Printf("  Secret: %s\n", secret)
	fmt.Printf("  Length: %d characters\n", len(secret))
	fmt.Println("  This secret will be used for UI session cookies.")
}

func listTokens() {
	fmt.Println("To list configured tokens, check your configs/config.yaml file under:")
	fmt.Println("  auth:")
	fmt.Println("    api:")
	fmt.Println("      tokens:")
	fmt.Println()
	fmt.Println("Example configuration structure:")
	showExample()
}

func showExample() {
	fmt.Println("# Authentication Configuration Example")
	fmt.Println("# Add this to configs/config.yaml:")
	fmt.Println()
	fmt.Println("auth:")
	fmt.Println("  enabled: true")
	fmt.Println("  ui:")
	fmt.Println("    secret: \"your-generated-ui-secret-here\"")
	fmt.Println("    session_name: \"sitewatch_session\"")
	fmt.Println("    expires_hours: 24")
	fmt.Println("  api:")
	fmt.Println("    tokens:")
	fmt.Println("      - token: \"sw_telegraf_a1b2c3d4e5f6...\"")
	fmt.Println("        name: \"Telegraf Monitoring\"")
	fmt.Println("        permissions: [\"metrics\"]")
	fmt.Println("        expires: \"2025-12-31\"")
	fmt.Println("      - token: \"sw_admin_f6e5d4c3b2a1...\"")
	fmt.Println("        name: \"Admin Access\"")
	fmt.Println("        permissions: [\"admin\"]")
	fmt.Println("        # expires: null  # Never expires")
	fmt.Println()
	fmt.Println("Available permissions:")
	fmt.Println("  - metrics: Access to /metrics, /health only")
	fmt.Println("  - read:    Access to /api/sites, /api/logs, /api/health")
	fmt.Println("  - test:    Access to read endpoints + /api/sites/:id/test")
	fmt.Println("  - admin:   Access to all endpoints (includes all permissions)")
	fmt.Println()
	fmt.Println("Usage Examples:")
	fmt.Println("  # Generate tokens:")
	fmt.Println("  make token-generate name=\"Telegraf\" permissions=\"metrics\"")
	fmt.Println("  make ui-secret-generate")
	fmt.Println()
	fmt.Println("  # Use API token:")
	fmt.Println("  curl -H \"Authorization: Bearer sw_token_...\" http://localhost:8080/api/sites")
}