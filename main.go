package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/abh/geoip"
	"github.com/hoisie/web"
	"log"
	"os"
)

var gi *geoip.GeoIP

type geoResponse struct {
	City      string
	Latitude  float32 `json:"Lat"`
	Longitude float32 `json:"Lon"`
}

/* Note that order here matters - the further entry is on the list,
the bigger it's priority. */

var remoteIPHeaders = [...]string{
	"X-Forwarded",
	"X-Cluster-Client-IP",
	"Forwarded-For",
	"Forwarded",
	"X-Forwarded-For",
	"Client-IP", // This one we trust the most.
}

func reverseGeoIP(ctx *web.Context, val string) string {
	var remoteIP string
	var numForHeader int = 0
	for _, headerName := range remoteIPHeaders {
		numForHeader = len(ctx.Request.Header[headerName])
		if numForHeader > 0 {
			/* We want the last entry, because that's where the
			"real" client IP usually ends up. */
			remoteIP = ctx.Request.Header[headerName][numForHeader-1]
		}
	}
	/* If we haven't found any of the remoteIPHeaders - just use the IP
	of the client. It CAN be the IP of the proxy server etc. */
	if remoteIP == "" {
		remoteIP = ctx.Request.RemoteAddr
	}
	var geoipRecord *geoip.GeoIPRecord = gi.GetRecord(remoteIP)
	/* TODO: add some nice logging tool, like log4go */
	log.Println(geoipRecord)
	var resp geoResponse
	var respStatus int = 200
	if geoipRecord != nil {
		resp = geoResponse{geoipRecord.City,
			geoipRecord.Latitude,
			geoipRecord.Longitude,
		}
	} else {
		resp = geoResponse{"", 0.0, 0.0}
		respStatus = 404
	}
	ctx.SetHeader("Content-Type", "application/javascript", true)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	ctx.WriteHeader(respStatus)
	return string(respJSON)
}

func main() {
	var geoipDBPath = flag.String("f", "./GeoIPCity.dat", "Path to GeoIPCity.dat file")
	var listenIP = flag.String("i", "127.0.0.1", "IP to listen on")
	var listenPort = flag.Int("p", 9999, "Port to listen on")
	flag.Parse()

	var err error
	gi, err = geoip.Open(*geoipDBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open GeoIP database\n")
		os.Exit(3)
	}

	web.Get("/(.*)", reverseGeoIP)
	web.Run(fmt.Sprintf("%s:%d", *listenIP, *listenPort))
}
