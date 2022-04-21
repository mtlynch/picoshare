package handlers

import "net/http"

func (s *Server) routes() {
	s.router.HandleFunc("/api/auth", s.authPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/auth", s.authDelete()).Methods(http.MethodDelete)

	authenticatedApis := s.router.PathPrefix("/api").Subrouter()
	authenticatedApis.Use(s.requireAuthentication)
	authenticatedApis.HandleFunc("/entry", s.entryPost()).Methods(http.MethodPost)
	authenticatedApis.HandleFunc("/entry/{id}", s.entryDelete()).Methods(http.MethodDelete)
	authenticatedApis.HandleFunc("/guest-links", s.guestLinksPost()).Methods(http.MethodPost)
	authenticatedApis.HandleFunc("/guest-links/{id}", s.guestLinksDelete()).Methods(http.MethodDelete)

	publicApis := s.router.PathPrefix("/api").Subrouter()
	publicApis.HandleFunc("/guest/{guestLinkID}", s.guestEntryPost()).Methods(http.MethodPost)

	static := s.router.PathPrefix("/").Subrouter()
	static.PathPrefix("/css/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/js/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/third-party/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)

	// Add all the root-level static resources.
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
		static.Path(f).HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	}

	authenticatedViews := s.router.PathPrefix("/").Subrouter()
	authenticatedViews.Use(s.requireAuthentication)
	authenticatedViews.HandleFunc("/files", s.fileIndexGet()).Methods(http.MethodGet)
	authenticatedViews.HandleFunc("/guest-links", s.guestLinkIndexGet()).Methods(http.MethodGet)
	authenticatedViews.HandleFunc("/guest-links/new", s.guestLinksNewGet()).Methods(http.MethodGet)

	views := s.router.PathPrefix("/").Subrouter()
	views.Use(upgradeToHttps)
	views.HandleFunc("/login", s.authGet()).Methods(http.MethodGet)
	views.PathPrefix("/!{id}").HandlerFunc(s.entryGet()).Methods(http.MethodGet)
	views.PathPrefix("/!{id}/{filename}").HandlerFunc(s.entryGet()).Methods(http.MethodGet)
	views.PathPrefix("/g/{guestLinkID}").HandlerFunc(s.guestUploadGet()).Methods(http.MethodGet)
	views.HandleFunc("/", s.indexGet()).Methods(http.MethodGet)
}
