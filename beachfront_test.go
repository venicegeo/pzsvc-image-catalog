// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"gopkg.in/redis.v3"
)

func TestBeachfront(t *testing.T) {
	var (
		err       error
		idmBytes  []byte
		red       *redis.Client
		idm, idID string
		status    *redis.StatusCmd
	)
	setName := "test_images"
	properties := make(map[string]interface{})
	properties["name"] = "Whatever"

	imageDescriptor := geojson.NewFeature(nil, "12345", properties)

	if red, err = catalog.RedisClient(); red == nil {
		t.Fatalf("Failed to create Redis client: %v", err.Error())
	}
	defer red.Close()

	if idmBytes, err = json.Marshal(imageDescriptor); err != nil {
		t.Error(err)
	}
	idm = string(idmBytes)
	idID = "test" + imageDescriptor.ID
	log.Printf("Setting %v to %v", idID, idm)
	status = red.Set(idID, idm, 0)
	if _, err = status.Result(); err != nil {
		t.Error(err.Error())
	}
	log.Printf("Setting %v to %v", idID, idm)
	intCmd := red.SAdd(setName, idID)
	if _, err = intCmd.Result(); err != nil {
		t.Error(err.Error())
	}

	images, _ := catalog.GetImages(setName, nil)

	t.Logf("%#v", images)
	if len(images.Images.Features) < 1 {
		t.Error("Where are the images?")
	}
	for _, curr := range images.Images.Features {
		t.Logf("%v", curr)
	}
	red.Del(setName)
}
