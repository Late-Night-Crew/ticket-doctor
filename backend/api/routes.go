package api

func (a *API) routes() {
	api := a.app.Group("/api")

	api.Get("/health", a.health)

	// B3 (see Docs/BACKEND_PLAN.md):
	//   GET  /leagues, /leagues/:id/fixtures   api/leagues
	//   POST /tickets/analyse, /suggestions/:id/accept   api/tickets
	//   POST /wizard                           api/wizard
	//   GET  /results                          api/results
}
