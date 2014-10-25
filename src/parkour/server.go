package main

import (
    "github.com/gocraft/web"
    "html/template"
    "net/http"
)

type Context struct {
    HelloCount int
}

func (c *Context) MainPage(rw web.ResponseWriter, req *web.Request) {
    tpl := template.Must(template.ParseFiles("src/parkour/templates/mainpage.html"))
    tpl.Execute(rw, map[string]interface{}{
        "kurs": "FG4711 Irrlära 8 hp",
        "lab": "Lab 1 - go write some code",
        "me": "Rasmus",
        "other": "Marcus",
    })
}

func main() {
    router := web.New(Context{}).
        Middleware(web.LoggerMiddleware).
        Middleware(web.ShowErrorsMiddleware).
        Get("/", (*Context).MainPage)

    router.Subrouter(Context{}, "/static").
        Middleware(web.StaticMiddleware("src/parkour")).
        Get("/style.css", (*Context).MainPage).
        Get("/parkour.js", (*Context).MainPage)

    http.ListenAndServe("localhost:3000", router)
}
