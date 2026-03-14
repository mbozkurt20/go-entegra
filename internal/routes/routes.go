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
	trendyolHandler    := handlers.NewTrendyolHandler(db, webhookSvc)
	migrosHandler        := handlers.NewMigrosHandler(db, webhookSvc)
	ysHandler            := handlers.NewYemeksepetiHandler(db, webhookSvc)

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

	// Public: Getir webhook'ları
	r.POST("/webhook/getir/:restaurant_slug",        getirHandler.IncomingOrder)
	r.POST("/webhook/getir/:restaurant_slug/status", getirHandler.IncomingStatusChange)
	// Public: YemekSepeti (DH) webhook — posVendorId ile
	r.POST("/webhook/yemeksepeti/:pos_vendor_id", ysHandler.IncomingOrder)
	// Public: Trendyol webhook
	r.POST("/webhook/trendyol/:restaurant_slug", trendyolHandler.IncomingOrder)
	// Public: Migros webhook
	r.POST("/webhook/migros/:restaurant_slug", migrosHandler.IncomingOrder)
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
			restaurants.POST("/:id/credits", restaurantHandler.AddCredits)
			restaurants.GET("/:id/credit-transactions", restaurantHandler.ListCreditTransactions)

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
			orders.POST("/:id/migros/approve",            migrosHandler.ApproveOrder)
			orders.POST("/:id/migros/prepare",            migrosHandler.PrepareOrder)
			orders.POST("/:id/migros/deliver",            migrosHandler.DeliverOrder)
			orders.POST("/:id/migros/cancel",             migrosHandler.CancelOrder)
			orders.POST("/:id/trendyol/approve",          trendyolHandler.ApproveOrder)
			orders.POST("/:id/trendyol/prepare",          trendyolHandler.PrepareOrder)
			orders.POST("/:id/trendyol/deliver",          trendyolHandler.DeliverOrder)
			orders.POST("/:id/trendyol/cancel",           trendyolHandler.CancelOrder)
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
			getir.PUT("/products/:product_id/price",               getirMenuHandler.UpdateProductPrice)
			getir.PUT("/options/:option_id/status",                getirMenuHandler.UpdateOptionStatus)
			getir.GET("/chain-menus",                              getirMenuHandler.GetChainMenus)
			getir.GET("/chain-menus/:chain_menu_id",               getirMenuHandler.GetChainMenu)
			getir.GET("/chain-option-categories",                  getirMenuHandler.GetChainOptionCategories)
			getir.POST("/chain-menus/:chain_menu_id/update-prices", getirMenuHandler.UpdateChainMenuPrices)
			getir.GET("/option-products",    getirMenuHandler.GetOptionProducts)
			getir.GET("/all-payment-methods", getirMenuHandler.GetAllPaymentMethods)
			getir.GET("/payment-methods",    getirMenuHandler.GetPaymentMethods)
			getir.POST("/payment-methods",   getirMenuHandler.AddPaymentMethod)
			getir.DELETE("/payment-methods", getirMenuHandler.DeletePaymentMethod)
		}

		// Trendyol — menü ve restoran yönetimi (integration_id = restaurant_provider.id)
		trendyol := api.Group("/trendyol/:integration_id")
		{
			trendyol.GET("/info",                             trendyolMenuHandler.GetInfo)
			trendyol.PUT("/status",                           trendyolMenuHandler.SetStatus)
			trendyol.PUT("/working-hours",                    trendyolMenuHandler.UpdateWorkingHours)
			trendyol.PUT("/delivery-time",                    trendyolMenuHandler.UpdateDeliveryTime)
			trendyol.PUT("/delivery-zones",                   trendyolMenuHandler.UpdateDeliveryZones)
			trendyol.GET("/menu",                             trendyolMenuHandler.GetMenu)
			trendyol.PUT("/products/:product_id/status",      trendyolMenuHandler.UpdateProductStatus)
			// trendyol.PUT("/products/:product_id/price", ...) // devre dışı — 404 hatası veriyor
			trendyol.PUT("/sections/:section_id/status",      trendyolMenuHandler.UpdateSectionStatus)
			trendyol.GET("/batch/:batch_id",                  trendyolMenuHandler.GetBatchStatus)
		}

		// Migros — menü ve mağaza yönetimi (integration_id = restaurant_provider.id)
		migros := api.Group("/migros/:integration_id")
		{
			migros.GET("/info",                          migrosHandler.GetInfo)
			migros.GET("/view-status",                   migrosHandler.GetStoreViewStatus)
			migros.PUT("/status",                        migrosHandler.SetStatus)
			migros.POST("/off-date",                     migrosHandler.AddStoreOffDate)
			migros.DELETE("/off-date",                   migrosHandler.RemoveStoreOffDate)
			migros.GET("/working-hours",                 migrosHandler.GetWorkingHours)
			migros.PUT("/working-hours",                 migrosHandler.UpdateWorkingHours)
			migros.GET("/payment-methods",               migrosHandler.GetPaymentMethods)
			migros.PUT("/payment-methods",               migrosHandler.UpdatePaymentMethods)
			migros.GET("/cancel-reasons",                migrosHandler.GetCancelReasons)
			migros.GET("/menu",                          migrosHandler.GetMenu)
			migros.PUT("/products/:product_id/status",   migrosHandler.UpdateProductStatus)
			migros.PUT("/products/:product_id/price",    migrosHandler.UpdateProductPrice)
			migros.PUT("/options/:option_id/status",     migrosHandler.UpdateOptionStatus)
		}

		// YemekSepeti (DH) — restoran yönetimi (integration_id = restaurant_provider.id)
		ys := api.Group("/yemeksepeti/:integration_id")
		{
			ys.GET("/status",  ysHandler.GetStatus)
			ys.PUT("/status",  ysHandler.SetStatus)
			ys.POST("/orders/:order_token/accept",   ysHandler.AcceptOrder)
			ys.POST("/orders/:order_token/reject",   ysHandler.RejectOrder)
			ys.POST("/orders/:order_token/pickup",   ysHandler.PickupOrder)
			ys.POST("/orders/:order_token/prepared", ysHandler.PreparationCompleted)
		}
	}
}
