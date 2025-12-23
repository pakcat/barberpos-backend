package server

import (
	"net/http"
	"time"

	"barberpos-backend/internal/config"
	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// NewRouter wires HTTP routes and middleware.
func NewRouter(cfg config.Config,
	health handler.HealthHandler,
	auth handler.AuthHandler,
	products handler.ProductHandler,
	productsAdmin handler.ProductAdminHandler,
	categories handler.CategoryHandler,
	customers handler.CustomerHandler,
	settings handler.SettingsHandler,
	finance handler.FinanceHandler,
	membership handler.MembershipHandler,
	tx handler.TransactionHandler,
	attendance handler.AttendanceHandler,
	dashboard handler.DashboardHandler,
	closing handler.ClosingHandler,
	payments handler.PaymentHandler,
	fcm handler.FCMHandler,
	home handler.HomeHandler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httprate.LimitByIP(200, 1*time.Minute))

	health.RegisterRoutes(r)
	auth.RegisterRoutes(r)
	home.RegisterRoutes(r)

	r.Group(func(pr chi.Router) {
		pr.Use(AuthMiddleware(cfg.JWTSecret))
		// staff-level (staff/manager/admin)
		pr.Group(func(sr chi.Router) {
			sr.Use(RequireRole(domain.RoleAdmin, domain.RoleManager, domain.RoleStaff))
			products.RegisterRoutes(sr)
			categories.RegisterRoutes(sr)
			customers.RegisterRoutes(sr)
			tx.RegisterRoutes(sr)
			attendance.RegisterRoutes(sr)
			payments.RegisterRoutes(sr)
			closing.RegisterRoutes(sr)
			fcm.RegisterRoutes(sr)
		})
		// manager-level (manager/admin)
		pr.Group(func(mr chi.Router) {
			mr.Use(RequireRole(domain.RoleAdmin, domain.RoleManager))
			dashboard.RegisterRoutes(mr)
			productsAdmin.RegisterRoutes(mr)
			settings.RegisterRoutes(mr)
			finance.RegisterRoutes(mr)
			membership.RegisterRoutes(mr)
		})
	})

	return r
}
