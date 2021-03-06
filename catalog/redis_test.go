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
	"log"
	"os"
	"testing"
)

// TestRedisClient tests the Redis connection
func TestRedisClient(t *testing.T) {
	SetMockConnCount(0)
	outputs := []string{}
	client = MakeMockRedisCli(outputs)

	os.Setenv("VCAP_SERVICES", "{\"p-redis\":[{\"credentials\":{\"host\":\"127.0.0.1\",\"port\":6379}}]}")
	os.Setenv("PL_API_KEY", "a1fa3d8df30545468052e45ae9e4520e")
	vcapServicesStr := os.Getenv("VCAP_SERVICES")
	log.Printf("VCAP_SERVICES: %v", vcapServicesStr)
	var VCapServHolder VcapServices
	var VCapRedHolder []VcapRedis

	VCapServHolder.Redis = VCapRedHolder
	VCapServHolder.RedisOptions()
	SetKey("Rubber", "Duckies")
	_, _ = GetKey("Duckies")
	_, _ = GetKey("Rubber")

}
