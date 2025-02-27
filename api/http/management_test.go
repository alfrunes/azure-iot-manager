// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"

	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/rest.utils"

	mapp "github.com/mendersoftware/azure-iot-manager/app/mocks"
	"github.com/mendersoftware/azure-iot-manager/model"
)

var contextMatcher = mock.MatchedBy(func(_ context.Context) bool { return true })

func compareParameterValues(t *testing.T, expected interface{}) interface{} {
	return mock.MatchedBy(func(actual interface{}) bool {
		return assert.EqualValues(t, expected, actual)
	})
}

func GenerateJWT(id identity.Identity) string {
	JWT := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"HS256","typ":"JWT"}`),
	)
	b, _ := json.Marshal(id)
	JWT = JWT + "." + base64.RawURLEncoding.EncodeToString(b)
	hash := hmac.New(sha256.New, []byte("hmac-sha256-secret"))
	JWT = JWT + "." + base64.RawURLEncoding.EncodeToString(
		hash.Sum([]byte(JWT)),
	)
	return JWT
}

func TestGetSettings(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Headers http.Header

		App func(t *testing.T) *mapp.App

		StatusCode int
		Response   interface{}
	}{
		{
			Name: "ok",

			Headers: http.Header{
				"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
					IsUser:  true,
					Subject: "829cbefb-70e7-438f-9ac5-35fd131c2111",
					Tenant:  "123456789012345678901234",
				})},
			},

			App: func(t *testing.T) *mapp.App {
				app := new(mapp.App)
				app.On("GetSettings",
					contextMatcher).
					Return(model.Settings{ConnectionString: "my://connection.string"}, nil)
				return app
			},

			StatusCode: http.StatusOK,
			Response: model.Settings{
				ConnectionString: "my://connection.string",
			},
		},
		{
			Name: "ok empty settings",

			Headers: http.Header{
				"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
					IsUser:  true,
					Subject: "829cbefb-70e7-438f-9ac5-35fd131c2111",
					Tenant:  "123456789012345678901234",
				})},
			},

			App: func(t *testing.T) *mapp.App {
				app := new(mapp.App)
				app.On("GetSettings",
					contextMatcher).
					Return(model.Settings{}, nil)
				return app
			},

			StatusCode: http.StatusOK,
			Response:   model.Settings{},
		},
		{
			Name: "error, invalid authorization header",

			Headers: http.Header{
				textproto.CanonicalMIMEHeaderKey(requestid.RequestIdHeader): []string{
					"829cbefb-70e7-438f-9ac5-35fd131c2111",
				},
				"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
					IsDevice: true,
					Subject:  "829cbefb-70e7-438f-9ac5-35fd131c2f76",
					Tenant:   "123456789012345678901234",
				})},
			},
			StatusCode: http.StatusForbidden,
			Response: rest.Error{
				Err:       ErrMissingUserAuthentication.Error(),
				RequestID: "829cbefb-70e7-438f-9ac5-35fd131c2111",
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var testApp *mapp.App
			if tc.App == nil {
				testApp = new(mapp.App)
			} else {
				testApp = tc.App(t)
			}
			defer testApp.AssertExpectations(t)
			handler, _ := NewRouter(testApp)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET",
				"http://localhost"+
					APIURLManagement+
					APIURLSettings,
				nil,
			)
			for key := range tc.Headers {
				req.Header.Set(key, tc.Headers.Get(key))
			}

			handler.ServeHTTP(w, req)
			assert.Equal(t, tc.StatusCode, w.Code, "invalid HTTP status code")
			b, _ := json.Marshal(tc.Response)
			assert.JSONEq(t, string(b), w.Body.String())
		})
	}
}

func TestSetSettings(t *testing.T) {
	t.Parallel()
	jitter768 := ""
	for i := 0; i < 2049; i++ {
		jitter768 += "1"
	}
	testCases := []struct {
		Name string

		RequestBody interface{}
		RequestHdrs http.Header

		App func(t *testing.T) *mapp.App

		RspCode int
		Error   error
	}{{
		Name: "ok",

		RequestBody: map[string]string{
			"connection_string": "my://connection.string",
		},
		RequestHdrs: http.Header{
			"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
				Subject: uuid.NewString(),
				Tenant:  "123456789012345678901234",
				IsUser:  true,
			})},
		},

		App: func(t *testing.T) *mapp.App {
			a := new(mapp.App)
			a.On("SetSettings", contextMatcher, mock.AnythingOfType("model.Settings")).
				Return(nil)
			return a
		},

		RspCode: http.StatusNoContent,
	}, {
		Name: "internal error",

		RequestBody: map[string]string{
			"connection_string": "my://connection.string",
		},
		RequestHdrs: http.Header{
			"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
				Subject: uuid.NewString(),
				Tenant:  "123456789012345678901234",
				IsUser:  true,
			})},
		},

		App: func(t *testing.T) *mapp.App {
			a := new(mapp.App)
			a.On("SetSettings", contextMatcher, mock.AnythingOfType("model.Settings")).
				Return(errors.New("internal error"))
			return a
		},

		RspCode: http.StatusInternalServerError,
		Error:   errors.New(http.StatusText(http.StatusInternalServerError)),
	}, {
		Name: "settings string too long",

		RequestBody: map[string]string{
			"connection_string": "my://long.connections.string" + jitter768,
		},
		RequestHdrs: http.Header{
			"Authorization": []string{"Bearer " + GenerateJWT(identity.Identity{
				Subject: uuid.NewString(),
				Tenant:  "123456789012345678901234",
				IsUser:  true,
			})},
		},

		App: func(t *testing.T) *mapp.App { return new(mapp.App) },

		RspCode: http.StatusBadRequest,
		Error:   errors.New("malformed request body"),
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			app := tc.App(t)
			defer app.AssertExpectations(t)
			var body io.Reader
			if tc.RequestBody != nil {
				b, _ := json.Marshal(tc.RequestBody)
				body = bytes.NewReader(b)
			}
			req, _ := http.NewRequest("PUT",
				"http://localhost"+APIURLManagement+APIURLSettings,
				body,
			)
			for k, v := range tc.RequestHdrs {
				req.Header[k] = v
			}

			router, _ := NewRouter(app)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.RspCode, w.Code)
			if tc.Error != nil {
				var erro rest.Error
				if assert.NotNil(t, w.Body) {
					err := json.Unmarshal(w.Body.Bytes(), &erro)
					require.NoError(t, err)
					assert.Regexp(t, tc.Error.Error(), erro.Error())
				}
			} else {
				assert.Empty(t, w.Body.Bytes(), string(w.Body.Bytes()))
			}
		})
	}
}
