package routes

import (
	"net/http"

	"go-entegra/internal/handlers"
	"go-entegra/internal/middleware"
	"go-entegra/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB, jwtExpireHours int) {
	webhookSvc := services.NewWebhookService(db)

	authHandler       := handlers.NewAuthHandler(db, jwtExpireHours)
	restaurantHandler := handlers.NewRestaurantHandler(db)
	providerHandler   := handlers.NewProviderHandler(db)
	rpHandler         := handlers.NewRestaurantProviderHandler(db)
	orderHandler      := handlers.NewOrderHandler(db, webhookSvc)
	getirHandler        := handlers.NewGetirHandler(db, webhookSvc)
	getirMenuHandler    := handlers.NewGetirMenuHandler(db)
	trendyolMenuHandler := handlers.NewTrendyolMenuHandler(db)
	migrosHandler        := handlers.NewMigrosHandler(db)

	// Panel (frontend HTML)
	panel := r.Group("/panel")
	{
		panel.GET("/login",     func(c *gin.Context) { c.File("web/login.html") })
		panel.GET("/register",  func(c *gin.Context) { c.File("web/register.html") })
		panel.GET("/dashboard", func(c *gin.Context) { c.File("web/dashboard.html") })
		panel.GET("/menu",      func(c *gin.Context) { c.File("web/menu.html") })
		panel.GET("/receipt",   func(c *gin.Context) { c.File("web/receipt.html") })
	}
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/panel/login") })

	// Public: Auth
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login",    authHandler.Login)
	}

	// Public: Getir webhook'ları (yeni sipariş + statü değişikliği)
	r.POST("/webhook/getir/:restaurant_slug",        getirHandler.IncomingOrder)
	r.POST("/webhook/getir/:restaurant_slug/status", getirHandler.IncomingStatusChange)
	// Public: Diğer pazaryeri webhook'ları
	r.POST("/webhook/:provider_slug/:restaurant_slug", orderHandler.IncomingOrder)

	// Protected routes
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		api.GET("/auth/me",    authHandler.Me)
		api.GET("/providers",  providerHandler.List)

		// Restaurants
		restaurants := api.Group("/restaurants")
		{
			restaurants.GET("",      restaurantHandler.List)
			restaurants.POST("",     restaurantHandler.Create)
			restaurants.GET("/:id",  restaurantHandler.Get)
			restaurants.PUT("/:id",  restaurantHandler.Update)
			restaurants.DELETE("/:id", restaurantHandler.Delete)

			// Entegrasyonlar
			restaurants.GET("/:id/providers",         rpHandler.List)
			restaurants.POST("/:id/providers",        rpHandler.Create)
			restaurants.GET("/:id/providers/:pid",    rpHandler.Get)
			restaurants.PUT("/:id/providers/:pid",    rpHandler.Update)
			restaurants.DELETE("/:id/providers/:pid", rpHandler.Delete)

			// Siparişler
			restaurants.GET("/:id/orders", orderHandler.ListByRestaurant)
		}

		// Orders
		orders := api.Group("/orders")
		{
			orders.GET("",               orderHandler.List)
			orders.GET("/:id",           orderHandler.Get)
			orders.PATCH("/:id/status",  orderHandler.UpdateStatus)
			orders.POST("/:id/getir/approve",           getirHandler.ApproveOrder)
			orders.POST("/:id/getir/approve-scheduled", getirHandler.ApproveScheduledOrder)
			orders.POST("/:id/getir/prepare",           getirHandler.PrepareOrder)
			orders.POST("/:id/getir/handover",          getirHandler.HandoverOrder)
			orders.POST("/:id/getir/deliver",           getirHandler.DeliverOrder)
			orders.POST("/:id/getir/cancel",            getirHandler.CancelOrder)
			orders.GET("/:id/getir/cancel-options",     getirHandler.GetCancelOptions)
			orders.GET("/:id/getir/inquiry",            getirHandler.InquireOrder)
		}

		// Getir — menü ve restoran yönetimi (integration_id = restaurant_provider.id)
		getir := api.Group("/getir/:integration_id")
		{
			getir.GET("/info",                             getirMenuHandler.GetRestaurantInfo)
			getir.PUT("/status",                           getirMenuHandler.SetStatus)
			getir.PUT("/pos-status",                       getirHandler.SetPosStatus)
			getir.PUT("/busyness",                         getirMenuHandler.SetBusyness)
			getir.PUT("/courier",                          getirMenuHandler.SetCourier)
			getir.GET("/working-hours",                    getirMenuHandler.GetWorkingHours)
			getir.PUT("/working-hours",                    getirMenuHandler.SetWorkingHours)
			getir.GET("/menu",                             getirMenuHandler.GetMenu)
			getir.GET("/categories",                       getirMenuHandler.GetCategories)
			getir.PUT("/products/:product_id/status",              getirMenuHandler.UpdateProductStatus)
			getir.PUT("/options/:option_id/status",                getirMenuHandler.UpdateOptionStatus)
			getir.GET("/chain-menus",                              getirMenuHandler.GetChainMenus)
			getir.GET("/chain-menus/:chain_menu_id",               getirMenuHandler.GetChainMenu)
			getir.GET("/chain-option-categories",                  getirMenuHandler.GetChainOptionCategories)
			getir.POST("/chain-menus/:chain_menu_id/update-prices", getirMenuHandler.UpdateChainMenuPrices)
		}

		// Trendyol — menü ve restoran yönetimi (integration_id = restaurant_provider.id)
		trendyol := api.Group("/trendyol/:integration_id")
		{
			trendyol.PUT("/status",                        trendyolMenuHandler.SetStatus)
			trendyol.GET("/menu",                          trendyolMenuHandler.GetMenu)
			trendyol.PUT("/products/:product_id/status",   trendyolMenuHandler.UpdateProductStatus)
			trendyol.PUT("/products/:product_id/price",    trendyolMenuHandler.UpdateProductPrice)
		}

		// Migros — menü ve mağaza yönetimi (integration_id = restaurant_provider.id)
		migros := api.Group("/migros/:integration_id")
		{
			migros.PUT("/status",                        migrosHandler.SetStatus)
			migros.GET("/menu",                          migrosHandler.GetMenu)
			migros.PUT("/products/:product_id/status",   migrosHandler.UpdateProductStatus)
			migros.PUT("/products/:product_id/price",    migrosHandler.UpdateProductPrice)
		}
	}
}
