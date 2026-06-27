package routes

import (
	"database/sql"
	"service-travego/configs"
	"service-travego/handler"
	"service-travego/helper"
	"service-travego/internal/waai"
	"service-travego/internal/wagy"
	"service-travego/repository"
	"service-travego/service"

	"github.com/gofiber/fiber/v2"
)

// SetupOrganizationRoutes configures organization routes
func SetupOrganizationRoutes(api fiber.Router, db *sql.DB, driver string, cfg *configs.Config) {
	// Initialize repositories
	orgRepo := repository.NewOrganizationRepository(db, driver)
	orgUserRepo := repository.NewOrganizationUserRepository(db, driver)
	userRepo := repository.NewUserRepository(db, driver)
	orgTypeRepo := repository.NewOrganizationTypeRepository(db, driver)
	subscriptionRepo := repository.NewSubscriptionRepository(db, driver)

	// Initialize services
	orgService := service.NewOrganizationService(orgRepo, userRepo)
	orgService.SetOrganizationUserRepository(orgUserRepo)
	orgService.SetOrganizationTypeRepository(orgTypeRepo)
	orgService.SetSubscriptionRepository(subscriptionRepo)
	notificationSvc := service.NewNotificationService(db, driver)
	orgJoinService := service.NewOrganizationJoinService(orgRepo, orgUserRepo, userRepo, notificationSvc, &cfg.Email)
	orgTypeService := service.NewOrganizationTypeService(orgTypeRepo)
	garageService := service.NewGarageService(repository.NewGarageRepository(db, driver))
	garageHandler := handler.NewGarageHandler(garageService)

	// Initialize handlers
	authService := service.NewAuthService(userRepo, &cfg.Email)
	authService.SetOrganizationUserRepository(orgUserRepo)
	orgHandler := handler.NewOrganizationHandler(orgService)
	orgHandler.SetAuthService(authService)
	orgHandler.SetJoinService(orgJoinService)
	orgHandler.SetOrganizationTypeService(orgTypeService)

	// Set WagyClient if config is available
	waaiCfg := waai.LoadConfig()
	if waaiCfg.WagyDeviceID != "" && waaiCfg.WagyToken != "" {
		wagyClient := wagy.NewWagyClient(waaiCfg.WagyDeviceID, waaiCfg.WagyToken)
		orgHandler.SetWagyClient(wagyClient)
	}

	// Organization routes
	organization := api.Group("/organization")
	organization.Post("/create", helper.JWTAuthorizationMiddleware(), orgHandler.CreateOrganization)
	organization.Post("/join", helper.JWTAuthorizationMiddleware(), orgHandler.JoinOrganization)
	organization.Get("/api-config", helper.JWTAuthorizationMiddleware(), orgHandler.GetAPIConfig)
	organization.Post("/update/domain-url", helper.JWTAuthorizationMiddleware(), orgHandler.UpdateDomainURL)
	organization.Get("/bank-accounts", helper.JWTAuthorizationMiddleware(), orgHandler.GetBankAccounts)
	organization.Get("/detail", helper.JWTAuthorizationMiddleware(), orgHandler.GetOrganizationDetail)
	organization.Get("/employee/whatsapp/:employee_id", helper.JWTAuthorizationMiddleware(), orgHandler.EmployeeWhatsApp)
	organization.Post("/update", helper.JWTAuthorizationMiddleware(), orgHandler.UpdateOrganizationDetail)
	organization.Post("/update/logo", helper.JWTAuthorizationMiddleware(), orgHandler.UpdateOrganizationLogo)
	organization.Post("/bank-account/create", helper.JWTAuthorizationMiddleware(), orgHandler.CreateBankAccount)
	organization.Post("/bank-account/update", helper.JWTAuthorizationMiddleware(), orgHandler.UpdateBankAccount)
	organization.Post("/bank-account/delete", helper.JWTAuthorizationMiddleware(), orgHandler.DeleteBankAccount)
	organization.Get("/types", orgHandler.GetOrganizationTypes)
	organization.Get("/users", helper.JWTAuthorizationMiddleware(), orgHandler.GetUsers)
	organization.Put("/join/:action/:user_id", helper.JWTAuthorizationMiddleware(), orgHandler.HandleJoinAction)
	organization.Put("/users/:action/:user_id", helper.JWTAuthorizationMiddleware(), orgHandler.HandleUserAction)
	// Assistant routes
	organization.Get("/assistant/list", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantList)
	organization.Post("/assistant/submit", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantSubmit)
	organization.Post("/assistant/update", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantUpdate)
	organization.Post("/assistant/delete", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantDelete)
	organization.Get("/assistant/whatsapp-business", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantWhatsAppBusinessList)
	organization.Post("/assistant/whatsapp-business/update", helper.JWTAuthorizationMiddleware(), orgHandler.AssistantWhatsAppBusinessUpdate)
	// Garage routes
	organization.Get("/garage/list", helper.JWTAuthorizationMiddleware(), garageHandler.GetGarages)
	organization.Post("/garage/create", helper.JWTAuthorizationMiddleware(), garageHandler.CreateGarage)
	organization.Post("/garage/update", helper.JWTAuthorizationMiddleware(), garageHandler.UpdateGarage)
	organization.Post("/garage/delete", helper.JWTAuthorizationMiddleware(), garageHandler.DeleteGarage)
}
