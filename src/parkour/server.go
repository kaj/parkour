package main

import (
    "bufio"
    "encoding/json"
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
    "os/exec"
    "regexp"
    "strings"
    "time"
)

// Select base URL for server
const SERVERHOST = "parkour.csc.kth.se"
const SERVERURL = "http://" + SERVERHOST
//const SERVERHOST = "localhost"
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
    Duration int
}

type User struct {
    Kthid string
    Firstname string
    Name string
}

type Context struct {
    session *Session
}

var courses = map[string]string{
    "aaaa":    "Obefintlig testkurs",
    "adk":     "Algoritmer, Datastrukturer, Komplexitet",
    "cprog":   "Programkonstruktion i C++",
    "prgcl":   "Programmeringsteknik för Civilingenjör & Lärare",
    "prgomed": "Programmeringsteknik för Media"
    "prgs":    "Programmeringsteknik för S",
}
var labs = map[string]string{
    "lab1": "Lab 1",
    "lab2": "Lab 2",
    "lab3": "Lab 3",
    "lab4": "Lab 4",
}

type Session struct {
    Key string
    User User
    Bout *bson.ObjectId
}

func findKthid(user string) string {
    i := strings.Index(user, "(")
    j := strings.Index(user, ")")
    if i >= 0 && j > 0 {
        return user[i+1:j]
    } else if len(user) == 8 {
        return user
    } else {
        return ""
    }
}

func (c *Context) NewBout(rw web.ResponseWriter, req *web.Request) {
    course := req.FormValue("course")
    lab := req.FormValue("lab")
    with := req.FormValue("with")
    withkthid := findKthid(with)
    // fmt.Println("Got form", course, lab, with)

    if course != "" && lab != "" && withkthid != "" {
        withuser := getUser(withkthid)

        mgo_conn := mgo_session.Copy()
        defer mgo_conn.Close()
        bout := new(Bout)
        bout.Id = bson.NewObjectId()
        bout.User = c.session.User.Kthid
        bout.Course = course
        bout.Lab = lab
        bout.Other = withuser.Kthid

        err := mgo_conn.DB(DB_name).C("bouts").Insert(bout)
        if err != nil {
            panic(err);
        }
        c.session.Bout = &bout.Id

        http.Redirect(rw, req.Request, SERVERURL + "/bout", http.StatusFound)
        return
    }
    tpl := template.Must(template.ParseFiles("src/parkour/templates/newbout.html"))
    tpl.Execute(rw, map[string]interface{}{
        "User": c.session.User,
        "courses": courses,
        "labs": labs,
    })
}

func (c *Context) MainPage(rw web.ResponseWriter, req *web.Request) {
    tpl := template.Must(template.ParseFiles("src/parkour/templates/mainpage.html"))

    bout := getBout(c.session.Bout)
    if bout == nil {
        http.Redirect(rw, req.Request, SERVERURL + "/", http.StatusFound)
        return
    }
    tpl.Execute(rw, map[string]interface{}{
        "User": c.session.User,
        "kurs": courses[bout.Course],
        "lab": labs[bout.Lab],
        "me": string(c.session.User.Firstname),
        "other": string(getUser(bout.Other).Firstname),
    })
}

func getBout(id *bson.ObjectId) *Bout {
    // fmt.Println("Try to work with bout", id)
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    if id == nil || !id.Valid() {
        return nil
    }
    var result = new(Bout)
    err := mgo_conn.DB(DB_name).C("bouts").FindId(id).One(result)
    if err == nil {
        return result
    } else if err == mgo.ErrNotFound {
        return nil
    } else {
        panic(err)
    }
}

