package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Tilemap struct {
	Tileset    *ebiten.Image
	TileCache  map[int]*ebiten.Image
	ChestImage *ebiten.Image
	Grid       [][]int
	Cols, Rows int
	TileSize   int
	DrawOpts   *ebiten.DrawImageOptions
}

func NewTilemap(tileset *ebiten.Image, tileSize int) *Tilemap {
	tm := &Tilemap{
		Tileset:   tileset,
		TileSize:  tileSize,
		TileCache: make(map[int]*ebiten.Image),
		DrawOpts:  &ebiten.DrawImageOptions{},
	}

	bounds := tileset.Bounds()
	tm.Cols = bounds.Dx() / tileSize
	tm.Rows = bounds.Dy() / tileSize // Tileset dimensions, not world dimensions

	// Chest used in NewTilemap
	chestSheet := loadImage("assets/images/tiles/chest.png")
	if chestSheet != nil {
		// Chest sheet 288x128
		// First frame
		frameW := 32
		frameH := 32
		firstFrame := chestSheet.SubImage(image.Rect(0, 0, frameW, frameH)).(*ebiten.Image)

		// 32x32 for visibility
		tm.ChestImage = ebiten.NewImage(frameW, frameH)
		op := &ebiten.DrawImageOptions{}
		tm.ChestImage.DrawImage(firstFrame, op)
	}

	return tm
}

func (tm *Tilemap) Draw(screen *ebiten.Image, camX, camY float64) {
	startCol := int(camX / float64(tm.TileSize))
	endCol := startCol + (ScreenWidth / tm.TileSize) + 2
	startRow := int(camY / float64(tm.TileSize))
	endRow := startRow + (ScreenHeight / tm.TileSize) + 2

	if startCol < 0 {
		startCol = 0
	}
	if startRow < 0 {
		startRow = 0
	}
	if endCol > tm.Cols {
		endCol = tm.Cols
	}
	if endRow > tm.Rows {
		endRow = tm.Rows
	}

	if tm.Grid == nil {
		return
	}

	for y := startRow; y < endRow; y++ {
		for x := startCol; x < endCol; x++ {
			tileID := tm.Grid[y][x]
			if tileID <= 0 {
				continue
			}

			var img *ebiten.Image
			isChest := tileID == ID_Chest
			if isChest {
				img = tm.ChestImage
			} else {
				// Lazy load
				if cached, ok := tm.TileCache[tileID]; ok {
					img = cached
				} else {
					// Calculate position in tileset
					tilesetCols := tm.Tileset.Bounds().Dx() / tm.TileSize
					tsX := (tileID % tilesetCols) * tm.TileSize
					tsY := (tileID / tilesetCols) * tm.TileSize

					// Bounds check
					if tsX+tm.TileSize <= tm.Tileset.Bounds().Dx() && tsY+tm.TileSize <= tm.Tileset.Bounds().Dy() {
						rect := image.Rect(tsX, tsY, tsX+tm.TileSize, tsY+tm.TileSize)
						img = tm.Tileset.SubImage(rect).(*ebiten.Image)
						tm.TileCache[tileID] = img
					}
				}
			}
			if img == nil {
				continue
			}

			tm.DrawOpts.GeoM.Reset()
			worldX := float64(x * tm.TileSize)
			worldY := float64(y * tm.TileSize)

			// Center chest
			if isChest {
				worldX -= 8  // Center X
				worldY -= 16 // Align bottom
			}

			tm.DrawOpts.GeoM.Translate(worldX-camX, worldY-camY)
			screen.DrawImage(img, tm.DrawOpts)
		}
	}
}

// Physics
func (tm *Tilemap) IsSolid(tileID int) bool {
	if tileID == 0 {
		return false
	}

	// Non-solid tiles: air, logs, leaves, chests, water, lava
	if tileID == 220 || tileID == 296 || tileID == ID_Chest || tileID == ID_Water || tileID == ID_Lava {
		return false
	}

	return true
}

func (tm *Tilemap) GetTile(x, y int) int {
	if x < 0 || x >= tm.Cols || y < 0 || y >= tm.Rows {
		return 0 // Air if out
	}
	return tm.Grid[y][x]
}
