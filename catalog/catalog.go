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

package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"gopkg.in/redis.v3"

	"github.com/venicegeo/geojson-go/geojson"
)

var imageCatalogPrefix string

// SetImageCatalogPrefix sets the prefix for this instance
// when it is necessary to override the default
func SetImageCatalogPrefix(prefix string) {
	imageCatalogPrefix = prefix
}

// ImageDescriptors is the response to a Discover query
type ImageDescriptors struct {
	Count      int                        `json:"count"`
	StartIndex int                        `json:"startIndex"`
	Images     *geojson.FeatureCollection `json:"images"`
}

// IndexSize returns the size of the index
func IndexSize() int64 {
	rc, _ := RedisClient()
	result := rc.ZCard(imageCatalogPrefix)
	return result.Val()
}

// GetImages returns images for the given set matching the criteria in the options
func GetImages(options *geojson.Feature, start int64, end int64) (ImageDescriptors, string) {

	var (
		result     ImageDescriptors
		resultText string
		results    *redis.StringSliceCmd
		zrbs       redis.ZRangeByScore
		ssc        *redis.StringSliceCmd
		fc         *geojson.FeatureCollection
		features   []*geojson.Feature
	)
	complete := false
	go PopulateIndex(options)
	index := GetDiscoverIndexName(options)
	red, _ := RedisClient()
	for !complete {
		card := red.ZCard(index)
		if card.Val() > end {
			complete = true
		} else {
			zrbs.Min = "1"
			zrbs.Max = "1"
			ssc = red.ZRangeByScore(index, zrbs)
			if len(ssc.Val()) > 0 {
				complete = true
			}
		}
		if complete {
			results = red.ZRange(index, start, end)
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	for _, curr := range results.Val() {
		var (
			cid      *geojson.Feature
			idString string
		)
		idString = red.Get(curr).Val()
		cid, _ = geojson.FeatureFromBytes([]byte(idString))
		features = append(features, cid)
	}
	result.Count = len(features)
	fc = geojson.NewFeatureCollection(features)
	result.Images = fc
	bytes, _ := json.Marshal(result)
	resultText = string(bytes)
	return result, resultText
}

// GetDiscoverIndexName returns the name of the index corresponding
// to the search criteria provided
func GetDiscoverIndexName(options *geojson.Feature) string {
	bytes, _ := json.Marshal(options)
	// TODO: hash this index name
	return imageCatalogPrefix + string(bytes)
}

// PopulateIndex populates an index corresponding
// to the search criteria provided
func PopulateIndex(options *geojson.Feature) {
	red, _ := RedisClient()
	var (
		cid      *geojson.Feature
		idString string
		err      error
		z        redis.Z
	)
	index := GetDiscoverIndexName(options)

	if indexExists := client.Exists(index); !indexExists.Val() {
		members := client.ZRange(imageCatalogPrefix, 0, -1)
		for _, curr := range members.Val() {
			idString = red.Get(curr).Val()
			if cid, err = geojson.FeatureFromBytes([]byte(idString)); err == nil {
				if passImageDescriptor(cid, options) {
					z.Member = curr
					z.Score = red.ZScore(imageCatalogPrefix, curr).Val()
					red.ZAdd(index, z)
				}
			}
		}
		// Stick a terminal entry in the index so we know it is done
		// This is the only one with a positive score
		z.Member = ""
		z.Score = 1
		red.ZAdd(index, z)
		duration, _ := time.ParseDuration("24h")
		client.Expire(index, duration)
	}
}

// pass returns true if the receiving object complies
// with all of the properties in the input
func passImageDescriptor(id, test *geojson.Feature) bool {
	if test == nil {
		return true
	}
	testCloudCover := test.PropertyFloat("cloudCover")
	idCloudCover := id.PropertyFloat("cloudCover")
	if testCloudCover != 0 && idCloudCover != 0 && !math.IsNaN(testCloudCover) && !math.IsNaN(idCloudCover) {
		if idCloudCover > testCloudCover {
			return false
		}
	}

	testBitDepth := test.PropertyInt("bitDepth")
	idBitDepth := id.PropertyInt("bitDepth")
	if testBitDepth != 0 && idBitDepth != 0 && (idBitDepth < testBitDepth) {
		return false
	}

	testBeachfrontScore := test.PropertyFloat("beachfrontScore")
	idBeachfrontScore := id.PropertyFloat("beachfrontScore")
	if testBeachfrontScore != 0 && idBeachfrontScore != 0 && !math.IsNaN(testBeachfrontScore) && !math.IsNaN(idBeachfrontScore) {
		if idBeachfrontScore < testBeachfrontScore {
			return false
		}
	}

	testAcquiredDate := test.PropertyString("acquiredDate")
	idAcquiredDate := id.PropertyString("acquiredDate")
	if testAcquiredDate != "" {
		var (
			idTime, testTime time.Time
			err              error
		)
		if idTime, err = time.Parse(time.RFC3339, idAcquiredDate); err == nil {
			if testTime, err = time.Parse(time.RFC3339, testAcquiredDate); err == nil {
				if idTime.Before(testTime) {
					return false
				}
			}
		}
	}

	testBands := test.PropertyStringSlice("bands")
	if len(testBands) > 0 {
		if idBandsIfc, ok := id.Properties["bands"]; ok {
			idBands := idBandsIfc.(map[string]interface{})
			for _, testBand := range testBands {
				if _, ok = idBands[testBand]; !ok {
					return false
				}
			}
		}
	}

	if (len(test.Bbox) > 0) && !test.Bbox.Overlaps(id.Bbox) {
		return false
	}
	return true
}

// GetImageMetadata returns the image metadata as a GeoJSON feature
func GetImageMetadata(id string) (*geojson.Feature, error) {
	rc, _ := RedisClient()
	var stringCmd *redis.StringCmd
	key := imageCatalogPrefix + ":" + id
	if stringCmd = rc.Get(key); stringCmd.Err() != nil {
		return nil, stringCmd.Err()
	}
	metadataString := stringCmd.Val()
	return geojson.FeatureFromBytes([]byte(metadataString))
}

// HTTPError represents any HTTP error
type HTTPError struct {
	Status  int
	Message string
}

func (err HTTPError) Error() string {
	return fmt.Sprintf("%d: %v", err.Status, err.Message)
}

// StoreFeature stores a feature into the catalog
// using a key based on the feature's ID
func StoreFeature(feature *geojson.Feature, score float64) {
	rc, _ := RedisClient()
	bytes, _ := json.Marshal(feature)
	key := imageCatalogPrefix + ":" + feature.ID
	rc.Set(key, string(bytes), 0)
	z := redis.Z{Score: score, Member: key}
	rc.ZAdd(imageCatalogPrefix, z)
}

// DropIndex drops the main index containing all known catalog entries
func DropIndex() {
	rc, _ := RedisClient()
	rc.Del(imageCatalogPrefix)
}

// ImageIOReader returns an io Reader for the requested band
func ImageIOReader(id, band string) (io.Reader, error) {
	var (
		feature *geojson.Feature
		err     error
	)
	if feature, err = GetImageMetadata(id); err != nil {
		return nil, err
	}
	return ImageFeatureIOReader(feature, band)
}

// ImageFeatureIOReader returns an io Reader for the requested band
func ImageFeatureIOReader(feature *geojson.Feature, band string) (io.Reader, error) {
	// This will work for Landsat but others will require additional code
	var (
		result    io.Reader
		err       error
		bandsMap  map[string]interface{}
		urlIfc    interface{}
		urlString string
		ok        bool
		response  *http.Response
	)
	if bandsMap, ok = feature.Properties["bands"].(map[string]interface{}); ok {
		if urlIfc, ok = bandsMap[band]; ok {
			if urlString, ok = urlIfc.(string); ok {
				if response, err = http.DefaultClient.Get(urlString); err != nil {
					return nil, err
				}
				result = response.Body
			}
		}
	}
	if ok {
		return result, nil
	}
	return nil, fmt.Errorf("Requested band \"%v\" not found in image %v.", band, feature.ID)
}
