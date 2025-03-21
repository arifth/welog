package welog

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/cruseraca/welog/pkg/constant/envkey"
	"github.com/cruseraca/welog/pkg/constant/generalkey"
	"github.com/cruseraca/welog/pkg/infrastructure/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var (
	welogConfig = Config{
		ElasticIndex:    "welog2",
		ElasticURL:      "http://182.19.168.194:9200",
		ElasticUsername: "elastic",
		ElasticPassword: "changeme",
		ResBodyType:     "string",
	}
)

// TestSetConfig tests the SetConfig function
func TestSetConfig(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Assert that environment variables are set correctly
	elasticIndex := os.Getenv(envkey.ElasticIndex)
	assert.Equal(t, welogConfig.ElasticIndex, elasticIndex, "ElasticIndex should be set correctly")

	elasticURL := os.Getenv(envkey.ElasticURL)
	assert.Equal(t, welogConfig.ElasticURL, elasticURL, "ElasticURL should be set correctly")

	elasticUsername := os.Getenv(envkey.ElasticUsername)
	assert.Equal(t, welogConfig.ElasticUsername, elasticUsername, "ElasticUsername should be set correctly")

	elasticPassword := os.Getenv(envkey.ElasticPassword)
	assert.Equal(t, welogConfig.ElasticPassword, elasticPassword, "ElasticPassword should be set correctly")
}

// TestNewFiber tests the NewFiber middleware to ensure it sets up the Fiber application correctly.
func TestNewFiber(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Fiber app and apply the middleware.
	app := fiber.New()
	app.Use(NewFiber(fiber.Config{}))

	// Create a new HTTP GET request.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Test-Agent")

	// Perform the request and capture the response.
	resp, err := app.Test(req, 5000) //nolint:bodyclose

	// Assert that there are no errors and the status is 404 Not Found.
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// TestLogFiber tests the logFiber function within the Fiber middleware.
func TestLogFiber(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Fiber app.
	app := fiber.New()

	// Create a POST request with a JSON body.
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"key": "value"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	// Define a middleware that logs the request using logFiber.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(generalkey.Logger, logger.Logger().WithField(generalkey.RequestID, c.Locals("requestid")))
		c.Locals(generalkey.ClientLog, []logrus.Fields{})
		logFiber(c, time.Now())
		return c.SendStatus(fiber.StatusOK)
	})

	// Perform the request and capture the response.
	_, err := app.Test(req, -1) //nolint:bodyclose
	assert.NoError(t, err)

	// Assert that the status code is 200 OK.
	assert.Equal(t, fiber.StatusOK, resp.Code)
}

// TestLogFiberClient tests the LogFiberClient function to ensure it logs client requests and responses correctly.
func TestLogFiberClient(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Fiber app.
	app := fiber.New()

	// Acquire a new context from the Fiber app for testing.
	fastCtx := &fasthttp.RequestCtx{}
	fiberCtx := app.AcquireCtx(fastCtx)
	defer app.ReleaseCtx(fiberCtx)

	// Set initial client log fields.
	fiberCtx.Locals(generalkey.ClientLog, []logrus.Fields{})

	// Define test input values.
	url := "https://example.com"
	method := "GET"
	contentType := "application/json"
	header := map[string]interface{}{"Content-Type": "application/json"}
	responseHeader := map[string]interface{}{"Content-Type": "application/json"}
	body := []byte(`{"test": "data"}`)
	response := []byte(`{"response": "ok"}`)
	status := http.StatusOK
	start := time.Now()
	elapsed := 100 * time.Millisecond

	// Log the client request and response.
	LogFiberClient(fiberCtx, url, method, contentType, header, body, responseHeader, response, status, start, elapsed)

	// Retrieve the client log and assert that it contains the correct values.
	clientLog := fiberCtx.Locals(generalkey.ClientLog).([]logrus.Fields)
	assert.Len(t, clientLog, 1)
	assert.Equal(t, status, clientLog[0]["targetResponseStatus"])
}

// TestNewGin tests the NewGin middleware to ensure it sets up the Gin application correctly.
func TestNewGin(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Gin router and apply the middleware.
	r := gin.New()
	r.Use(NewGin())

	type BodyReq struct {
		Key string `json:"key"`
	}

	// Define a simple GET endpoint.
	r.GET("/", func(c *gin.Context) {
		respBody := `{"key": "test"}`
		//var respBody []map[string]any
		var reqBody BodyReq
		c.BindJSON(&reqBody)
		respBody = respBody
		c.JSON(http.StatusOK, respBody)
	})

	// Create a GET request with a custom Request ID.
	req, _ := http.NewRequest(http.MethodGet, "/", bytes.NewBuffer([]byte(`{"key": "test"}`)))

	w := httptest.NewRecorder()

	// Serve the request and capture the response.
	r.ServeHTTP(w, req)

	// Assert that the response status and body are correct.
	assert.Equal(t, http.StatusOK, w.Code)
	//assert.Equal(t, "ok", w.Body.String())
}

