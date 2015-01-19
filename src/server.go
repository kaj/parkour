package main

import (
	"github.com/gocraft/web"
	"math/rand"
	"net/http"
	"parkour"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	parkour.InitDb()

	router := web.New(parkour.Context{}).
		Middleware(web.LoggerMiddleware).
		Middleware(web.ShowErrorsMiddleware).
		Get("/suggestuser", parkour.SuggestUser)

	router.Subrouter(parkour.Context{}, "/static").
		Middleware(web.StaticMiddleware("src/parkour")).
		Get("/style.css", (*parkour.Context).MainPage).
		Get("/parkour.js", (*parkour.Context).MainPage).
		Get("/jquery.jqplot.min.js", (*parkour.Context).MainPage).
		Get("/jqplot.pieRenderer.min.js", (*parkour.Context).MainPage).
		Get("/jquery.jqplot.min.css", (*parkour.Context).MainPage)

	router.Subrouter(parkour.Context{}, "/").
		Middleware((*parkour.Context).KthSessionMiddleware).
		Get("/", (*parkour.Context).NewBout).
		Get("/bout", (*parkour.Context).MainPage).
		Get("/history", (*parkour.Context).History).
		Get("/boutlog", (*parkour.Context).CurrentLog).
		Get("/logout", (*parkour.Context).Logout).
		Put("/driver", (*parkour.Context).ChangeDriver).
		Put("/pause", (*parkour.Context).Pause)

	http.ListenAndServe("localhost:3000", router)
}
