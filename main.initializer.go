package main

func InitHttpServer() {
	if g.Cfg.HttpServerEnabled {
		startHttpEnabled()
	} else {
		clog(LevelInfo, "HTTP Server is disabled via config.")
	}
}
