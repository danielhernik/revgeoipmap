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

func (t UrlGen) coordsToTileNumber(lat float64, lon float64, zoom int) (tileX, tileY int) {
	latRad := lat * degToRad
	n := math.Pow(2, float64(zoom))
	tileY = int((1.0 - math.Log(math.Tan(lat)+(1/math.Cos(latRad)))/math.Pi) / 2.0 * float64(n))
	tileX = int((lon + 180.0) / 360.0 * float64(n))
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
			url := fmt.Sprintf("%s/%d/%d/%d.png", t.baseUrl, zoom, tileX+xx, tileY+yy)
			go func(xx, yy int) {
				tilesChannel <- chanAndPos{t.getTileFromServer(url), [2]int{xx, yy}}
				log.Println(xx, yy, "Done")
			}(xx, yy)
		}
	}
	var tilePos chanAndPos
	dest := image.NewRGBA(image.Rect(0, 0, tileSize*numOfTiles, tileSize*numOfTiles))
	for ii := 0; ii < numOfTiles*numOfTiles; ii++ {
		tilePos = <-tilesChannel
		log.Println(tilePos)
		sr := tilePos.Tile.Bounds()
		dstPos := image.Rect(
			tilePos.Pos[0]*tileSize,
			tilePos.Pos[1]*tileSize,
			tilePos.Pos[0]*tileSize+tileSize,
			tilePos.Pos[1]*tileSize+tileSize,
		)
		log.Println(dstPos)
		draw.Draw(dest, dstPos, tilePos.Tile, sr.Min, draw.Src)
	}
	return dest
}

/*
func main() {
	m, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	dest := image.NewRGBA(image.Rect(0, 0, 512, 512))
	sr := m.Bounds()
	draw.Draw(dest, image.Rect(0, 0, 255, 255), m, sr.Min, draw.Src)
	//png.Encode(fo, dest)

}*/
