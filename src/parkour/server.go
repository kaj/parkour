package main

import (
    "bufio"
    "fmt"
    "github.com/clbanning/mxj"
    "github.com/gocraft/web"
    mgo "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "html/template"
    "io"
    "io/ioutil"
    "math/rand"
    "net/http"
    "net/url"
    "time"
)

// Select base URL for server
const SERVERURL = "http://parkour.csc.kth.se"
//const SERVERURL = "http://localhost:3000"

// Select a login server!
// const LOGINSERVER = "MOCK" // offline development
// const LOGINSERVER = "https://login-r.referens.sys.kth.se/" // online dev
const LOGINSERVER = "https://login.kth.se/" // production

var (
	mgo_session *mgo.Session
	DB_name     string
)

// Collection entry for database
type Bout struct {
    Id bson.ObjectId `bson:"_id"`
    User string
    Other string
    Course string
    Lab string
    Logs []LogEntry
}

type LogEntry struct {
    Timestamp time.Time
    Entry string // Enum? "self", "other", "pause"
}

type User struct {
    userid string
}

type Context struct {
    session *Session
}

var courses = map[string]string{
    "aaaa":  "Obefintlig testkurs",
    "adk":   "Algoritmer, Datastrukturer, Komplexitet",
    "cprog": "Programkonstruktion i C++",
    "prgcl": "Programmeringsteknik för Civilingenjör & Lärare",
    "prgs":  "Programmeringsteknik för S",
}
var labs = map[string]string{
    "lab1": "Lab 1",
    "lab2": "Lab 2",
    "lab3": "Lab 3",
    "lab4": "Lab 4",
}

type Session struct {
    user User
    bout bson.ObjectId
}

func (c *Context) NewBout(rw web.ResponseWriter, req *web.Request) {
    course := req.FormValue("course")
    lab := req.FormValue("lab")
    with := req.FormValue("with")
    fmt.Println("Got form", course, lab, with)

    if course != "" && lab != "" && with != "" {
        mgo_conn := mgo_session.Copy()
        defer mgo_conn.Close()
        bout := new(Bout)
        bout.Id = bson.NewObjectId()
        bout.User = c.session.user.userid
        bout.Course = course
        bout.Lab = lab
        bout.Other = with

        err := mgo_conn.DB(DB_name).C("bouts").Insert(bout)
        if err != nil {
            panic(err);
        }
        fmt.Println("Inserted", bout.Id)
        c.session.bout = bout.Id

        http.Redirect(rw, req.Request, SERVERURL + "/bout", http.StatusFound)
        return
    }
    tpl := template.Must(template.ParseFiles("src/parkour/templates/newbout.html"))
    tpl.Execute(rw, map[string]interface{}{
        "user": c.session.user,
        "courses": courses,
        "labs": labs,
    })
}

func (c *Context) MainPage(rw web.ResponseWriter, req *web.Request) {
    tpl := template.Must(template.ParseFiles("src/parkour/templates/mainpage.html"))

    bout := getBout(c.session.bout)
    if bout == nil {
        http.Redirect(rw, req.Request, SERVERURL + "/", http.StatusFound)
        return
    }
    tpl.Execute(rw, map[string]interface{}{
        "kurs": courses[bout.Course],
        "lab": labs[bout.Lab],
        "me": string(bout.User), // TODO Get real name
        "other": string(bout.Other),
    })
}

func getBout(id bson.ObjectId) *Bout {
    fmt.Println("Try to work with bout", id)
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    if !id.Valid() {
        fmt.Println("Got bout nil")
        return nil
    }
    var result = new(Bout)
    err := mgo_conn.DB(DB_name).C("bouts").FindId(id).One(result)
    if err == nil {
        fmt.Println("Got bout", result)
        return result
    } else if err == mgo.ErrNotFound {
        fmt.Println("Not found, got nil")
        return nil
    } else {
        fmt.Println("Panic", err)
        panic(err)
    }
}

func (c *Context) ChangeDriver(rw web.ResponseWriter, req *web.Request) {
    body := bufio.NewReader(req.Body)
    name, err := body.ReadString(0)
    if (err == nil || err == io.EOF) && (name != "") {
        fmt.Println("Change driver for", c.session.user, "to", name)
        addLog(c.session.bout, name)
        rw.WriteHeader(200)
    } else {
        fmt.Println("Error: ", err, "after", name)
        rw.WriteHeader(400)
    }
}

func (c *Context) Pause(rw web.ResponseWriter, req *web.Request) {
    fmt.Println("Pause for", c.session.user)
    addLog(c.session.bout, "pause")
    rw.WriteHeader(200)
}

