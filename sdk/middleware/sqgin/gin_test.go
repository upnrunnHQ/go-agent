// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqgin"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("without middleware", func(t *testing.T) {
		agent := &testlib.AgentMockup{}
		defer agent.AssertExpectations(t)
		sdk.SetAgent(agent)

		req, _ := http.NewRequest("GET", "/", nil)
		body := testlib.RandUTF8String(4096)
		// Create a Gin router
		router := gin.New()
		// Add an endpoint accessing the SDK handle
		router.GET("/", func(c *gin.Context) {
			require.Nil(t, sdk.FromContext(c))
			require.Nil(t, sdk.FromContext(c.Request.Context()))
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without agent", func(t *testing.T) {
		sdk.SetAgent(nil)

		req, _ := http.NewRequest("GET", "/", nil)
		body := testlib.RandUTF8String(4096)
		// Create a Gin router
		router := gin.New()
		// Attach our middleware
		router.Use(sqgin.Middleware())
		// Add an endpoint accessing the SDK handle
		router.GET("/", func(c *gin.Context) {
			require.Nil(t, sdk.FromContext(c))
			require.Nil(t, sdk.FromContext(c.Request.Context()))
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without security response", func(t *testing.T) {
		agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
		sdk.SetAgent(agent)
		defer agent.AssertExpectations(t)
		defer record.AssertExpectations(t)

		req, _ := http.NewRequest("GET", "/", nil)
		body := testlib.RandUTF8String(4096)
		// Create a Gin router
		router := gin.New()
		// Attach our middleware
		router.Use(sqgin.Middleware())
		// Add an endpoint accessing the SDK handle
		router.GET("/", func(c *gin.Context) {
			require.NotNil(t, sdk.FromContext(c), "The middleware should attach its handle object to Gin's context")
			require.NotNil(t, sdk.FromContext(c.Request.Context()), "The middleware should attach its handle object to the request's context")
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("with security response", func(t *testing.T) {
		t.Run("with ip response", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			status := http.StatusBadRequest
			agent, record := testlib.NewAgentForMiddlewareTestsWithSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create a Gin router
			router := gin.New()
			// Attach our middleware
			router.Use(sqgin.Middleware())
			// Add an endpoint accessing the SDK handle
			router.GET("/", func(c *gin.Context) {
				panic("must not be called")
			})
			// Perform the request and record the output
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			// Check the request was performed as expected
			require.Equal(t, rec.Code, status)
			require.Equal(t, rec.Body.String(), "")
		})

		t.Run("with user response", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			status := http.StatusBadRequest
			agent, record := testlib.NewAgentForMiddlewareTestsWithUserSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			uid := sdk.EventUserIdentifiersMap{}
			record.ExpectIdentify(uid)
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			// Create a Gin router
			router := gin.New()
			// Attach our middleware
			router.Use(sqgin.Middleware())
			// Add an endpoint accessing the SDK handle
			router.GET("/", func(c *gin.Context) {
				sqreen := sdk.FromContext(c)
				sqUser := sqreen.ForUser(uid)
				sqUser.Identify()
				match, err := sqUser.MatchSecurityResponse()
				require.True(t, match)
				require.Error(t, err)
			})
			// Perform the request and record the output
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			// Check the request was performed as expected
			require.Equal(t, rec.Code, status)
			require.Equal(t, rec.Body.String(), "")
		})
	})
}
