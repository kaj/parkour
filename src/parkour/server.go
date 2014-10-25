package main

import (
    "github.com/gocraft/web"
    "html/template"
    "net/http"
    "fmt"
    "bufio"
    "io"
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

func (c *Context) ChangeDriver(rw web.ResponseWriter, req *web.Request) {
    body := bufio.NewReader(req.Body)
    name, err := body.ReadString(0)
    if (err == nil || err == io.EOF) && (name != "") {
        fmt.Println("Change driver to", name) // Should actually write db log
        rw.WriteHeader(200)
    } else {
        fmt.Println("Error: ", err, "after", name)
        rw.WriteHeader(400)
    }
}

func (c *Context) Pause(rw web.ResponseWriter, req *web.Request) {
    fmt.Println("Pause") // Should actually write db log
    rw.WriteHeader(200)
}

func main() {
    router := web.New(Context{}).
        Middleware(web.LoggerMiddleware).
        Middleware(web.ShowErrorsMiddleware).
        Get("/", (*Context).MainPage).
        Put("/driver", (*Context).ChangeDriver).
        Put("/pause", (*Context).Pause)
    
    router.Subrouter(Context{}, "/static").
        Middleware(web.StaticMiddleware("src/parkour")).
        Get("/style.css", (*Context).MainPage).
        Get("/parkour.js", (*Context).MainPage)

    http.ListenAndServe("localhost:3000", router)
}