func getUser(kthid string) User {
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    var result User
    err := mgo_conn.DB(DB_name).C("users").Find(bson.M{"kthid": kthid}).One(&result)
    if err == nil {
        return result
    } else {
        // No such user!  Check with LDAP!
		out, err := exec.Command("ldapsearch", "-x", "-LLL", "ugKthid="+kthid).Output()
		if err != nil {
			panic(err)
		}
		reg := regexp.MustCompile("(?m)^([^:]+): ([^\n]+)$")
		matches := reg.FindAllStringSubmatch(string(out), -1)
		matchmap := make(map[string]string)
		for _, match := range matches {
			key := match[1]
			value := match[2]
			matchmap[key] = value
		}
		// username := matchmap["uid"]
        result.Kthid = kthid
		result.Name = matchmap["cn"]
		result.Firstname = matchmap["givenName"]

        err = mgo_conn.DB(DB_name).C("users").Insert(result)
        if err != nil {
            panic(err)
        }
        return result
    }
}

func SuggestUser(rw web.ResponseWriter, req *web.Request) {
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    var results []bson.M
    q := strings.Replace(req.FormValue("term"), " ", ".*", -1)
    mgo_conn.DB(DB_name).C("users").
        Find(bson.M{"name": bson.M{"$regex": q, "$options": "i"}}).
        Limit(10).All(&results)

    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(200)
    io.WriteString(rw, "[")
    for i, obj := range results {
        if i > 0 {
            io.WriteString(rw, ", ")
        }
        io.WriteString(rw, "\"" + obj["name"].(string) +
            " (" + obj["kthid"].(string) + ")\"")
    }
    io.WriteString(rw, "]\n")
}


func (c *Context) ChangeDriver(rw web.ResponseWriter, req *web.Request) {
    body := bufio.NewReader(req.Body)
    name, err := body.ReadString(0)
    if (err == nil || err == io.EOF) && (name != "") {
        addLog(*c.session.Bout, name)
        rw.WriteHeader(200)
    } else {
        fmt.Println("Error: ", err, "after", name)
        rw.WriteHeader(400)
    }
}

func (c *Context) Pause(rw web.ResponseWriter, req *web.Request) {
    addLog(*c.session.Bout, "pause")
    rw.WriteHeader(200)
}


func (c *Context) Logout(rw web.ResponseWriter, req *web.Request) {
    if c.session != nil {
        bout := c.session.Bout
        if bout != nil && bout.Valid() {
            addLog(*bout, "pause")
        }
        mgo_conn := mgo_session.Copy()
        defer mgo_conn.Close()

        mgo_conn.DB(DB_name).C("sessions").Remove(bson.M{"key": c.session.Key})
        c.session = nil
    }
    http.Redirect(rw, req.Request, LOGINSERVER + "logout", http.StatusFound)
}


func (c *Context) CurrentLog(rw web.ResponseWriter, req *web.Request) {
    logs := getBout(c.session.Bout).Logs
    for i := range logs {
        if i > 0 && logs[i-1].Duration == 0 {
            logs[i-1].Duration =
                int(logs[i].Timestamp.Sub(logs[i-1].Timestamp).Seconds())
        }
    }
    data, err := json.Marshal(logs)
    if err != nil {
        panic(err)
    }
    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(200)
    rw.Write(data)
}

func addLog(bout bson.ObjectId, name string) {
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    log := new(LogEntry)
    log.Timestamp = time.Now()
    log.Entry = name
    // fmt.Printf("Add to log for %v: %v\n", bout, name)
    err := mgo_conn.DB(DB_name).C("bouts").UpdateId(bout, bson.M{
        "$push": bson.M{
            "logs": log,
        },
    })
    if err != nil {
        panic(err)
    }
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
    "abcdefghijklmnopqrstuvwxyz" +
    "0123456789")

func makeSession(session *Session, key string) string {
    // Create a new session, store the user, return the session key
    if key == "" {
        keydata := make([]rune, 42)
        for i := range keydata {
            keydata[i] = letters[rand.Intn(len(letters))]
        }
        key = string(keydata)
//    } else {
//        if getSession(key) != nil {
//            panic("Bad session reuse!")
//        }
    }
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    session.Key = key
    _, err := mgo_conn.DB(DB_name).C("sessions").Upsert(
        bson.M{"key": session.Key},
        *session)
    if err != nil {
        panic(err)
    }
    return key
}

