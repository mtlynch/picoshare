package handlers

import "net/http"

func (s *Server) routes() {
	// Public auth API routes
	s.router.HandleFunc("POST /api/auth", s.authPost())
	s.router.HandleFunc("DELETE /api/auth", s.authDelete())

	// Authenticated API routes
	s.router.Handle("POST /api/entry", s.authMiddleware(s.entryPost()))
	s.router.Handle("PUT /api/entry/{id}", s.authMiddleware(s.entryPut()))
	s.router.Handle("DELETE /api/entry/{id}", s.authMiddleware(s.entryDelete()))
	s.router.Handle("POST /api/guest-links", s.authMiddleware(s.guestLinksPost()))
	s.router.Handle("DELETE /api/guest-links/{id}", s.authMiddleware(s.guestLinksDelete()))
	s.router.Handle("PUT /api/guest-links/{id}/enable", s.authMiddleware(s.guestLinksEnableDisable()))
	s.router.Handle("PUT /api/guest-links/{id}/disable", s.authMiddleware(s.guestLinksEnableDisable()))
	s.router.Handle("PUT /api/settings", s.authMiddleware(s.settingsPut()))

	// Public API routes
	s.router.Handle("POST /api/guest/{guestLinkID}", s.checkAuthentication(s.guestEntryPost()))

	// Static file routes
	s.router.Handle("GET /css/", serveStaticResource())
	s.router.Handle("GET /js/", serveStaticResource())
	s.router.Handle("GET /third-party/", serveStaticResource())

	// Root-level static resources
	for _, f := range []string{
		"/android-chrome-192x192.png",
		"/android-chrome-384x384.png",
		"/apple-touch-icon.png",
		"/browserconfig.xml",
		"/favicon-16x16.png",
		"/favicon-32x32.png",
		"/favicon.ico",
		"/mstile-150x150.png",
		"/safari-pinned-tab.svg",
		"/site.webmanifest",
	} {
		s.router.Handle("GET "+f, serveStaticResource())
	}

	// Authenticated view routes
	s.router.Handle("GET /information", s.authViewMiddleware(s.systemInformationGet()))
	s.router.Handle("GET /files", s.authViewMiddleware(s.fileIndexGet()))
	s.router.Handle("GET /files/{id}/downloads", s.authViewMiddleware(s.fileDownloadsGet()))
	s.router.Handle("GET /files/{id}/edit", s.authViewMiddleware(s.fileEditGet()))
	s.router.Handle("GET /files/{id}/info", s.authViewMiddleware(s.fileInfoGet()))
	s.router.Handle("GET /files/{id}/confirm-delete", s.authViewMiddleware(s.fileConfirmDeleteGet()))
	s.router.Handle("GET /guest-links", s.authViewMiddleware(s.guestLinkIndexGet()))
	s.router.Handle("GET /guest-links/new", s.authViewMiddleware(s.guestLinksNewGet()))
	s.router.Handle("GET /settings", s.authViewMiddleware(s.settingsGet()))

	// Public view routes (with upgradeToHttps and CSP)
	viewMiddleware := func(h http.Handler) http.Handler {
		return s.checkAuthentication(enforceContentSecurityPolicy(upgradeToHttps(h)))
	}
	s.router.Handle("GET /login", viewMiddleware(s.authGet()))
	s.router.Handle("GET /g/{guestLinkID}", viewMiddleware(s.guestUploadGet()))
	s.router.Handle("GET /{$}", viewMiddleware(s.indexGet()))

	// Entry routes: /-{id} and /!{id} patterns can't use ServeMux wildcards
	// directly since wildcards must be entire path segments. Handle with catch-all.
	s.router.Handle("GET /{path...}", viewMiddleware(s.entryPathHandler()))

	s.addDevRoutes()
}

// authMiddleware applies checkAuthentication and requireAuthentication to a handler.
func (s *Server) authMiddleware(h http.Handler) http.Handler {
	return s.checkAuthentication(s.requireAuthentication(h))
}

// authViewMiddleware applies checkAuthentication, requireAuthentication, and CSP to a view handler.
func (s *Server) authViewMiddleware(h http.Handler) http.Handler {
	return s.checkAuthentication(s.requireAuthentication(enforceContentSecurityPolicy(h)))
}
