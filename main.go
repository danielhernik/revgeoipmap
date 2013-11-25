package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/abh/geoip"
	"github.com/hoisie/web"
	"image/png"
	"log"
	"operarevgeoip/tiles"
	"os"
	"strconv"
)

var gi *geoip.GeoIP
var tileUrlGenerator *tiles.UrlGen

type geoResponse struct {
	City      string
	Latitude  float32 `json:"Lat"`
	Longitude float32 `json:"Lon"`
	TileUrl   string
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

func getRealIP(ctx *web.Context) (remoteIP string) {
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
	return
}

func reverseGeoIP(ctx *web.Context, val string) string {
	remoteIP := getRealIP(ctx)
	var geoipRecord *geoip.GeoIPRecord = gi.GetRecord(remoteIP)
	/* TODO: add some nice logging tool, like log4go */
	log.Println(geoipRecord)
	var resp geoResponse
	var respStatus int = 200
	if geoipRecord != nil {
		resp = geoResponse{geoipRecord.City,
			geoipRecord.Latitude,
			geoipRecord.Longitude,
			tileUrlGenerator.GetUrl(float64(geoipRecord.Latitude), float64(geoipRecord.Longitude), 8),
		}
	} else {
		resp = geoResponse{"", 0.0, 0.0, ""}
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

func getBigTile(ctx *web.Context, val string) {
	zoom, err := strconv.Atoi(val)
	if err != nil {
		zoom = 8 /* Some sensible fallback value */
		log.Println(err)
	}
	remoteIP := getRealIP(ctx)
	var geoipRecord *geoip.GeoIPRecord = gi.GetRecord(remoteIP)
	if geoipRecord == nil {
		ctx.WriteHeader(404)
		ctx.WriteString(fmt.Sprintf("GeoIP record for given IP (%s) not found", remoteIP))
		return
	}
	tilefile := tileUrlGenerator.GetAllSurroundingTiles(float64(geoipRecord.Latitude), float64(geoipRecord.Longitude), zoom)
	ctx.SetHeader("Content-Type", "image/png", true)
	png.Encode(ctx, tilefile)
}

func main() {
	var geoipDBPath = flag.String("f", "./GeoIPCity.dat", "Path to GeoIPCity.dat file")
	var listenIP = flag.String("i", "127.0.0.1", "IP to listen on")
	var listenPort = flag.Int("p", 9999, "Port to listen on")
	var enableSSL = flag.Bool("s", true, "Enable SSL (you have to supply cert and key files)")
	var cert = flag.String("c", "cert.crt", "Certificate file")
	var certKey = flag.String("k", "cert.key", "Certificate file")
	var tileServerBaseURL = flag.String("t", "http://bizon.opera.com:1069/osm_tiles", "Base path of tiles server")
	flag.Parse()

	var err error
	gi, err = geoip.Open(*geoipDBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open GeoIP database\n")
		os.Exit(3)
	}
	tileUrlGenerator = tiles.NewUrlGen(*tileServerBaseURL)

	web.Get("/image/(.*)", getBigTile)
	web.Get("/(.*)", reverseGeoIP)
	if !*enableSSL {
		log.Println("Running as http")
		web.Run(fmt.Sprintf("%s:%d", *listenIP, *listenPort))
	}
	certAndKey, err := tls.LoadX509KeyPair(*cert, *certKey)
	if err != nil {
		log.Println("Error loading cert or key")
		log.Println(err)
		return
	}
	tlsConfig := tls.Config{Certificates: []tls.Certificate{certAndKey}}
	log.Println("Cert file supplied - running as HTTPS")
	web.RunTLS(fmt.Sprintf("%s:%d", *listenIP, *listenPort), &tlsConfig)
}