func TestNewGinWithError(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Gin router and apply the middleware.
	r := gin.New()
	r.Use(NewGin())
	r.Use(requestid.New())

	// Define a simple GET endpoint.
	r.GET("/", func(c *gin.Context) {
		err := errors.New("something went wrong")
		c.Set(generalkey.ErrorLog, err)
		c.JSON(http.StatusInternalServerError, nil)
	})

	// Create a GET request with a custom Request ID.
	req, _ := http.NewRequest(http.MethodGet, "/", bytes.NewBuffer([]byte(`asdfasdf`)))
	w := httptest.NewRecorder()

	// Serve the request and capture the response.
	r.ServeHTTP(w, req)

	// Assert that the response status and body are correct.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	//assert.Equal(t, "ok", w.Body.String())
}

// TestLogGin tests the logGin function within the Gin middleware.
func TestLogGin(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a buffer and logger to capture log output.
	buf := &bytes.Buffer{}
	log := logrus.New()
	log.Out = buf

	// Create a POST request with a JSON body.
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"key": "value"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a Gin context for testing.
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set the logger and client log fields.
	c.Set(generalkey.Logger, log.WithField(generalkey.RequestID, "test-request-id"))
	c.Set(generalkey.ClientLog, []logrus.Fields{})

	// Capture the response body using a custom response writer.
	bodyBuf := &bytes.Buffer{}
	c.Writer = &responseBodyWriter{body: bodyBuf, ResponseWriter: c.Writer}

	// Log the request and response.
	requestTime := time.Now()
	logGin(c, bodyBuf, requestTime)

	// Retrieve and assert the log output.
	logOutput := buf.String()
	assert.Contains(t, logOutput, `requestMethod=POST`)
	assert.Contains(t, logOutput, `responseStatus=200`)
}

// TestLogGinClient tests the LogGinClient function to ensure it logs client requests and responses correctly.
func TestLogGinClient(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a POST request with a JSON body.
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"key": "value"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a Gin context for testing.
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set initial client log fields.
	c.Set(generalkey.ClientLog, []logrus.Fields{})

	// Define test input values.
	url := "https://example.com"
	method := "POST"
	contentType := "application/json"
	header := map[string]interface{}{"Content-Type": "application/json"}
	responseHeader := map[string]interface{}{"Content-Type": "application/json"}
	body := []byte(`{"test": "data"}`)
	response := []byte(`{"response": "ok"}`)
	status := http.StatusOK
	start := time.Now()
	elapsed := 100 * time.Millisecond

	// Log the client request and response.
	LogGinClient(c, url, method, contentType, header, body, responseHeader, response, status, start, elapsed)

	// Retrieve the client log and assert that it contains the correct values.
	clientLog, exists := c.Get(generalkey.ClientLog)
	assert.True(t, exists)
	logFields := clientLog.([]logrus.Fields)
	assert.Len(t, logFields, 1)
	assert.Equal(t, status, logFields[0]["targetResponseStatus"])
	assert.Equal(t, "POST", logFields[0]["targetRequestMethod"])
}

func TestLogTarget(t *testing.T) {
	// Call the SetConfig function
	SetConfig(welogConfig)

	// Create a new Gin router and apply the middleware.
	r := gin.New()
	r.Use(NewGin())

	// Define a simple GET endpoint.
	r.GET("/", func(c *gin.Context) {
		// Make a client request to the target server.
		err := makeHTTPRequest(c)
		if err != nil {
			c.Set(generalkey.ErrorLog, err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Create a GET request with a custom Request ID.
	req, _ := http.NewRequest(http.MethodGet, "/", bytes.NewBuffer([]byte(`{"key": "value"}`)))
	w := httptest.NewRecorder()

	// Serve the request and capture the response.
	r.ServeHTTP(w, req)

	// Assert that the response status and body are correct.
	assert.Equal(t, http.StatusOK, w.Code)
	//assert.Equal(t, "ok", w.Body.String())
}

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func makeHTTPRequest(ctx *gin.Context) error {
	// Initialize a new Resty client
	client := resty.New()

	// Define middleware logging
	middlewareTargetLog(ctx, client)

	// Define the endpoint URL (a public mock server)
	url := "https://httpstat.us/502"
	// Make the GET request
	resp, err := client.R().
		SetDebug(true).
		SetResult(&Post{}). // Set the result to a Post struct to automatically parse JSON
		Get(url)

	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	// Check the status code
	if resp.IsError() {
		fmt.Println("Error:", resp.Status())
	}

	return nil
}

func middlewareTargetLog(ctx *gin.Context, client *resty.Client) {
	var targetLog Target
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {

		// manipulate it as per your need
		targetLog.TargetUrl = req.URL
		targetLog.Method = req.Method
		targetLog.RequestHeader = req.Header
		targetLog.RequestBody = req.Body

		return nil // if its success otherwise return error
	})

	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		// manipulate it as per your need
		targetLog.ResponseHeader = resp.Header()
		targetLog.ResponseBody = resp.Body()
		targetLog.StatusCode = resp.StatusCode()
		targetLog.ElapsedTime = resp.Time()

		LogTarget(ctx, targetLog)
		return nil
	})

	client.OnError(func(req *resty.Request, err error) {
		// manipulate it as per your need
		ctx.Set(generalkey.ErrorLog, err)

		LogTarget(ctx, targetLog)
	})
}
