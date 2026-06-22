package routes

import (
	"log"
	"service-travego/config"
	"service-travego/configs"
	"service-travego/database"
	"service-travego/helper"
	"service-travego/internal/waai"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(app *fiber.App, cfg *configs.Config) {
	// Initialize database
	db, err := database.InitDatabase(cfg)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	// API routes
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return helper.SuccessResponse(c, fiber.StatusOK, "Service is running", fiber.Map{
			"status": "ok",
		})
	})

	// Initialize Redis
	_, err = helper.InitRedis(&cfg.Redis)
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	// Initialize Midtrans
	midtransCfg := config.InitMidtrans()
	rdb := helper.GetRedisClient()

	// Initialize services
	notificationSvc := service.NewNotificationService(db, cfg.Database.Driver)

	// Setup route groups
	SetupNotificationRoutes(app, db, cfg.Database.Driver) // Register public routes first
	SetupGeneralRoutes(api, db, cfg.Database.Driver)
	SetupAuthRoutes(api, db, cfg.Database.Driver, cfg)
	SetupBookingRoutes(api)
	SetupOrganizationRoutes(api, db, cfg.Database.Driver, cfg)
	SetupTeamRoutes(api, db, cfg.Database.Driver)
	SetupEmployeeRoutes(api, db, cfg.Database.Driver)
	SetupUserRoutes(api, db, cfg.Database.Driver)
	SetupUploadRoutes(api, db, cfg.Database.Driver)
	SetupFleetRoutes(api, db, cfg.Database.Driver)
	SetupFleetUnitRoutes(api, db, cfg.Database.Driver)
	SetupPartnerRoutes(api, db, cfg.Database.Driver)
	SetupScheduleRoutes(api, db, cfg.Database.Driver)
	SetupContentRoutes(api, db, cfg.Database.Driver)
	SetupServiceRoutes(api, db, cfg.Database.Driver)
	SetupCustomersRoutes(api, db, cfg.Database.Driver)
	SetupMessagesRoutes(api, db, cfg.Database.Driver)
	SetupOrderRoutes(api, db, cfg.Database.Driver, cfg)
	SetupDashboardRoutes(api, db, cfg.Database.Driver)
	SetupTransactionRoutes(api, db, cfg.Database.Driver, notificationSvc)
	SetupTourPackageRoutes(api, db, cfg.Database.Driver)
	SetupLeaveManagementRoutes(api, db, cfg.Database.Driver)
	SetupPrintManagementRoutes(api, db, cfg.Database.Driver)
	SetupPaymentRoutes(api, db, cfg.Database.Driver, midtransCfg)
	SetupPreferenceCityRoutes(api, db, cfg.Database.Driver)

	waaiCfg := waai.LoadConfig()
	var wagyClient *waai.WagyClient
	if waaiCfg.WagyDeviceID != "" && waaiCfg.WagyToken != "" {
		wagyClient = waai.NewWagyClient(waaiCfg.WagyDeviceID, waaiCfg.WagyToken)
	}

	SetupInventoryRoutes(api, db, cfg.Database.Driver, notificationSvc, wagyClient)
	SetupAssistantRoutes(api, db, cfg.Database.Driver, rdb)

	// Setup WhatsApp AI Assistant module (WAAI)
	if rdb == nil {
		log.Printf("Warning: Redis client is nil, WAAI may not work properly")
	} else {
		waaiCfg := waai.LoadConfig()
		if err := waai.RegisterRoutes(app, waaiCfg, db, cfg.Database.Driver, rdb); err != nil {
			log.Printf("Warning: Failed to register WAAI routes: %v", err)
		}
	}
}
