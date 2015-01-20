package parkour

import (
    "gopkg.in/mgo.v2/bson"
    "os/exec"
    "regexp"
)

type User struct {
    Kthid     string
    Firstname string
    Name      string
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
