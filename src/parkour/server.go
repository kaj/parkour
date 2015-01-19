package parkour

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

var (
	mgo_session *mgo.Session
	DB_name     string
)

type Context struct {
	session *Session
}

var courses = map[string]string{
	"aaaa":    "Obefintlig testkurs",
	"adk":     "Algoritmer, Datastrukturer, Komplexitet",
	"cprog":   "Programkonstruktion i C++",
	"prgcl":   "Programmeringsteknik för Civilingenjör & Lärare",
	"prgomed": "Programmeringsteknik för Media",
	"prgs":    "Programmeringsteknik för S",
	"tilda":   "Tillämpad datalogi",
	"nump":    "Numeriska metoder och grundläggande programmering",
}
var labs = map[string]string{
	"labb1": "Labb 1",
	"labb2": "Labb 2",
	"labb3": "Labb 3",
	"labb4": "Labb 4",
	"labb5": "Labb 5",
	"labb6": "Labb 6",
	"labb7": "Labb 7",
}

type Session struct {
	Key  string
	User User
	Bout *bson.ObjectId
}

func findKthid(user string) string {
	i := strings.Index(user, "(")
	j := strings.Index(user, ")")
	if i >= 0 && j > 0 {
		return user[i+1 : j]
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
		withuser := GetUser(withkthid)

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
			panic(err)
		}
		c.session.Bout = &bout.Id

		http.Redirect(rw, req.Request, SERVERURL+"/bout", http.StatusFound)
		return
	}
	tpl := template.Must(template.ParseFiles("src/parkour/templates/newbout.html"))
	tpl.Execute(rw, map[string]interface{}{
		"User":    c.session.User,
		"courses": courses,
		"labs":    labs,
	})
}

func (c *Context) MainPage(rw web.ResponseWriter, req *web.Request) {
	tpl := template.Must(template.ParseFiles("src/parkour/templates/mainpage.html"))

	bout := getBout(c.session.Bout)
	if bout == nil {
		http.Redirect(rw, req.Request, SERVERURL+"/", http.StatusFound)
		return
	}
	tpl.Execute(rw, map[string]interface{}{
		"User":  c.session.User,
		"kurs":  courses[bout.Course],
		"lab":   labs[bout.Lab],
		"me":    c.session.User,
		"other": GetUser(bout.Other),
	})
}


func (c *Context) History(rw web.ResponseWriter, req *web.Request) {
    tpl := template.Must(template.ParseFiles("src/parkour/templates/history.html"))

    course := req.FormValue("course")
    lab := req.FormValue("lab")

    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    var bouts []Bout
    filter := bson.M {
        "user": c.session.User.Kthid,
        "course": course,
    }
    if lab != "" {
        filter["lab"] = lab
    }
    err := mgo_conn.DB(DB_name).C("bouts").Find(filter).All(&bouts)
    if err != nil {
        panic(err)
    }
    my, others := 0, 0
    othername := ""
    for _, bout := range bouts {
        dd := bout.DriverDurations()
        my += dd.MySeconds
        others += dd.OthersSeconds
        othername = dd.OthersNames
    }
    balance := &Balance{c.session.User.Kthid, my, othername, others}
    foo := tpl.Execute(rw, map[string]interface{}{
        "User": c.session.User,
        "balance": balance,
        "bouts": bouts,
        "courses": courses,
        "course":  course,
        "labs":    labs,
        "lab":     lab,
    })
    if (foo != nil) {
        fmt.Println("Template error:", foo);
    }
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

func GetUser(kthid string) User {
	mgo_conn := mgo_session.Copy()
	defer mgo_conn.Close()

	var result User
	err := mgo_conn.DB(DB_name).C("users").Find(bson.M{"kthid": kthid}).One(&result)
	if err == nil {
		return result
	} else {
		// No such user!  Check with LDAP!
        // return User{kthid, kthid, kthid}
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
		io.WriteString(rw, "\""+obj["name"].(string)+
			" ("+obj["kthid"].(string)+")\"")
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
	http.Redirect(rw, req.Request, LOGINSERVER+"logout", http.StatusFound)
}

func (c *Context) CurrentLog(rw web.ResponseWriter, req *web.Request) {
    logs := getBout(c.session.Bout).GetLogs()
    for i := range logs {
        if logs[i].Entry != "pause" {
            logs[i].Entry = GetUser(logs[i].Entry).Name
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
	if session == nil {
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
			if session != nil {
				oldkey = session.Value
			} else {
				oldkey = ""
			}
			session = new(http.Cookie)
			session.Name = "PARSESS"
			session.Path = "/"
			session.Domain = SERVERHOST
			session.Value = makeSession(c.session, oldkey)
			session.MaxAge = 3600
			// fmt.Println("Setting cookie", session, "and redirect to hide ticket")
			http.SetCookie(rw, session)
			http.Redirect(rw, r.Request, SERVERURL+r.URL.Path, http.StatusFound)
			return // early

		} else if ticket == "" {
			v := url.Values{}
			v.Set("service", SERVERURL+r.URL.Path)
			target := LOGINSERVER + "login?" + v.Encode()
			// fmt.Println("Redirecting to", target, "for login");
			http.Redirect(rw, r.Request, target, http.StatusFound)
			return // Early!
		} else {
			// Ok, there seem to be login
			// fmt.Println("Got ticket", ticket, "to validate")
			v := url.Values{}
			v.Set("ticket", ticket)
			v.Set("service", SERVERURL+r.URL.Path)
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
			c.session.User = GetUser(userid)
			var oldkey string
			if session != nil {
				oldkey = session.Value
			} else {
				oldkey = ""
			}
			session = new(http.Cookie)
			session.Name = "PARSESS"
			session.Path = "/"
			session.Domain = SERVERHOST
			session.Value = makeSession(c.session, oldkey)
			// fmt.Println("Session:", c.session)
			session.MaxAge = 3600
			// fmt.Println("Setting cookie", session, "and redirect to hide ticket")
			http.SetCookie(rw, session)
			http.Redirect(rw, r.Request, SERVERURL+r.URL.Path, http.StatusFound)
			return // early
		}
	}
	// fmt.Println("Write session cookie back", session)
	http.SetCookie(rw, session)

	next(rw, r)

	// fmt.Println("Store session to db", c.session)
	saveSession(c.session)
}

func InitDb() {
	session, err := mgo.Dial("mongodb://localhost/parkour")
	if err != nil {
		panic(err)
	}
	mgo_session = session
	DB_name = "parkour"
}