func getSession(key string) *Session {
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    var result = new(Session)
    err := mgo_conn.DB(DB_name).C("sessions").Find(bson.M{"key": key}).One(result)
    if err == nil {
        return result
    } else if err == mgo.ErrNotFound {
        return nil
    } else {
        panic(err)
    }
}

func saveSession(session *Session) {
    if (session == nil) {
        return
    }
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()
    // fmt.Println("Trying to save session", session.Key, ":", session)
    _, err := mgo_conn.DB(DB_name).C("sessions").Upsert(
        bson.M{"key": session.Key},
        *session)
    if err != nil {
        panic(err)
    }
}

func (c *Context) KthSessionMiddleware(rw web.ResponseWriter, r *web.Request,
    next web.NextMiddlewareFunc) {
    session, err := r.Cookie("PARSESS")

    if (session != nil) && (err == nil) {
        c.session = getSession(session.Value)
    }
    // fmt.Println("Cookie", session, err, " -> session", c.session)
    if c.session == nil || c.session.User.Kthid == "" {
        ticket := r.URL.Query().Get("ticket")
        if LOGINSERVER == "MOCK" {
            c.session = new(Session)
            c.session.User.Kthid = "u1famwov"
            c.session.User.Firstname = "Rasmus"
            c.session.User.Name = "Rasmus Kaj"
            var oldkey string
            if session != nil { oldkey = session.Value } else { oldkey = "" }
            session = new (http.Cookie)
            session.Name = "PARSESS"
            session.Path = "/"
            session.Domain = SERVERHOST
            session.Value = makeSession(c.session, oldkey)
            session.MaxAge = 3600
            // fmt.Println("Setting cookie", session, "and redirect to hide ticket")
            http.SetCookie(rw, session)
            http.Redirect(rw, r.Request, SERVERURL + r.URL.Path, http.StatusFound)
            return; // early

        } else if ticket == "" {
            v := url.Values{}
            v.Set("service", SERVERURL + r.URL.Path);
            target := LOGINSERVER + "login?" + v.Encode()
            // fmt.Println("Redirecting to", target, "for login");
            http.Redirect(rw, r.Request, target, http.StatusFound)
            return // Early!
        } else {
            // Ok, there seem to be login
            // fmt.Println("Got ticket", ticket, "to validate")
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

            // fmt.Println("User is", userid)
            c.session = new(Session)
            c.session.User = getUser(userid)
            var oldkey string
            if session != nil { oldkey = session.Value } else { oldkey = "" }
            session = new (http.Cookie)
            session.Name = "PARSESS"
            session.Path = "/"
            session.Domain = SERVERHOST
            session.Value = makeSession(c.session, oldkey)
            // fmt.Println("Session:", c.session)
            session.MaxAge = 3600
            // fmt.Println("Setting cookie", session, "and redirect to hide ticket")
            http.SetCookie(rw, session)
            http.Redirect(rw, r.Request, SERVERURL + r.URL.Path, http.StatusFound)
            return; // early
        }
    }
    // fmt.Println("Write session cookie back", session)
    http.SetCookie(rw, session)

    next(rw, r)

    // fmt.Println("Store session to db", c.session)
    saveSession(c.session)
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
        Middleware(web.ShowErrorsMiddleware).
        Get("/suggestuser", SuggestUser)

    router.Subrouter(Context{}, "/static").
        Middleware(web.StaticMiddleware("src/parkour")).
        Get("/style.css", (*Context).MainPage).
        Get("/parkour.js", (*Context).MainPage)

    router.Subrouter(Context{}, "/").
        Middleware((*Context).KthSessionMiddleware).
        Get("/", (*Context).NewBout).
        Get("/bout", (*Context).MainPage).
        Get("/boutlog", (*Context).CurrentLog).
        Get("/logout", (*Context).Logout).
        Put("/driver", (*Context).ChangeDriver).
        Put("/pause", (*Context).Pause)

    http.ListenAndServe("localhost:3000", router)
}
