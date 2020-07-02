package main

import (
	"bytes"
	"fmt"
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"net/http"
	"strings"
	"encoding/json"

	"github.com/esimov/caire"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/mgo.v2"
	"github.com/aws/aws-lambda-go/lambda"
)

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

type ImageDetails struct {
	Source struct {
		Width  int
		Height int
	}
}

func main() {
	lambda.Start(daily)
}

func daily() {
	subData := GetSubData() // stores all of the data for r/EarthPorn/top
	imgData, details, searchErr := FindBestImage(subData)
	if searchErr != nil { // don't update the image today if there were no suitable posts
		return
	}

	imgReader := bytes.NewBuffer(imgData) // buffer reader that holds the pre-processed image
	resizedBuffer := new(bytes.Buffer)

	resize(imgReader, resizedBuffer) // resizes image, stores it in resized buffer
	imageURL := imgurUpload(resizedBuffer)

	insertData(details, imageURL)
}

func insertData(details *PostDetails, imageURL string) {
	c := getCollection()
	err := c.Insert(bson.M{"Author": details.Data.Author, "Link": details.Data.Permalink, "ImageURL": imageURL})
	check(err)
	fmt.Println("inserted table")
}

func getCollection() *mgo.Collection {
	session, err := mgo.Dial("mongodb://micah:afF2cU9PtvJ8yb@ds261479.mlab.com:61479/heroku_v9g0gb74")
	check(err)
	c := session.DB("heroku_v9g0gb74").C("details")
	return c
}

func imgurUpload(data *bytes.Buffer) string {
	url := "https://api.imgur.com/3/image"
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, data)
	check(err)

	req.Header.Add("Authorization", "Client-ID 6d76ba00db84cc6")
	req.Header.Set("Content-Type", "image/jpg")

	res, err := client.Do(req)
	check(err)
	defer res.Body.Close()

	type Result struct {
		Data struct {
			Link string
		}
	}
	var r Result

	err = json.NewDecoder(res.Body).Decode(&r)
	check(err)
	fmt.Println("image uploaded at " + r.Data.Link)
	return r.Data.Link
}

func FindBestImage(data []byte) ([]byte, *PostDetails, error) {
	var posts Posts
	err := json.Unmarshal(data, &posts)
	check(err)

	for _, post := range posts.Data.Children {
		width := post.Data.Preview.Images[0].Source.Width
		height := post.Data.Preview.Images[0].Source.Height
		asp_ratio := float64(width) / float64(height)

		isOC := strings.Contains(post.Data.Title, "[OC]") || strings.Contains(post.Data.Title, "(OC)")
		good_asp_ratio := asp_ratio > 1.5 && asp_ratio < 1.9

		if isOC && good_asp_ratio  { // find first OC post with acceptable aspect ratio
			fmt.Println(asp_ratio)
			res, getErr := http.Get(post.Data.Url)
			check(getErr)
			defer res.Body.Close()

			buf, readErr := ioutil.ReadAll(res.Body)
			check(readErr)

			return buf, &post, nil
		}
	}
	fmt.Println("Got image data")
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
	p := &caire.Processor {
		NewWidth:  1920,
		NewHeight: 1080,
		Scale: 	   true,
	}

	err := p.Process(in, out)
	check(err)
	fmt.Println("Resized image")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}