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
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/esimov/caire"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/mgo.v2"
	"github.com/pkg/profile"
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
	daily()
}

func daily() {
	defer profile.Start(profile.MemProfile).Stop()
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
	fmt.Println("Resized image")
	S3Upload(resizedBuffer)          // uploads to s3
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