package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"html/template"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/azure-sdk-for-go/services/cognitiveservices/v2.0/computervision"
	"github.com/Azure/go-autorest/autorest"
)

var (
	accountName string = os.Getenv("AZURE_STORAGE_ACCOUNT")
	accountKey  string = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	computerVisionKey string = os.Getenv("COMPUTER_VISION_SUBSCRIPTION_KEY")
	computerVisionEndpointURL string = os.Getenv("COMPUTER_VISION_ENDPOINT")
)

const containerName string = "imgup"

func randomString(sLength int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	
	s := make([]byte, sLength)
    for i := range s {
        s[i] = letterBytes[rand.Intn(len(letterBytes))]
    }
    return string(s)
}

func handleErrors(err error) {
	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok { // This error is a Service-specific
			switch serr.ServiceCode() { // Compare serviceCode to ServiceCodeXxx constants
			case azblob.ServiceCodeContainerAlreadyExists:
				fmt.Println("Received 409. Container already exists")
				return
			}
		}
		log.Fatal(err)
	}
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {

	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	if len(accountName) == 0 || len(accountKey) == 0 {
		log.Fatal("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}
	if (computerVisionKey == "") {
		log.Fatal("\n\nPlease set a COMPUTER_VISION_SUBSCRIPTION_KEY environment variable.\n" +
							  "**You may need to restart your shell or IDE after it's set.**\n")
	}
	if (computerVisionEndpointURL == "") {
		log.Fatal("\n\nPlease set a COMPUTER_VISION_ENDPOINT environment variable.\n" +
							  "**You may need to restart your shell or IDE after it's set.**")
	}

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("page/*.html")),
	}

	// Echo instance
	e := echo.New()

	e.Renderer = renderer

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", pageHome)
	e.GET("/i/:id", pageImageDetail)
	e.PUT("/api/upload", apiUpload)
	e.GET("/api/recent", apiRecent)
	e.GET("/api/detail/:id", apiImageDetail)

	// Static
	e.Static("/static", "static")

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

func pageHome(c echo.Context) error {
	return c.Render(http.StatusOK, "home.html", map[string]string{})
}

func pageImageDetail(c echo.Context) error {
	data := map[string]interface{}{}
	data["id"] = c.Param("id")

	storageURL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	data["storageURL"] = storageURL
	return c.Render(http.StatusOK, "imagedetail.html", data)
}

func apiImageDetail(c echo.Context) error {
	computerVisionClient := computervision.New(computerVisionEndpointURL);
	computerVisionClient.Authorizer = autorest.NewCognitiveServicesAuthorizer(computerVisionKey)

	storageURL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	id := c.Param("id")
	outputJson := map[string]interface{}{}
	outputJson["id"] = id
	ctx := context.Background()
	url := fmt.Sprintf("%s/%s", storageURL, id)
	imageURL := computervision.ImageURL{
		URL: &url,
	}
	maxNumberDescriptionCandidates := new(int32)
    *maxNumberDescriptionCandidates = 1
	imageDescription, err := computerVisionClient.DescribeImage(ctx, imageURL, maxNumberDescriptionCandidates, "en")
	handleErrors(err)
	outputJson["captions"] = imageDescription.Captions
	outputJson["tags"] = imageDescription.Tags
	return c.JSON(http.StatusOK, outputJson)
}

func apiRecent(c echo.Context) error {
	// Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	storageURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName)
	parsedStorageURL, _ := url.Parse(storageURL)

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*parsedStorageURL, p)

	ctx := context.Background() // This example uses a never-expiring context

	outputJson := map[string]interface{}{}
	var recentID []string

	// List the container that we have created above
	for marker := (azblob.Marker{}); marker.NotDone(); {
		// Get a result segment starting with the blob indicated by the current Marker.
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		handleErrors(err)

		// ListBlobs returns the start of the next segment; you MUST use this to get
		// the next segment (after processing the current result segment).
		marker = listBlob.NextMarker

		// Process the blobs returned in this result segment (if the segment is empty, the loop body won't execute)
		for _, blobInfo := range listBlob.Segment.BlobItems {
			recentID = append(recentID, blobInfo.Name)
		}
	}
	outputJson["ids"] = recentID
	outputJson["storageURL"] = storageURL
	return c.JSON(http.StatusOK, outputJson)
}

func apiUpload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create("tempfile")
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)

	// Create the container
	fmt.Printf("Creating a container named %s\n", containerName)
	ctx := context.Background() // This example uses a never-expiring context
	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	handleErrors(err)

	fileName := randomString(6)
	blobURL := containerURL.NewBlockBlobURL(fileName)
	file2, err := os.Open("tempfile")
	handleErrors(err)

	// You can use the low-level PutBlob API to upload files. Low-level APIs are simple wrappers for the Azure Storage REST APIs.
	// Note that PutBlob can upload up to 256MB data in one shot. Details: https://docs.microsoft.com/en-us/rest/api/storageservices/put-blob
	// Following is commented out intentionally because we will instead use UploadFileToBlockBlob API to upload the blob
	// _, err = blobURL.PutBlob(ctx, file, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	// handleErrors(err)

	// The high-level API UploadFileToBlockBlob function uploads blocks in parallel for optimal performance, and can handle large files as well.
	// This function calls PutBlock/PutBlockList for files larger 256 MBs, and calls PutBlob for any file smaller
	_, err = azblob.UploadFileToBlockBlob(ctx, file2, blobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	handleErrors(err)
	outputJson:= map[string]interface{}{
		"success": true,
		"id": fileName,
	}
	return c.JSON(http.StatusOK, outputJson)
}
