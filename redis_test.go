// Copyright 2015 tango authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package redistore

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/lunny/tango"
	"github.com/tango-contrib/session"
)

type SessionAction struct {
	session.Session
}

type Test struct {
	Id   int64
	Name string
}

func (action *SessionAction) Get() string {
	var test1 = []string{"1", "2"}
	action.Session.Set("11", test1)
	res := action.Session.Get("11")

	fmt.Println(res.([]string))

	var test2 = map[string]int{"1": 1, "2": 2}
	action.Session.Set("22", test2)
	res2 := action.Session.Get("22")

	fmt.Println(res2.(map[string]int))

	for i := 0; i < 3; i++ {
		var test3 = Test{1, "xlw"}
		action.Session.Set("33", &test3)
		res3 := action.Session.Get("33")

		fmt.Println(res3.(*Test))
	}

	action.Session.Set("test", "1")
	return action.Session.Get("test").(string)
}

func TestSession(t *testing.T) {
	buff := bytes.NewBufferString("")
	recorder := httptest.NewRecorder()
	recorder.Body = buff

	tg := tango.Classic()
	tg.Use(session.New(session.Options{
		Store: New(Options{
			Host:    "127.0.0.1",
			DbIndex: 0,
			MaxAge:  30 * time.Minute,
		}),
	}))
	tg.Get("/", new(SessionAction))

	req, err := http.NewRequest("GET", "http://localhost:8000/", nil)
	if err != nil {
		t.Error(err)
	}

	tg.ServeHTTP(recorder, req)
	expect(t, recorder.Code, http.StatusOK)
	refute(t, len(buff.String()), 0)
	expect(t, buff.String(), "1")
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
