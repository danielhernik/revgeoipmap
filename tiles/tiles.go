package tiles

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"math"
	"net/http"
)

type UrlGen struct {
	baseUrl string
}

func NewUrlGen(baseUrl string) *UrlGen {
	return &UrlGen{baseUrl}
}

const degToRad = math.Pi / 180

func (t UrlGen) coordsToTileNumber(lat float64, lon float64, zoom int) (tileX, tileY int64) {
	latRad := lat * degToRad
	n := math.Pow(2, float64(zoom))
	tileY = int64((1.0 - math.Log(math.Tan(latRad)+(1/math.Cos(latRad)))/math.Pi) / 2.0 * float64(n))
	tileX = int64((lon + 180.0) / 360.0 * float64(n))
	return
}

func (t UrlGen) GetUrl(lat float64, lon float64, zoom int) (url string) {
	tileX, tileY := t.coordsToTileNumber(lat, lon, zoom)
	url = fmt.Sprintf("%s/%d/%d/%d.png", t.baseUrl, zoom, tileX, tileY)
	return
}

func (t UrlGen) getTileFromServer(url string) image.Image {
	webResp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	tile, _, err := image.Decode(webResp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return tile
}

type chanAndPos struct {
	Tile image.Image
	Pos  [2]int
}

var tilesChannel = make(chan chanAndPos)

func (t UrlGen) GetAllSurroundingTiles(lat float64, lon float64, zoom int) image.Image {
	const tileSize int = 255
	const numOfTiles int = 3

	tileX, tileY := t.coordsToTileNumber(lat, lon, zoom)
	for xx := 0; xx < numOfTiles; xx++ {
		for yy := 0; yy < numOfTiles; yy++ {
			url := fmt.Sprintf("%s/%d/%d/%d.png", t.baseUrl, zoom, tileX+int64(xx), tileY+int64(yy))
			go func(xx, yy int) {
				tilesChannel <- chanAndPos{t.getTileFromServer(url), [2]int{xx, yy}}
			}(xx, yy)
		}
	}
	var tilePos chanAndPos
	dest := image.NewRGBA(image.Rect(0, 0, tileSize*numOfTiles, tileSize*numOfTiles))
	for ii := 0; ii < numOfTiles*numOfTiles; ii++ {
		tilePos = <-tilesChannel
		sr := tilePos.Tile.Bounds()
		dstPos := image.Rect(
			tilePos.Pos[0]*tileSize,
			tilePos.Pos[1]*tileSize,
			tilePos.Pos[0]*tileSize+tileSize,
			tilePos.Pos[1]*tileSize+tileSize,
		)
		draw.Draw(dest, dstPos, tilePos.Tile, sr.Min, draw.Src)
	}
	return dest
}
