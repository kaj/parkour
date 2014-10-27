package main

import (
    "bufio"
    "fmt"
    "github.com/clbanning/mxj"
    "github.com/gocraft/web"
    "html/template"
    "io"
    "io/ioutil"
    "math/rand"
    "net/http"
    "net/url"
    "time"
)

type User struct {
    userid string
}

type Context struct {
    user *User
}

func (c *Context) MainPage(rw web.ResponseWriter, req *web.Request) {
    tpl := template.Must(template.ParseFiles("src/parkour/templates/mainpage.html"))
    tpl.Execute(rw, map[string]interface{}{
        "kurs": "FG4711 Irrlära 8 hp",
        "lab": "Lab 1 - go write some code",
        "me": string(c.user.userid), // TODO Get real name
        "other": "Marcus",
    })
}

func (c *Context) ChangeDriver(rw web.ResponseWriter, req *web.Request) {
    body := bufio.NewReader(req.Body)
    name, err := body.ReadString(0)
    if (err == nil || err == io.EOF) && (name != "") {
        // TODO Write to actual log in database!
        fmt.Println("Change driver for", c.user, "to", name)
        rw.WriteHeader(200)
    } else {
        fmt.Println("Error: ", err, "after", name)
        rw.WriteHeader(400)
    }
}

func (c *Context) Pause(rw web.ResponseWriter, req *web.Request) {
    // TODO Write to actual log in database!
    fmt.Println("Pause for", c.user)
    rw.WriteHeader(200)
}

var sessiondb map[string]*User // TODO Store in actual db!

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
    "abcdefghijklmnopqrstuvwxyz" +
    "0123456789")

func makeSession(user *User) string {
    if sessiondb == nil {
        sessiondb = make(map[string]*User)
    }
    // Create a new session, store the user, return the session key
    keydata := make([]rune, 42)
    for i := range keydata {
        keydata[i] = letters[rand.Intn(len(letters))]
    }
    key := string(keydata)
    sessiondb[key] = user
    return key
}

func getSession(key string) *User {
    return sessiondb[key]
}

func (c *Context) KthSessionMiddleware(rw web.ResponseWriter, r *web.Request,
    next web.NextMiddlewareFunc) {
    session, err := r.Cookie("PARSESS")

    if (session != nil) && (err == nil) {
        c.user = getSession(session.Value)
    }
    fmt.Println("Cookie", session, err, " -> user", c.user)
    if c.user == nil {
        ticket := r.URL.Query().Get("ticket")
        if ticket == "" {
            v := url.Values{}
            v.Set("service", "http://localhost:3000" + r.URL.Path);
            target := "http://login-r.referens.sys.kth.se/login?" + v.Encode()
            fmt.Println("Redirecting to", target, "for login");
            http.Redirect(rw, r.Request, target, http.StatusFound)
            return // Early!
        } else {
            // Ok, there seem to be login
            fmt.Println("Got ticket", ticket, "to validate")
            v := url.Values{}
            v.Set("ticket", ticket)
            v.Set("service", "http://localhost:3000" + r.URL.Path);
            validator := "http://login-r.referens.sys.kth.se/serviceValidate?" + v.Encode()
            client := new(http.Client)
            res, err := client.Get(validator)
            if err != nil {
                rw.WriteHeader(500)
                return
            }
            data, err := ioutil.ReadAll(res.Body)
            if err != nil {
                rw.WriteHeader(500)
                return
            }
            doc, err := mxj.NewMapXml(data, false)
            if err != nil {
                rw.WriteHeader(500)
                return
            }
            serv := doc["serviceResponse"].(map[string]interface{})
            result := serv["authenticationSuccess"].(map[string]interface{})
            userid := result["user"].(string)

            c.user = new(User)
            c.user.userid = userid
            session = new (http.Cookie)
            session.Name = "PARSESS"
            session.Path = "/"
            session.Domain = "localhost"
            session.Value = makeSession(c.user)
            fmt.Println("Setting cookie", session, "and redirect to hide ticket")
            http.SetCookie(rw, session)
            http.Redirect(rw, r.Request, "http://localhost:3000" + r.URL.Path, http.StatusFound)
            return; // early
        }
    }
    fmt.Println("Write session cookie back", session)
    http.SetCookie(rw, session)

    fmt.Println("User is", c.user)
    next(rw, r)
}


func main() {
    rand.Seed(time.Now().UTC().UnixNano())
    router := web.New(Context{}).
        Middleware(web.LoggerMiddleware).
        Middleware(web.ShowErrorsMiddleware)
    
    router.Subrouter(Context{}, "/static").
        Middleware(web.StaticMiddleware("src/parkour")).
        Get("/style.css", (*Context).MainPage).
        Get("/parkour.js", (*Context).MainPage)

    router.Subrouter(Context{}, "/").
        Middleware((*Context).KthSessionMiddleware).
        Get("/", (*Context).MainPage).
        Put("/driver", (*Context).ChangeDriver).
        Put("/pause", (*Context).Pause)

    http.ListenAndServe("localhost:3000", router)
}
