package main

import (
	//"runtime/debug"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"	
	"io"
	"io/ioutil"
	//"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	//"github.com/pkg/profile"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/esimov/caire"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/mgo.v2"
)

// cd server && go run server.go
// go run Documents/reddit-chrome-wallpapers/server/server.go
// curl -X POST -H "Content-Type: application/json" http://localhost:8080 -d '{"height": 800, "width": 900}'

type Posts struct {
	Data struct {
		Children []PostDetails
	}
}

type PostDetails struct {
	Data struct {
		Title     string
		Author    string
		Permalink string
		Url       string
		Preview   struct {
			Images []ImageDetails
		}
	}
}

type SendData struct {
	Author 	  string
	Permalink string
}

type ImageDetails struct {
	Source struct {
		Width  int
		Height int
	}
}

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "update" {
		daily()
	} else {
		port := getPort()
		sd := getData()
		http.HandleFunc("/details", sd.handleDetails)
		http.ListenAndServe(port, nil)
	}
}

func daily() {
	subData := GetSubData() // stores all of the data for r/EarthPorn/top
	imgData, details, searchErr := FindBestImage(subData)
	fmt.Println("Got image data")
	if searchErr != nil { // don't update the image today if there were no suitable posts
		return
	}
	
	insertData(details)
	imgReader := bytes.NewBuffer(imgData) // buffer reader that holds the pre-processed image
	resizedBuffer := new(bytes.Buffer)

	resize(imgReader, resizedBuffer) // resizes image, stores it in resized buffer
	//debug.FreeOSMemory()
	fmt.Println("Resized image")
	S3Upload(resizedBuffer)          // uploads to s3
}

func getData() *SendData {
	c := getCollection()
	var m bson.M
	dbSize, err := c.Count()
	check(err)
	err = c.Find(nil).Skip(dbSize - 1).One(&m)
	check(err)
	sd := SendData{Author: m["Author"].(string), Permalink: m["Link"].(string)}
	fmt.Println("got data from mongo", sd)
	return &sd
}

func insertData(details *PostDetails) {
	c := getCollection()
	err := c.Insert(bson.M{"Author": details.Data.Author, "Link": details.Data.Permalink})
	check(err)
	fmt.Println("inserted table")
}

func getCollection() *mgo.Collection {
	session, err := mgo.Dial("mongodb://micah:afF2cU9PtvJ8yb@ds261479.mlab.com:61479/heroku_v9g0gb74")
	check(err)
	c := session.DB("heroku_v9g0gb74").C("details")
	return c
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

func S3Upload(data *bytes.Buffer) {

	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-2")}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	// Upload the file to S3.
	result, uploadErr := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String("reddit-chrome-wallpapers"),
		Key:         aws.String("daily-image.jpg"),
		ACL:         aws.String("public-read"),
		ContentType: aws.String("image/jpeg"),
		Body:        data,
	})
	check(uploadErr)

	fmt.Println("file uploaded to S3", result)
}

func FindBestImage(data []byte) ([]byte, *PostDetails, error) {
	var posts Posts
	err := json.Unmarshal(data, &posts)
	check(err)

	for i, post := range posts.Data.Children {
		fmt.Println(i)
		isOC := strings.Contains(post.Data.Title, "[OC]") || strings.Contains(post.Data.Title, "(OC)")
		width := post.Data.Preview.Images[0].Source.Width
		height := post.Data.Preview.Images[0].Source.Height
		asp_ratio := float64(width) / float64(height)

		if isOC && asp_ratio > 1.5 { // find first OC post with acceptable aspect ratio
			fmt.Println(asp_ratio)
			res, getErr := http.Get(post.Data.Url)
			check(getErr)
			defer res.Body.Close()

			buf, readErr := ioutil.ReadAll(res.Body)
			check(readErr)

			return buf, &post, nil
		}
	}

	return nil, nil, errors.New("no posts")
}

func GetSubData() []byte {
	reqUrl, parseErr := url.Parse("https://old.reddit.com/r/EarthPorn/top/.json")
	check(parseErr)

	req := &http.Request{ // set request params
		Method: "GET",
		URL:    reqUrl,
		Header: map[string][]string{
			"User-agent": {"macOS:https://github.com/micahcantor/reddit-chrome-wallpapers:0.1.0 (by /u/HydroxideOH-)"},
		},
	}

	res, getErr := http.DefaultClient.Do(req) // perform GET request
	check(getErr)
	defer res.Body.Close()

	body, readErr := ioutil.ReadAll(res.Body) // read response body
	check(readErr)
	return body
}

func resize(in io.Reader, out io.Writer) {
	//defer profile.Start(profile.MemProfile).Stop()
	p := &caire.Processor {
		NewWidth:  1920,
		NewHeight: 1080,
		Scale: true,
	}

	err := p.Process(in, out)
	check(err)
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