package api

type preloadRoutesFunc func(IRoutes)

var preloadRoutes []preloadRoutesFunc

func PreloadRoutes(h preloadRoutesFunc) {
	preloadRoutes = append(preloadRoutes, h)
}

func init() {
	PreloadRoutes(registerHandlers)
}

func registerHandlers(r IRoutes) {
	r.Any("/pay/test/add_order", addTestOrder)
	r.Any("/pay/test/notify", addTestOrder)
	r.POST("/pay/apple/notify_sub", notifyAppleSubscription)
	r.POST("/plate2/upload_package", uploadConfigTables)

	handleCmd("/plate/check_version_new", checkClientVersionAndConfig, (*clientVersionArgs)(nil))
	handleCmd("/pay/google/add_order", addGoogleOrder, (*googlePayArgs)(nil))
	handleCmd("/pay/apple/add_order", addAppleOrder, (*applePayArgs)(nil))
	handleCmd("/plate/save_icon", savePlateIcon, (*plateArgs)(nil))
}
