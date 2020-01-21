package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

func TestClientListClones(t *testing.T) {
	expectedClones := []*models.Clone{{
		ID:          "testCloneID",
		Name:        "mockClone",
		CloneSize:   450,
		CloningTime: 1,
		Protected:   true,
		DeleteAt:    "2020-01-10 00:00:05.000 UTC",
		CreatedAt:   "2020-01-10 00:00:00.000 UTC",
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		Db: &models.Database{
			Username: "john",
			Password: "doe",
		},
		Project: "testProject",
	}}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/status")

		// Prepare response.
		body, err := json.Marshal(models.InstanceStatus{Clones: expectedClones})
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(body)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	cloneList, err := c.ListClones(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, expectedClones, cloneList)
}

func TestClientListClonesWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	cloneList, err := c.ListClones(context.Background())
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, cloneList)
}

func TestClientCreateClone(t *testing.T) {
	expectedClone := &models.Clone{
		ID:          "testCloneID",
		Name:        "mockClone",
		CloneSize:   450,
		CloningTime: 1,
		Protected:   true,
		DeleteAt:    "2020-01-10 00:00:05.000 UTC",
		CreatedAt:   "2020-01-10 00:00:00.000 UTC",
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		Db: &models.Database{
			Username: "john",
			Password: "doe",
		},
		Project: "testProject",
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone")

		requestBody, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		cloneRequest := CreateRequest{}
		err = json.Unmarshal(requestBody, &cloneRequest)
		require.NoError(t, err)

		// Prepare response.
		responseBody, err := json.Marshal(expectedClone)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	newClone, err := c.CreateClone(context.Background(), CreateRequest{
		Name:      "mockClone",
		Project:   "testProject",
		Protected: true,
		DB: &DatabaseRequest{
			Username: "john",
			Password: "doe",
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, expectedClone, newClone)
}

func TestClientCreateCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.CreateClone(context.Background(), CreateRequest{})
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, clone)
}

func TestClientGetClone(t *testing.T) {
	expectedClone := &models.Clone{
		ID:          "testCloneID",
		Name:        "mockClone",
		CloneSize:   450,
		CloningTime: 1,
		Protected:   true,
		DeleteAt:    "2020-01-10 00:00:05.000 UTC",
		CreatedAt:   "2020-01-10 00:00:00.000 UTC",
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		Db: &models.Database{
			Username: "john",
			Password: "doe",
		},
		Project: "testProject",
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

		// Prepare response.
		responseBody, err := json.Marshal(expectedClone)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	clone, err := c.GetClone(context.Background(), expectedClone.ID)
	require.NoError(t, err)

	assert.EqualValues(t, expectedClone, clone)
}

func TestClientGetCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte{})),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.GetClone(context.Background(), "cloneID")
	require.EqualError(t, err, "failed to decode a response body: EOF")
	require.Nil(t, clone)
}

func TestClientUpdateClone(t *testing.T) {
	cloneModel := &models.Clone{
		ID:          "testCloneID",
		Name:        "mockClone",
		CloneSize:   450,
		CloningTime: 1,
		Protected:   true,
		DeleteAt:    "2020-01-10 00:00:05.000 UTC",
		CreatedAt:   "2020-01-10 00:00:00.000 UTC",
		Status: &models.Status{
			Code:    "OK",
			Message: "Instance is ready",
		},
		Db: &models.Database{
			Username: "john",
			Password: "doe",
		},
		Project: "testProject",
	}

	mockClient := NewTestClient(func(r *http.Request) *http.Response {
		assert.Equal(t, r.URL.String(), "https://example.com/clone/testCloneID")

		requestBody, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		defer func() { _ = r.Body.Close() }()

		updateRequest := UpdateRequest{}
		err = json.Unmarshal(requestBody, &updateRequest)
		require.NoError(t, err)

		cloneModel.Name = updateRequest.Name
		cloneModel.Protected = updateRequest.Protected

		// Prepare response.
		responseBody, err := json.Marshal(cloneModel)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	newClone, err := c.UpdateClone(context.Background(), cloneModel.ID, UpdateRequest{
		Name:      "UpdatedName",
		Protected: false,
	})
	require.NoError(t, err)

	assert.EqualValues(t, cloneModel, newClone)
}

func TestClientUpdateCloneWithFailedRequest(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		errorBadRequest := models.Error{
			Code:    "BAD_REQUEST",
			Message: "Wrong request format.",
			Detail:  "Clone not found.",
			Hint:    "Check request params.",
		}

		responseBody, err := json.Marshal(errorBadRequest)
		require.NoError(t, err)

		return &http.Response{

			StatusCode: 400,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	clone, err := c.UpdateClone(context.Background(), "testCloneID", UpdateRequest{})
	require.EqualError(t, err, `failed to get response: Code "BAD_REQUEST". Message: Wrong request format. Detail: Clone not found. Hint: Check request params.`)
	require.Nil(t, clone)
}

func TestClientDestroyClone(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID")

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.DestroyClone(context.Background(), "testCloneID")
	require.NoError(t, err)
}

func TestClientDestroyCloneWithFailedRequest(t *testing.T) {
	errorNotFound := models.Error{
		Code:    "NOT_FOUND",
		Message: "Not found.",
		Detail:  "Requested object does not exist.",
		Hint:    "Specify your request.",
	}
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID")

		responseBody, err := json.Marshal(errorNotFound)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.DestroyClone(context.Background(), "testCloneID")
	assert.EqualError(t, err, `failed to get response: Code "NOT_FOUND". Message: Not found. Detail: Requested object does not exist. Hint: Specify your request.`)
}

func TestClientResetClone(t *testing.T) {
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID/reset")

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.ResetClone(context.Background(), "testCloneID")
	require.NoError(t, err)
}

func TestClientResetCloneWithFailedRequest(t *testing.T) {
	errorUnauthorized := models.Error{
		Code:    "UNAUTHORIZED",
		Message: "Unauthorized.",
		Detail:  "Invalid token.",
		Hint:    "Check your verification token.",
	}
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), "https://example.com/clone/testCloneID/reset")

		responseBody, err := json.Marshal(errorUnauthorized)
		require.NoError(t, err)

		return &http.Response{
			StatusCode: 401,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			Header:     make(http.Header),
		}
	})

	logger, _ := test.NewNullLogger()
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "token",
	}, logger)
	require.NoError(t, err)

	c.client = mockClient

	// Send a request.
	err = c.ResetClone(context.Background(), "testCloneID")
	assert.EqualError(t, err, `failed to get response: Code "UNAUTHORIZED". Message: Unauthorized. Detail: Invalid token. Hint: Check your verification token.`)
}
