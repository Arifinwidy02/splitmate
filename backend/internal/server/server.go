package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arifinwidy02/splitmate-backend/internal/auth"
	"github.com/Arifinwidy02/splitmate-backend/internal/balance"
	"github.com/Arifinwidy02/splitmate-backend/internal/dashboard"
	"github.com/Arifinwidy02/splitmate-backend/internal/expense"
	"github.com/Arifinwidy02/splitmate-backend/internal/group"
	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/internal/report"
	"github.com/Arifinwidy02/splitmate-backend/internal/session"
	"github.com/Arifinwidy02/splitmate-backend/internal/settlement"
	"github.com/Arifinwidy02/splitmate-backend/internal/user"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Session       *session.Service
	SecureCookies bool
	OAuth         *auth.OAuthConfig
	AppBaseURL    string
}

func New(deps Dependencies) http.Handler {
	userRepo := user.NewRepository(deps.Pool)
	authService := auth.NewService(userRepo)
	authHandler := auth.NewHandler(authService, deps.Session, deps.SecureCookies, deps.OAuth)

	groupRepo := group.NewRepository(deps.Pool)
	groupService := group.NewService(groupRepo, userRepo)
	groupHandler := group.NewHandler(groupService, deps.AppBaseURL)

	expenseRepo := expense.NewRepository(deps.Pool)
	expenseService := expense.NewService(expenseRepo, groupRepo)
	expenseHandler := expense.NewHandler(expenseService)

	balanceRepo := balance.NewRepository(deps.Pool)
	balanceService := balance.NewService(balanceRepo, groupRepo)
	balanceHandler := balance.NewHandler(balanceService)

	settlementRepo := settlement.NewRepository(deps.Pool)
	settlementService := settlement.NewService(settlementRepo, groupRepo)
	settlementHandler := settlement.NewHandler(settlementService)

	dashboardRepo := dashboard.NewRepository(deps.Pool)
	dashboardService := dashboard.NewService(dashboardRepo, groupRepo, balanceRepo)
	dashboardHandler := dashboard.NewHandler(dashboardService)

	reportRepo := report.NewRepository(deps.Pool)
	reportService := report.NewService(reportRepo)
	reportHandler := report.NewHandler(reportService)

	requireAuth := middleware.RequireAuth(deps.Session)
	optionalAuth := middleware.OptionalAuth(deps.Session)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("GET /api/v1/auth/google", authHandler.GoogleLogin)
	mux.HandleFunc("GET /api/v1/auth/google/callback", authHandler.GoogleCallback)
	mux.Handle("GET /api/v1/me", requireAuth(http.HandlerFunc(authHandler.Me)))

	mux.Handle("POST /api/v1/groups", requireAuth(http.HandlerFunc(groupHandler.Create)))
	mux.Handle("GET /api/v1/groups", requireAuth(http.HandlerFunc(groupHandler.List)))
	mux.Handle("GET /api/v1/groups/{groupId}", requireAuth(http.HandlerFunc(groupHandler.Get)))
	mux.Handle("GET /api/v1/groups/{groupId}/logo", requireAuth(http.HandlerFunc(groupHandler.GetLogo)))
	mux.Handle("PATCH /api/v1/groups/{groupId}", requireAuth(http.HandlerFunc(groupHandler.Update)))
	mux.Handle("DELETE /api/v1/groups/{groupId}", requireAuth(http.HandlerFunc(groupHandler.Delete)))
	mux.Handle("GET /api/v1/groups/{groupId}/members", requireAuth(http.HandlerFunc(groupHandler.ListMembers)))
	mux.Handle("DELETE /api/v1/groups/{groupId}/members/{userId}", requireAuth(http.HandlerFunc(groupHandler.RemoveMember)))
	mux.Handle("POST /api/v1/groups/{groupId}/invitations", requireAuth(http.HandlerFunc(groupHandler.CreateInvitation)))
	mux.Handle("POST /api/v1/groups/{groupId}/invitations/bulk", requireAuth(http.HandlerFunc(groupHandler.CreateBulkInvitations)))
	mux.Handle("POST /api/v1/groups/invitations/{token}/accept", requireAuth(http.HandlerFunc(groupHandler.AcceptInvitation)))
	mux.Handle("POST /api/v1/groups/{groupId}/invite-link", requireAuth(http.HandlerFunc(groupHandler.GetOrCreateInviteLink)))
	mux.Handle("DELETE /api/v1/groups/{groupId}/invite-link", requireAuth(http.HandlerFunc(groupHandler.RevokeInviteLink)))
	mux.Handle("GET /api/v1/invitations/{token}/preview", optionalAuth(http.HandlerFunc(groupHandler.PreviewInviteLink)))
	mux.Handle("POST /api/v1/invitations/{token}/join", requireAuth(http.HandlerFunc(groupHandler.JoinGroupViaLink)))

	mux.Handle("GET /api/v1/groups/{groupId}/expenses", requireAuth(http.HandlerFunc(expenseHandler.List)))
	mux.Handle("POST /api/v1/groups/{groupId}/expenses", requireAuth(http.HandlerFunc(expenseHandler.Create)))
	mux.Handle("GET /api/v1/expenses/{expenseId}", requireAuth(http.HandlerFunc(expenseHandler.Get)))
	mux.Handle("GET /api/v1/expenses/{expenseId}/receipt", requireAuth(http.HandlerFunc(expenseHandler.GetReceipt)))
	mux.Handle("PATCH /api/v1/expenses/{expenseId}", requireAuth(http.HandlerFunc(expenseHandler.Update)))
	mux.Handle("DELETE /api/v1/expenses/{expenseId}", requireAuth(http.HandlerFunc(expenseHandler.Delete)))

	mux.Handle("GET /api/v1/groups/{groupId}/balances", requireAuth(http.HandlerFunc(balanceHandler.GroupBalances)))
	mux.Handle("GET /api/v1/groups/{groupId}/settlement-suggestions", requireAuth(http.HandlerFunc(balanceHandler.SettlementSuggestions)))
	mux.Handle("GET /api/v1/me/balance", requireAuth(http.HandlerFunc(balanceHandler.PersonalBalance)))

	mux.Handle("GET /api/v1/groups/{groupId}/export", requireAuth(http.HandlerFunc(reportHandler.Export)))

	mux.Handle("GET /api/v1/groups/{groupId}/settlements", requireAuth(http.HandlerFunc(settlementHandler.List)))
	mux.Handle("POST /api/v1/groups/{groupId}/settlements", requireAuth(http.HandlerFunc(settlementHandler.Create)))

	mux.Handle("GET /api/v1/dashboard", requireAuth(http.HandlerFunc(dashboardHandler.Get)))

	return mux
}
