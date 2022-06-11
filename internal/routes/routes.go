package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mikarios/golib/routerwrapper"

	"github.com/mikarios/imageresizer/internal/routes/imageroute"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	unprotected := router.PathPrefix("/api/v1").Subrouter()

	routerwrapper.New(unprotected, nil).
		HandleFunc("/job", imageroute.AddImageScaleJob).
		Methods(http.MethodPost).
		Create()

	return router
}
