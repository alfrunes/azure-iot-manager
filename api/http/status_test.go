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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	app_mocks "github.com/mendersoftware/azure-iot-manager/app/mocks"
)

func TestAlive(t *testing.T) {
	azureIotManagerApp := &app_mocks.App{}

	router, _ := NewRouter(azureIotManagerApp)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", APIURLInternal+APIURLAlive, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	azureIotManagerApp.AssertExpectations(t)
}

func TestHealth(t *testing.T) {
	testCases := []struct {
		Name           string
		HealthCheckErr error

		HTTPStatus int
		HTTPBody   map[string]interface{}
	}{
		{
			Name:       "ok",
			HTTPStatus: http.StatusNoContent,
		},
		{
			Name:           "ko",
			HealthCheckErr: errors.New("error"),
			HTTPStatus:     http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			azureIotManagerApp := &app_mocks.App{}
			azureIotManagerApp.On("HealthCheck",
				mock.MatchedBy(func(_ context.Context) bool {
					return true
				})).Return(tc.HealthCheckErr)

			router, _ := NewRouter(azureIotManagerApp)
			req, err := http.NewRequest("GET", APIURLInternal+APIURLHealth, nil)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.HTTPStatus, w.Code)
			if tc.HTTPStatus == http.StatusNoContent {
				assert.Nil(t, w.Body.Bytes())
			}

			azureIotManagerApp.AssertExpectations(t)
		})
	}
}
