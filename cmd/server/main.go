package main

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"barberpos-backend/internal/config"
	"barberpos-backend/internal/db"
	"barberpos-backend/internal/handler"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server"
	"barberpos-backend/internal/service"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := db.New(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect database", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	// Firebase Auth (optional)
	var firebaseAuth *auth.Client
	if cfg.FirebaseProjectID != "" {
		app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FirebaseProjectID}, firebaseOptions(cfg)...)
		if err != nil {
			logger.Error("failed to init firebase app", "err", err)
			os.Exit(1)
		}
		client, err := app.Auth(ctx)
		if err != nil {
			logger.Error("failed to init firebase auth", "err", err)
			os.Exit(1)
		}
		firebaseAuth = client
	}

	// repositories
	userRepo := repository.UserRepository{DB: pg}
	productRepo := repository.ProductRepository{DB: pg}
	categoryRepo := repository.CategoryRepository{DB: pg}
	customerRepo := repository.CustomerRepository{DB: pg}
	regionRepo := repository.RegionRepository{DB: pg}
	settingsRepo := repository.SettingsRepository{DB: pg}
	financeRepo := repository.FinanceRepository{DB: pg}
	membershipRepo := repository.MembershipRepository{DB: pg}
	stockRepo := repository.StockRepository{DB: pg}
	employeeRepo := repository.EmployeeRepository{DB: pg}
	fcmRepo := repository.FCMRepository{DB: pg}
	notificationRepo := repository.NotificationRepository{DB: pg}
	txRepo := repository.TransactionRepository{DB: pg}
	attendanceRepo := repository.AttendanceRepository{DB: pg}
	dashboardRepo := repository.DashboardRepository{DB: pg}
	closingRepo := repository.ClosingRepository{DB: pg}
	activityLogRepo := repository.ActivityLogRepository{DB: pg}

	// services
	authSvc := service.AuthService{
		Config:       cfg,
		Users:        userRepo,
		Employees:    employeeRepo,
		Logger:       logger,
		FirebaseAuth: firebaseAuth,
	}
	membershipSvc := service.MembershipService{Repo: membershipRepo}

	// handlers
	healthHandler := handler.HealthHandler{DB: pg}
	authHandler := handler.AuthHandler{Service: &authSvc}
	productHandler := handler.ProductHandler{Repo: productRepo, Currency: cfg.DefaultCurrency}
	productAdminHandler := handler.ProductAdminHandler{Repo: productRepo}
	categoryHandler := handler.CategoryHandler{Repo: categoryRepo}
	customerHandler := handler.CustomerHandler{Repo: customerRepo}
	regionHandler := handler.RegionHandler{Repo: regionRepo}
	settingsHandler := handler.SettingsHandler{Repo: settingsRepo}
	financeHandler := handler.FinanceHandler{Repo: financeRepo}
	membershipHandler := handler.MembershipHandler{Service: &membershipSvc}
	stockHandler := handler.StockHandler{Repo: stockRepo}
	employeeHandler := handler.EmployeeHandler{Repo: employeeRepo}
	fcmHandler := handler.FCMHandler{Repo: fcmRepo}
	notificationHandler := handler.NotificationHandler{Repo: notificationRepo}
	transactionHandler := handler.TransactionHandler{Repo: txRepo, Currency: cfg.DefaultCurrency, Membership: &membershipSvc, Employees: employeeRepo}
	attendanceHandler := handler.AttendanceHandler{Repo: attendanceRepo}
	dashboardHandler := handler.DashboardHandler{Repo: dashboardRepo}
	closingHandler := handler.ClosingHandler{Repo: closingRepo}
	activityLogHandler := handler.ActivityLogHandler{Repo: activityLogRepo}
	paymentHandler := handler.PaymentHandler{}
	homeHandler := handler.HomeHandler{}
	docsHandler := handler.DocsHandler{OpenAPIPath: "openapi.yaml"}

	router := server.NewRouter(cfg, logger, healthHandler, authHandler, productHandler, productAdminHandler, categoryHandler, customerHandler, regionHandler, settingsHandler, financeHandler, membershipHandler, transactionHandler, attendanceHandler, dashboardHandler, closingHandler, activityLogHandler, paymentHandler, fcmHandler, notificationHandler, stockHandler, employeeHandler, docsHandler, homeHandler)

	if err := server.Start(ctx, cfg, router, logger); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}

func firebaseOptions(cfg config.Config) []option.ClientOption {
	if cfg.FirebaseCredFile == "" {
		return nil
	}

	cred := cfg.FirebaseCredFile
	// Allow inline JSON or base64-encoded JSON in env to avoid writing a file.
	if strings.HasPrefix(strings.TrimSpace(cred), "{") {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(cred))}
	}
	if decoded, err := base64.StdEncoding.DecodeString(cred); err == nil && strings.HasPrefix(strings.TrimSpace(string(decoded)), "{") {
		return []option.ClientOption{option.WithCredentialsJSON(decoded)}
	}

	return []option.ClientOption{option.WithCredentialsFile(cred)}
}
