package main

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
    "fmt"
    "parkour"
)

func main() {
    parkour.InitDb()
    mgo_session, err := mgo.Dial("mongodb://localhost/parkour")
    if err != nil {
        panic(err)
    }
    mgo_conn := mgo_session.Copy()
    defer mgo_conn.Close()

    fmt.Println("Hello?")
    // TODO: Use non-empty logs as selector!
    it := mgo_conn.DB("parkour").C("bouts").Find(bson.M{"user": "u1famwov"}).Iter()
    var doc parkour.Bout
    for it.Next(&doc) {
        // fmt.Println("Document is", doc);
        one := parkour.GetUser(doc.User)
        other := parkour.GetUser(doc.Other)
        fmt.Println("Found bout for", one, "and", other);
        fmt.Println("Orig  logs:", doc.Logs)
        for i := range doc.Logs {
            if doc.Logs[i].Entry == one.Firstname {
                doc.Logs[i].Entry = one.Kthid
            } else if doc.Logs[i].Entry == other.Firstname {
                doc.Logs[i].Entry = other.Kthid
            }
        }
        fmt.Println("Fixed logs:", doc.Logs)
        err := mgo_conn.DB("parkour").C("bouts").UpdateId(doc.Id, doc)
        if err != nil {
            panic(err)
        }
    }
    fmt.Println("Done.")
}
