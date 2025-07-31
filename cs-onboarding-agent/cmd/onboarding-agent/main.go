package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	
	"github.com/openshift-online/ocm-cluster-service/pkg/onboarding"
	"github.com/openshift-online/ocm-cluster-service/pkg/servicelog"
)

var (
	port        string
	interactive bool
	userID      string
	username    string
	email       string
	apiURL      string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "onboarding-agent",
		Short: "Team onboarding agent for CS service",
		Long:  "An interactive onboarding agent that guides new team members through setup and first tasks",
	}

	// Server command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the onboarding agent HTTP server",
		Run:   runServer,
	}
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "Server port")

	// Interactive CLI command
	interactiveCmd := &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive onboarding session",
		Run:   runInteractive,
	}
	interactiveCmd.Flags().StringVar(&userID, "user-id", "", "User ID for onboarding")
	interactiveCmd.Flags().StringVar(&username, "username", "", "Username for onboarding")
	interactiveCmd.Flags().StringVar(&email, "email", "", "Email for onboarding")
	interactiveCmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "Onboarding API URL")

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status [session-id]",
		Short: "Get onboarding session status",
		Args:  cobra.ExactArgs(1),
		Run:   runStatus,
	}
	statusCmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "Onboarding API URL")

	rootCmd.AddCommand(serverCmd, interactiveCmd, statusCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	// Initialize logging
	logger, err := logging.NewGoLoggerBuilder().
		Debug(true).
		Build()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Initialize OCM SDK connection for service logging
	connection, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Build()
	if err != nil {
		log.Fatalf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Create service log client
	serviceLogClient := &servicelog.ServiceLogClient{
		GatewayConnection: connection,
		Logger:           logger,
	}

	// Create onboarding service
	onboardingService := onboarding.NewOnboardingService(serviceLogClient, logger)

	// Setup HTTP router
	router := mux.NewRouter()
	router.Use(onboardingService.LoggingMiddleware)
	
	// Register onboarding routes
	onboardingService.RegisterRoutes(router)

	// Add CORS headers for development
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start session cleanup background task
	go onboardingService.StartSessionCleanup(ctx)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info(ctx, "Shutting down server...")
		
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error(ctx, "Failed to shutdown server: %v", err)
		}
	}()

	logger.Info(ctx, "Starting onboarding agent server on port %s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func runInteractive(cmd *cobra.Command, args []string) {
	if userID == "" || username == "" || email == "" {
		fmt.Println("Please provide --user-id, --username, and --email")
		os.Exit(1)
	}

	fmt.Printf("🎉 Welcome to the CS Team Onboarding Agent!\n")
	fmt.Printf("Starting interactive session for %s (%s)\n\n", username, email)

	// Start onboarding session
	sessionID, err := startOnboardingSession(apiURL, userID, username, email)
	if err != nil {
		fmt.Printf("Failed to start onboarding session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Type 'help' for available commands, 'status' for progress, or 'quit' to exit\n\n")

	// Interactive chat loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}
		
		if message == "quit" || message == "exit" {
			fmt.Println("Goodbye! You can resume your onboarding session later.")
			break
		}

		if message == "status" {
			if err := showStatus(apiURL, sessionID); err != nil {
				fmt.Printf("Failed to get status: %v\n", err)
			}
			continue
		}

		// Send message to onboarding agent
		response, err := sendMessage(apiURL, sessionID, message)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\nAgent: %s\n", response.Message)
		
		if len(response.NextActions) > 0 {
			fmt.Println("\nNext actions:")
			for _, action := range response.NextActions {
				fmt.Printf("  • %s\n", action)
			}
		}
		
		fmt.Printf("\nProgress: %.0f%% complete\n", response.Progress*100)
		fmt.Println(strings.Repeat("-", 50))
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	sessionID := args[0]
	if err := showStatus(apiURL, sessionID); err != nil {
		fmt.Printf("Failed to get status: %v\n", err)
		os.Exit(1)
	}
}

type StartSessionResponse struct {
	SessionID string  `json:"session_id"`
	Message   string  `json:"message"`
	Stage     string  `json:"stage"`
	Progress  float64 `json:"progress"`
}

type MessageResponse struct {
	Message     string   `json:"message"`
	NextActions []string `json:"next_actions"`
	Stage       string   `json:"stage"`
	Progress    float64  `json:"progress"`
}

type ApiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func startOnboardingSession(apiURL, userID, username, email string) (string, error) {
	payload := map[string]string{
		"user_id":  userID,
		"username": username,
		"email":    email,
	}
	
	jsonData, _ := json.Marshal(payload)
	
	resp, err := http.Post(apiURL+"/api/v1/onboarding/start", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}

	if !apiResp.Success {
		return "", fmt.Errorf("API error: %s", apiResp.Error)
	}

	var sessionResp StartSessionResponse
	if err := json.Unmarshal(apiResp.Data, &sessionResp); err != nil {
		return "", err
	}

	fmt.Printf("Agent: %s\n", sessionResp.Message)
	return sessionResp.SessionID, nil
}

func sendMessage(apiURL, sessionID, message string) (*MessageResponse, error) {
	payload := map[string]string{
		"session_id": sessionID,
		"message":    message,
	}
	
	jsonData, _ := json.Marshal(payload)
	
	resp, err := http.Post(apiURL+"/api/v1/onboarding/message", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	var msgResp MessageResponse
	if err := json.Unmarshal(apiResp.Data, &msgResp); err != nil {
		return nil, err
	}

	return &msgResp, nil
}

func showStatus(apiURL, sessionID string) error {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/onboarding/status/%s", apiURL, sessionID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		return fmt.Errorf("API error: %s", apiResp.Error)
	}

	var statusResp MessageResponse
	if err := json.Unmarshal(apiResp.Data, &statusResp); err != nil {
		return err
	}

	fmt.Println(statusResp.Message)
	return nil
}