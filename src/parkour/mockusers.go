package main

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	Kthid     string
	Firstname string
	Name      string
}

func upsertUser(users *mgo.Collection, kthid, givenname, name string) {
	_, err := users.Upsert(bson.M{"kthid": kthid}, User{kthid, givenname, name})
	if err != nil {
		panic(err)
	}
}

func main() {
	mgo_session, err := mgo.Dial("mongodb://localhost/parkour")
	if err != nil {
		panic(err)
	}
	mgo_conn := mgo_session.Copy()
	defer mgo_conn.Close()

	users := mgo_conn.DB("parkour").C("users")
	upsertUser(users, "u1famwov", "Rasmus", "Rasmus Kaj")
	upsertUser(users, "u1i6bme8", "Marcus", "Marcus Dicander")
	upsertUser(users, "u1kd2mni", "Viggo", "Viggo Kann")
}
