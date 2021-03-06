#pzsvc-image-catalog
Nascent catalog for Beachfront et. al.

Currently relies on Redis being installed and running locally at the default location.

Now relies on pz-workflow so that it can trigger events when new images are detected.

Now relies on GEOS, and therefore has its own buildpack.


# Startup procedures
replace localhost:8080 to wherever this application is deployed (e.g., http://pzsvc-image-catalog.stage.geointservices.io/)

## Harvesting Planet Labs
POST http://localhost:8080/planet with the following parameters:
* Content-Type=application/json
* Body
   * PL_API_KEY=...
   * pzGateway=http://pz-gateway.stage.geointservices.io
   * reharvest
      * true: ignore the presence of previous entries (a good idea for fresh harvests since Planet Labs seems to have some duplicates)
      * false: stop when reaching an entry that already exists (better for [subsequent harvests](#subsequent-harvests)
   * filter
      * whitelist
      * blacklist
   * cap=[int] caps the size of the index at approximately that amount (for testing only)
   * requestPageSize: number of scenes harvested at a time (default: 1000)
* Provide auth information for the Piazza Gateway in the header - you must authenticate for this process to work.

### Filter Descriptors
* geojson=a valid GeoJSON block

*OR*

* wfsurl = something like `http://gsn-geose-loadbala-17usyyb36bfdl-1788485819.us-east-1.elb.amazonaws.com/geoserver/piazza/wfs`
* featureType: the name of the WFS layer, something like `46a50997-709e-40f7-9abc-9438da773a72` 

### Example
```
{  
   "pzGateway":"http://piazza.stage.geointservices.io",
   "PL_API_KEY":...,
   "cap":10000,
   "reharvest":true,
   "filter":{  
      "whitelist":{  
         "geojson":{  
            "type":"FeatureCollection",
            "crs":{  
               "type":"name",
               "properties":{  
                  "name":"urn:ogc:def:crs:OGC:1.3:CRS84"
               }
            },
            "features":[  
               {  
                  "type":"Feature",
                  "properties":{  
                     "id":null
                  },
                  "geometry":{  
                     "type":"Polygon",
                     "coordinates":[  
                        [  
                           [  
                              0,
                              90
                           ],
                           [  
                              180,
                              90
                           ],
                           [  
                              180,
                              -90
                           ],
                           [  
                              -180,
                              -90
                           ],
                           [  
                              0,
                              90
                           ]
                        ]
                     ]
                  }
               }
           ]
         }
      }
   }
}
```

### Example
```
{  
   "pzGateway":"http://piazza.stage.geointservices.io",
   "reharvest":true,
   "filter":{  
      "blacklist":{  
        "wfsurl":"http://gsp-geose-LoadBala-4EP8UFUE9SXL-919040015.us-east-1.elb.amazonaws.com:80/geoserver/piazza/wfs",
        "featureType":"8e31e022-4e1f-4a32-b341-4eb019ab45bc"
      },
      "whitelist":{  
        "wfsurl":"http://gsp-geose-LoadBala-4EP8UFUE9SXL-919040015.us-east-1.elb.amazonaws.com:80/geoserver/piazza/wfs",
        "featureType":"cb76fb5e-bd7d-44f7-bc03-22fdefcdf68e"
      }
   }
}
```

## Clearing out harvested data
GET http://localhost:8080/dropIndex
* Provide auth information for the Piazza Gateway in the header - you must authenticate for this process to work.

## Testing Discovery
Call http://localhost:8080/discover with one or more of the following:
* bbox = x1,y1,x2,y2
* acquiredDate (RFC 3339)
* cloudCover (0 to 100)
* Example: http://localhost:8080/discover?bbox=-120,-60,-90,-10&acquiredDate=2016-09-01T00:00:00Z

## Subsequent harvests
Use the same endpoint as the initial harvest
* event=true (optional) (this causes the catalog to post a Piazza event each time a new scene is harvested. This is not recommended for the initial harvest, but may be done in subsequent harvests when the number of harvested scenes is lower)
  
## Setting up recurring harvests
Call the harvest operation as per [Subsequent harvests](#subsequent-harvests) with one additional parameter:
* recurring=true

When this is done, the image catalog will set up the following:
* service 
* event type
* event (with cron of something like every 1h)
* trigger to call the service when the event fires

When this is working right, the following will occur:
* HTTP response contains the event ID and trigger ID (plain text currently)
* harvest operations kicked off in image catalog (see PCF logs for evidence)
* events fired in Piazza (event name is something like `beachfront:harvest:new-image-harvested:0`) for each newly harvested scene 

## Finding the right Event Type ID
There is no way to search events by Event Type Name at this time. You need to resolve to an Event Type ID. Once you get this ID, you can call the `/event` endpoint on the gateway with `?eventTypeId=...`
* Call http://localhost:8080/eventTypeID
* pzGateway=http://pz-gateway.stage.geointservices.io



