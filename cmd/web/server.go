
package main

import (
	"encoding/json"
	"fmt"	
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/mgo.v2"
)

type SendData struct {
	Author 	  string
	Permalink string
	ImageURL  string
}

func main() {
	port := getPort()
	sd := getData()
	http.HandleFunc("/details", sd.handleDetails)
	http.ListenAndServe(port, nil)
}

func (sd *SendData) handleDetails(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "invalid_http_method")
		return
	}

	json, encodeErr := json.Marshal(sd)
	check(encodeErr)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(json)
}

func getData() *SendData {
	c := getCollection()
	var m bson.M
	dbSize, err := c.Count()
	check(err)
	err = c.Find(nil).Skip(dbSize - 1).One(&m)
	check(err)
	sd := SendData{Author: m["Author"].(string), Permalink: m["Link"].(string), ImageURL: m["ImageURL"].(string)}
	fmt.Println("got data from mongo", sd)
	return &sd
}

func getCollection() *mgo.Collection {
	session, err := mgo.Dial("mongodb://micah:afF2cU9PtvJ8yb@ds261479.mlab.com:61479/heroku_v9g0gb74")
	check(err)
	c := session.DB("heroku_v9g0gb74").C("details")
	return c
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getPort() string {
	p := os.Getenv("PORT")
	if p != "" {
		return ":" + p
	}
	return ":8080"
}