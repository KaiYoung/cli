package api

import (
	"cf"
	term "cf/terminal"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const PRIVATE_DATA_PLACEHOLDER = "[PRIVATE DATA HIDDEN]"

type Request struct {
	*http.Request
}

func NewRequest(method, path, accessToken string, body io.Reader) (authReq *Request, err error) {
	request, err := http.NewRequest(method, path, body)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", accessToken)
	request.Header.Set("accept", "application/json")
	request.Header.Set("User-Agent", "go-cli "+cf.Version+" / "+runtime.GOOS)
	authReq = &Request{request}
	return
}

func PerformRequestAndParseResponse(request *Request, response interface{}) (errorCode int, err error) {
	rawResponse, errorCode, err := doRequest(request.Request)
	if err != nil {
		return
	}
	errorCode, err = parseResponse(rawResponse, response)
	return
}

type ApiClient struct {
	authenticator Authenticator
}

func NewApiClient(auth Authenticator) (client ApiClient) {
	client.authenticator = auth
	return
}

func (c ApiClient) PerformRequest(request *Request) (errorCode int, err error) {
	_, errorCode, err = c.doRequestHandlingAuth(request)
	return
}

func (c ApiClient) PerformRequestAndParseResponse(request *Request, response interface{}) (errorCode int, err error) {
	rawResponse, errorCode, err := c.doRequestHandlingAuth(request)
	if err != nil {
		return
	}
	errorCode, err = parseResponse(rawResponse, response)
	return
}

func (c ApiClient) doRequestHandlingAuth(request *Request) (response *http.Response, errorCode int, err error) {
	response, errorCode, err = doRequest(request.Request)

	if err != nil && response == nil {
		println("Error", err.Error())
		return
	}

	if response.StatusCode == 401 && errorCode == 1000 {
		newToken, err := c.authenticator.RefreshAuthToken()
		if err == nil {
			request.Header.Set("Authorization", newToken)
			return doRequest(request.Request)
		}
	}

	return
}

type errorResponse struct {
	Code        int
	Description string
}

func newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}
	return &http.Client{Transport: tr}
}

func doRequest(request *http.Request) (response *http.Response, errorCode int, err error) {
	httpClient := newHttpClient()

	if traceEnabled() {
		dumpedRequest, err := httputil.DumpRequest(request, true)
		if err != nil {
			fmt.Println("Error dumping request")
		} else {
			fmt.Printf("\n%s\n%s\n", term.Cyan("REQUEST:"), Sanitize(string(dumpedRequest)))
		}
	}

	response, err = httpClient.Do(request)

	if err != nil {
		err = errors.New(fmt.Sprintf("Error performing request: %s", err.Error()))
		return
	}

	if traceEnabled() {
		dumpedResponse, err := httputil.DumpResponse(response, true)
		if err != nil {
			fmt.Println("Error dumping response")
		} else {
			fmt.Printf("\n%s\n%s\n", term.Cyan("RESPONSE:"), Sanitize(string(dumpedResponse)))
		}
	}

	if response.StatusCode > 299 {
		errorResponse := getErrorResponse(response)
		errorCode = errorResponse.Code
		message := fmt.Sprintf("Server error, status code: %d, error code: %d, message: %s", response.StatusCode, errorCode, errorResponse.Description)
		err = errors.New(message)
	}

	return
}

func Sanitize(input string) (sanitized string) {
	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized = re.ReplaceAllString(input, "Authorization: "+PRIVATE_DATA_PLACEHOLDER)
	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PRIVATE_DATA_PLACEHOLDER+"&")
	re = regexp.MustCompile(`"access_token":"[^"]*"`)
	sanitized = re.ReplaceAllString(sanitized, `"access_token":"`+PRIVATE_DATA_PLACEHOLDER+`"`)
	re = regexp.MustCompile(`"refresh_token":"[^"]*"`)
	sanitized = re.ReplaceAllString(sanitized, `"refresh_token":"`+PRIVATE_DATA_PLACEHOLDER+`"`)
	return
}

func parseResponse(rawResponse *http.Response, response interface{}) (errorCode int, err error) {
	jsonBytes, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
		return
	}

	err = json.Unmarshal(jsonBytes, &response)

	if err != nil {
		err = errors.New(fmt.Sprintf("Invalid JSON response from server: %s", err.Error()))
	}

	return
}

func traceEnabled() bool {
	traceEnv := strings.ToLower(os.Getenv("CF_TRACE"))
	return traceEnv == "true" || traceEnv == "yes"
}

func getErrorResponse(response *http.Response) (eR errorResponse) {
	jsonBytes, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	eR = errorResponse{}
	_ = json.Unmarshal(jsonBytes, &eR)
	return
}
