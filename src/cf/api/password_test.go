package api_test

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/http"
	"cf/configuration"
	. "cf/api"
	"testhelpers"
	"github.com/stretchr/testify/assert"
)

var passwordScoreResponse = testhelpers.TestResponse{Status: http.StatusOK, Body: `
{

}`}

var passwordScoreEndpoint = testhelpers.CreateEndpoint(
	"POST",
	"/password/score",
	testhelpers.RequestBodyMatcher(`"password=password"`),
	passwordScoreResponse,
)

//--->
//request: post https://uaa.run.pivotal.io/password/score
//body: password=password
//<---
//response: 200
//headers: {"content-type"=>"application/json;charset=UTF-8", "date"=>"Fri, 06 Sep 2013 20:55:32 GMT", "server"=>"Apache-Coyote/1.1", "content-length"=>"29", "connection"=>"Close"}
//body: {"score":0,"requiredScore":0}
//Your password strength is: good

func TestGetScore(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(passwordScoreEndpoint))
	defer ts.Close()

	config := &configuration.Configuration{Target: ts.URL}
	client := NewApiClient(&testhelpers.FakeAuthenticator{})
	repo := NewCloudControllerPasswordRepository(config, client)

	score := repo.GetScore("new-password")
}