func addLog(bout bson.ObjectId, name string) {
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    log := new(LogEntry)
    log.Timestamp = time.Now()
    log.Entry = name
    fmt.Printf("Add to log for %v: %v\n", bout, name)
    err := mgo_conn.DB(DB_name).C("bouts").UpdateId(bout, bson.M{
        "$push": bson.M{
            "logs": log,
        },
    })
    if err != nil {
        panic(err)
    }
}

var sessiondb map[string]*Session // TODO Store in actual db!

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
    "abcdefghijklmnopqrstuvwxyz" +
    "0123456789")

func makeSession(session *Session, key string) string {
    if sessiondb == nil {
        sessiondb = make(map[string]*Session)
    }
    // Create a new session, store the user, return the session key
    if key == "" {
        keydata := make([]rune, 42)
        for i := range keydata {
            keydata[i] = letters[rand.Intn(len(letters))]
        }
        key = string(keydata)
    } else {
        if sessiondb[key] != nil {
            panic("Bad session reuse!")
        }
    }
    sessiondb[key] = session
    return key
}

func getSession(key string) *Session {
    return sessiondb[key]
}

func (c *Context) KthSessionMiddleware(rw web.ResponseWriter, r *web.Request,
    next web.NextMiddlewareFunc) {
    session, err := r.Cookie("PARSESS")

    if (session != nil) && (err == nil) {
        c.session = getSession(session.Value)
    }
    fmt.Println("Cookie", session, err, " -> session", c.session)
    if c.session == nil {
        ticket := r.URL.Query().Get("ticket")
        if LOGINSERVER == "MOCK" {
            c.session = new(Session)
            c.session.user.userid = "u1famwov"
            var oldkey string
            if session != nil { oldkey = session.Value } else { oldkey = "" }
            session = new (http.Cookie)
            session.Name = "PARSESS"
            session.Path = "/"
            // session.Domain = "localhost" -- hope it defaults sanely?
            session.Value = makeSession(c.session, oldkey)
            session.MaxAge = 3600
            fmt.Println("Setting cookie", session, "and redirect to hide ticket")
            http.SetCookie(rw, session)
            http.Redirect(rw, r.Request, SERVERURL + r.URL.Path, http.StatusFound)
            return; // early

        } else if ticket == "" {
            v := url.Values{}
            v.Set("service", SERVERURL + r.URL.Path);
            target := LOGINSERVER + "login?" + v.Encode()
            fmt.Println("Redirecting to", target, "for login");
            http.Redirect(rw, r.Request, target, http.StatusFound)
            return // Early!
        } else {
            // Ok, there seem to be login
            fmt.Println("Got ticket", ticket, "to validate")
            v := url.Values{}
            v.Set("ticket", ticket)
            v.Set("service", SERVERURL + r.URL.Path);
            validator := LOGINSERVER + "serviceValidate?" + v.Encode()
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

            
            c.session = new(Session)
            c.session.user.userid = userid
            var oldkey string
            if session != nil { oldkey = session.Value } else { oldkey = "" }
            session = new (http.Cookie)
            session.Name = "PARSESS"
            session.Path = "/"
            session.Domain = "localhost"
            session.Value = makeSession(c.session, oldkey)
            session.MaxAge = 3600
            fmt.Println("Setting cookie", session, "and redirect to hide ticket")
            http.SetCookie(rw, session)
            http.Redirect(rw, r.Request, SERVERURL + r.URL.Path, http.StatusFound)
            return; // early
        }
    }
    fmt.Println("Write session cookie back", session)
    http.SetCookie(rw, session)

    fmt.Println("User is", c.session.user)
    next(rw, r)
}

func initdb() (*mgo.Session, string) {
    session, err := mgo.Dial("mongodb://localhost/parkour")
    if err != nil {
        panic(err)
    }
    return session, "parkour"
}

func main() {
    rand.Seed(time.Now().UTC().UnixNano())
    mgo_session, DB_name = initdb()

    router := web.New(Context{}).
        Middleware(web.LoggerMiddleware).
        Middleware(web.ShowErrorsMiddleware)
    
    router.Subrouter(Context{}, "/static").
        Middleware(web.StaticMiddleware("src/parkour")).
        Get("/style.css", (*Context).MainPage).
        Get("/parkour.js", (*Context).MainPage)

    router.Subrouter(Context{}, "/").
        Middleware((*Context).KthSessionMiddleware).
        Get("/", (*Context).NewBout).
        Get("/bout", (*Context).MainPage).
        Put("/driver", (*Context).ChangeDriver).
        Put("/pause", (*Context).Pause)

    http.ListenAndServe("localhost:3000", router)
}
