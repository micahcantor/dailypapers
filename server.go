package main

import (
	"fmt"
	"io"
	"errors"
	"bytes"
	"encoding/json"
	"strings"
	"io/ioutil"
	"net/http"
	"net/url"
	"github.com/esimov/caire"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// cd server && go run server.go
// go run Documents/reddit-chrome-wallpapers/server/server.go
// curl -X POST -H "Content-Type: application/json" http://localhost:8080 -d '{"height": 800, "width": 900}'

type Dimensions struct {
	Width int 
	Height int
}

type Posts struct {
	Data struct {
		Children []PostDetails
	}
}

type PostDetails struct {
	Data struct {
		Title string
		Author string
		Permalink string
		Url string
		Preview struct {
			Images []ImageDetails
		}
	}
}

type ImageDetails struct {
	Source struct {
		Width int
		Height int
	}
}

func main() {
	http.HandleFunc("/details", details.HandleDetails)
	http.ListenAndServe(":8080", nil)
}

func daily() {
	subData := GetSubData() // stores all of the data for r/EarthPorn/top
	imgData, details, searchErr := FindBestImage(subData)
	if searchErr != nil {	// don't update the image today if there were no suitable posts
		return
	}
	imgReader := bytes.NewBuffer(imgData) // buffer reader that holds the pre-processed image
	resizedBuffer := new(bytes.Buffer)

	resize(imgReader, resizedBuffer) // resizes image, stores it in resized buffer
	S3Upload(resizedBuffer)   // uploads to s3
}

func (pd *PostDetails) HandleDetails(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "invalid_http_method")
		return
	}
	
	json, encodeErr := json.Marshal(pd.Data)
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
		Bucket: aws.String("reddit-chrome-wallpapers"),
		Key:    aws.String("daily-image.jpg"),
		ACL: 	aws.String("public-read"),
		ContentType: aws.String("image/jpeg"),
		Body:   data,
	})
	check(uploadErr)

	fmt.Printf("file uploaded to S3", result)	
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

		if isOC && asp_ratio > 1.32 { // find first OC post with acceptable aspect ratio
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

func GetSubData() ([]byte) {
	reqUrl, parseErr := url.Parse("https://old.reddit.com/r/EarthPorn/top/.json")
	check(parseErr)

	req := &http.Request { // set request params
		Method: "GET",
		URL: reqUrl,
		Header: map[string][]string {
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
	p := &caire.Processor{
		NewWidth: 1920,
		NewHeight: 1080,
	}

	err := p.Process(in, out)
	check(err)
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}