package main

import (
	"math"
	"math/rand"

	"github.com/aquilax/go-perlin"
)

// Tile IDs
const (
	ID_Grass   = 552
	ID_Dirt    = 575
	ID_Stone   = 578
	ID_Log     = 220
	ID_Leaves  = 296
	ID_Planks  = 244
	ID_Furnace = 597
	ID_OreGold = 133
	ID_OreIron = 135
	ID_OreCoal = 128

	ID_Lava  = 294
	ID_Sand  = 576
	ID_Water = 295
	ID_Chest = 600

	// New decorative tiles
	ID_Mushroom   = 297 // Assuming these exist in tileset
	ID_Crystal    = 298
	ID_Flower     = 299
	ID_SnowGrass  = 553
	ID_MossyStone = 579
	ID_DarkStone  = 580
	ID_SkyGrass   = 554
)

// Biomes
const (
	BiomePlains = iota
	BiomeForest
	BiomeMountains
	BiomeDesert
	BiomeSwamp
)

// Underground biomes
const (
	UndergroundNormal = iota
	UndergroundCrystal
	UndergroundMushroom
	UndergroundLava
)

// Biome storage
type BiomeData struct {
	SurfaceHeights    []int
	Biomes            []int
	UndergroundBiomes [][]int // 2D grid of underground biome types
}

var worldBiomeData *BiomeData

func (tm *Tilemap) GenerateTerrariaWorld() {
	if IsEmbedded() {
		tm.Cols = 200
		tm.Rows = 100
	} else {
		tm.Cols = 400
		tm.Rows = 150
	}

	tm.Grid = make([][]int, tm.Rows)
	for y := 0; y < tm.Rows; y++ {
		tm.Grid[y] = make([]int, tm.Cols)
	}

	seed := rand.Int63()
	perlinGen := perlin.NewPerlin(2, 2, 3, seed)
	surfaceHeight := make([]int, tm.Cols)
	biomes := make([]int, tm.Cols)

	// Store biome data for runtime use
	worldBiomeData = &BiomeData{
		SurfaceHeights:    surfaceHeight,
		Biomes:            biomes,
		UndergroundBiomes: make([][]int, tm.Rows),
	}
	for y := 0; y < tm.Rows; y++ {
		worldBiomeData.UndergroundBiomes[y] = make([]int, tm.Cols)
	}

	// Determine biomes
	biomeNoise := make([]float64, tm.Cols)
	for x := 0; x < tm.Cols; x++ {
		biomeNoise[x] = perlinGen.Noise1D(float64(x) * 0.003) // Lower frequency for larger biomes
	}

	// Assign biomes with transition zones
	for x := 0; x < tm.Cols; x++ {
		bn := biomeNoise[x]

		var biome int
		if bn < -0.35 {
			biome = BiomePlains
		} else if bn < -0.1 {
			biome = BiomeForest
		} else if bn < 0.15 {
			biome = BiomeMountains
		} else if bn < 0.4 {
			biome = BiomeDesert
		} else {
			biome = BiomeSwamp
		}
		biomes[x] = biome
	}

	// Generate surface height
	for x := 0; x < tm.Cols; x++ {
		biome := biomes[x]

		// Get heights for current biome
		height := generateBiomeHeight(perlinGen, x, biome)

		// Smooth transitions
		transitionWidth := 30
		if x > transitionWidth && biomes[x] != biomes[x-transitionWidth] {
			// We're in a transition zone - blend heights
			prevBiome := biomes[x-transitionWidth]
			prevHeight := generateBiomeHeight(perlinGen, x, prevBiome)

			// Calculate blend factor based on position in transition
			blendFactor := 0.5
			for checkX := x - transitionWidth; checkX < x; checkX++ {
				if biomes[checkX] != biomes[x] {
					blendFactor = float64(x-checkX) / float64(transitionWidth)
					break
				}
			}

			height = prevHeight*(1-blendFactor) + height*blendFactor
		}

		// Clamp
		if height < 15 {
			height = 15
		}
		if height > float64(tm.Rows-30) {
			height = float64(tm.Rows - 30)
		}

		surfaceHeight[x] = int(height)
	}

	// Fill blocks
	for x := 0; x < tm.Cols; x++ {
		yGeo := surfaceHeight[x]

		// Surface Block
		if yGeo >= 0 && yGeo < tm.Rows {
			switch biomes[x] {
			case BiomeDesert:
				tm.Grid[yGeo][x] = ID_Sand
			case BiomeSwamp:
				tm.Grid[yGeo][x] = ID_Dirt
			case BiomeMountains:
				if rand.Float64() < 0.3 {
					tm.Grid[yGeo][x] = ID_Stone
				} else {
					tm.Grid[yGeo][x] = ID_Grass
				}
			default:
				tm.Grid[yGeo][x] = ID_Grass
			}
		}

		// Dirt layer
		dirtDepth := 8 + rand.Intn(6)
		if biomes[x] == BiomeMountains {
			dirtDepth = 2 + rand.Intn(3)
		} else if biomes[x] == BiomeDesert {
			dirtDepth = 12 + rand.Intn(5) // Deeper sand
		}

		for y := yGeo + 1; y < tm.Rows; y++ {
			depth := y - yGeo

			if depth < dirtDepth {
				// Dirt/Sand layer
				switch biomes[x] {
				case BiomeDesert:
					tm.Grid[y][x] = ID_Sand
				default:
					tm.Grid[y][x] = ID_Dirt
				}
			} else {
				// Stone layer
				undergroundBiome := determineUndergroundBiome(perlinGen, x, y, tm.Rows)
				worldBiomeData.UndergroundBiomes[y][x] = undergroundBiome

				switch undergroundBiome {
				case UndergroundCrystal:
					if rand.Float64() < 0.1 {
						tm.Grid[y][x] = ID_Crystal
					} else {
						tm.Grid[y][x] = ID_Stone
					}
				case UndergroundMushroom:
					tm.Grid[y][x] = ID_MossyStone
				case UndergroundLava:
					tm.Grid[y][x] = ID_DarkStone
				default:
					tm.Grid[y][x] = ID_Stone
				}

				// Ores
				depthFactor := float64(depth) / float64(tm.Rows-yGeo)
				if rand.Float64() < 0.015*(1+depthFactor) {
					generateOreVein(tm, x, y, ID_OreCoal)
				} else if rand.Float64() < 0.008*(1+depthFactor*0.5) && depth > 30 {
					generateOreVein(tm, x, y, ID_OreIron)
				} else if rand.Float64() < 0.003*(1+depthFactor) && depth > 60 {
					generateOreVein(tm, x, y, ID_OreGold)
				}
			}
		}
	}

	// Lava and water
	for y := tm.Rows - 8; y < tm.Rows; y++ {
		for x := 0; x < tm.Cols; x++ {
			tm.Grid[y][x] = ID_Lava
		}
	}

	// Swamp water
	for x := 0; x < tm.Cols; x++ {
		if biomes[x] == BiomeSwamp {
			waterLevel := surfaceHeight[x] + 2
			for y := waterLevel; y < waterLevel+5 && y < tm.Rows-10; y++ {
				if tm.Grid[y][x] == 0 || tm.Grid[y][x] == ID_Dirt {
					tm.Grid[y][x] = ID_Water
				}
			}
		}
	}

	numCaves := tm.Cols / 15
	if IsEmbedded() {
		numCaves = tm.Cols / 25
	}
	for i := 0; i < numCaves; i++ {
		cx := rand.Intn(tm.Cols)
		// Fix WASM panic: ensure positive range
		minDepth := surfaceHeight[cx] + 20
		maxDepth := tm.Rows - 20
		if maxDepth <= minDepth {
			continue // Skip if no space for cave
		}
		cy := minDepth + rand.Intn(maxDepth-minDepth)

		if rand.Float64() < 0.25 {
			generateCavern(tm, cx, cy)
		} else if rand.Float64() < 0.1 {
			generateUndergroundLake(tm, cx, cy)
		} else {
			generateSmootherCave(tm, cx, cy)
		}
	}

	// Sky islands
	numSkyIslands := 3 + rand.Intn(3)
	for i := 0; i < numSkyIslands; i++ {
		ix := 50 + rand.Intn(tm.Cols-100)
		iy := 10 + rand.Intn(30) // High in the sky
		generateSkyIsland(tm, ix, iy)
	}

	// Surface features
	for x := 5; x < tm.Cols-5; x++ {
		y := surfaceHeight[x]
		if y <= 0 || y >= tm.Rows {
			continue
		}

		surfaceTile := tm.Grid[y][x]

		// Trees
		if surfaceTile == ID_Grass {
			chance := 0.0
			switch biomes[x] {
			case BiomePlains:
				chance = 0.02
			case BiomeForest:
				chance = 0.12
			case BiomeMountains:
				chance = 0.005
			}

			if rand.Float64() < chance && tm.Grid[y-1][x] == 0 {
				generateTree(tm, x, y)
				x += 3 // Skip area after tree
			} else if rand.Float64() < 0.05 {
				// Flowers
				if y-1 >= 0 && tm.Grid[y-1][x] == 0 {
					tm.Grid[y-1][x] = ID_Flower
				}
			}
		}

		// Cacti
		if surfaceTile == ID_Sand && rand.Float64() < 0.025 && tm.Grid[y-1][x] == 0 {
			generateCactus(tm, x, y)
			x += 2
		}

		// Rocks
		if biomes[x] == BiomeMountains && rand.Float64() < 0.03 {
			generateRockFormation(tm, x, y)
		}
	}

	// Spawn house
	spawnX := 100
	bestX := spawnX
	minVar := 1000.0
	for x := spawnX - 20; x < spawnX+20 && x+10 < tm.Cols; x++ {
		if x < 0 {
			continue
		}
		v := math.Abs(float64(surfaceHeight[x] - surfaceHeight[x+10]))
		if v < minVar {
			minVar = v
			bestX = x
		}
	}
	generateHouse(tm, bestX, surfaceHeight[bestX])

	if tm.Cols > 250 {
		numDungeons := 3 + rand.Intn(3)
		for i := 0; i < numDungeons; i++ {
			dungeonRange := tm.Cols - 200
			if dungeonRange < 1 {
				dungeonRange = 1
			}
			dx := 100 + rand.Intn(dungeonRange)
			if dx >= tm.Cols {
				dx = tm.Cols - 50
			}
			dy := surfaceHeight[dx] + 40 + rand.Intn(60)
			if dy < tm.Rows-20 {
				generateDungeon(tm, dx, dy)
			}
		}
	}
}

func generateBiomeHeight(p *perlin.Perlin, x int, biome int) float64 {
	baseHeight := 60.0

	switch biome {
	case BiomePlains:
		h1 := p.Noise1D(float64(x)*0.015) * 5.0
		h2 := p.Noise1D(float64(x)*0.08+100) * 1.5
		return baseHeight + h1 + h2
	case BiomeForest:
		h1 := p.Noise1D(float64(x)*0.012) * 12.0
		h2 := p.Noise1D(float64(x)*0.04+200) * 4.0
		return baseHeight + h1 + h2 - 5
	case BiomeMountains:
		h1 := p.Noise1D(float64(x)*0.008) * 35.0
		h2 := p.Noise1D(float64(x)*0.03+300) * 15.0
		h3 := p.Noise1D(float64(x)*0.1+350) * 5.0 // Jagged detail
		return baseHeight + h1 + h2 + h3 - 25
	case BiomeDesert:
		h1 := p.Noise1D(float64(x)*0.025) * 3.0
		h2 := p.Noise1D(float64(x)*0.12+400) * 1.0 // Dunes
		return baseHeight + h1 + h2 + 8
	case BiomeSwamp:
		h1 := p.Noise1D(float64(x)*0.02) * 4.0
		h2 := p.Noise1D(float64(x)*0.06+500) * 1.5
		return baseHeight + h1 + h2 - 12
	}
	return baseHeight
}

func determineUndergroundBiome(p *perlin.Perlin, x, y, maxY int) int {
	// Use 2D noise for underground biome distribution
	noise := p.Noise2D(float64(x)*0.02, float64(y)*0.02)
	depthRatio := float64(y) / float64(maxY)

	// Lava biome more common near bottom
	if depthRatio > 0.8 && noise > 0.2 {
		return UndergroundLava
	}

	// Crystal caves at mid-depth
	if depthRatio > 0.4 && depthRatio < 0.7 && noise > 0.3 {
		return UndergroundCrystal
	}

	// Mushroom caves scattered
	if noise < -0.3 && depthRatio > 0.3 {
		return UndergroundMushroom
	}

	return UndergroundNormal
}

func generateSmootherCave(tm *Tilemap, startX, startY int) {
	cx := float64(startX)
	cy := float64(startY)

	angle := rand.Float64() * 2 * math.Pi
	velocity := 1.0

	life := 80 + rand.Intn(120)

	for l := 0; l < life; l++ {
		// Change angle slowly
		angle += (rand.Float64() - 0.5) * 0.25

		cx += math.Cos(angle) * velocity
		cy += math.Sin(angle) * velocity

		radius := 2.0 + rand.Float64()*1.5

		// Dig
		ix := int(cx)
		iy := int(cy)
		r := int(radius)

		for rx := -r; rx <= r; rx++ {
			for ry := -r; ry <= r; ry++ {
				if rx*rx+ry*ry <= r*r {
					nx := ix + rx
					ny := iy + ry
					if nx > 0 && nx < tm.Cols-1 && ny > 0 && ny < tm.Rows-10 {
						tm.Grid[ny][nx] = 0
					}
				}
			}
		}
	}
}

func generateOreVein(tm *Tilemap, x, y, oreID int) {
	size := 3 + rand.Intn(5)
	for i := 0; i < size; i++ {
		ox := x + rand.Intn(3) - 1
		oy := y + rand.Intn(3) - 1
		if ox >= 0 && ox < tm.Cols && oy >= 0 && oy < tm.Rows-10 {
			if tm.Grid[oy][ox] == ID_Stone || tm.Grid[oy][ox] == ID_DarkStone {
				tm.Grid[oy][ox] = oreID
			}
		}
	}
}

func generateTree(tm *Tilemap, x, rootY int) {
	height := 5 + rand.Intn(5)
	// Trunk
	for h := 1; h <= height; h++ {
		if rootY-h >= 0 {
			tm.Grid[rootY-h][x] = ID_Log
		}
	}
	// Leaves - more natural shape
	top := rootY - height
	for ly := top - 3; ly <= top+1; ly++ {
		radiusAtHeight := 3 - int(math.Abs(float64(ly-top+1)))
		if radiusAtHeight < 1 {
			radiusAtHeight = 1
		}
		for lx := x - radiusAtHeight; lx <= x+radiusAtHeight; lx++ {
			if lx >= 0 && lx < tm.Cols && ly >= 0 && ly < tm.Rows {
				dist := math.Abs(float64(lx-x)) + math.Abs(float64(ly-top))*0.5
				if dist <= float64(radiusAtHeight)+0.5 {
					if tm.Grid[ly][lx] == 0 {
						tm.Grid[ly][lx] = ID_Leaves
					}
				}
			}
		}
	}
}

func generateCavern(tm *Tilemap, startX, startY int) {
	radius := 8 + rand.Intn(12)
	for rx := -radius; rx <= radius; rx++ {
		for ry := -radius; ry <= radius; ry++ {
			// Irregular shape
			distSq := float64(rx*rx) + float64(ry*ry)*0.8
			if distSq <= float64(radius*radius) {
				nx := startX + rx
				ny := startY + ry
				if nx > 0 && nx < tm.Cols-1 && ny > 0 && ny < tm.Rows-10 {
					tm.Grid[ny][nx] = 0
				}
			}
		}
	}

	// Stalactites and stalagmites
	for i := 0; i < radius; i++ {
		// Stalactite (from top)
		sx := startX + rand.Intn(radius*2) - radius
		sy := startY - radius + rand.Intn(radius/2)
		stalagHeight := 2 + rand.Intn(4)
		for h := 0; h < stalagHeight; h++ {
			if sx >= 0 && sx < tm.Cols && sy+h >= 0 && sy+h < tm.Rows {
				tm.Grid[sy+h][sx] = ID_Stone
			}
		}

		// Stalagmite (from bottom)
		sx = startX + rand.Intn(radius*2) - radius
		sy = startY + radius - rand.Intn(radius/2)
		for h := 0; h < stalagHeight; h++ {
			if sx >= 0 && sx < tm.Cols && sy-h >= 0 && sy-h < tm.Rows {
				tm.Grid[sy-h][sx] = ID_Stone
			}
		}
	}

	// Chests on floors
	for ry := -radius; ry <= radius; ry++ {
		for rx := -radius; rx <= radius; rx++ {
			nx := startX + rx
			ny := startY + ry
			if nx >= 0 && nx < tm.Cols && ny >= 0 && ny < tm.Rows && tm.Grid[ny][nx] == 0 {
				if ny+1 < tm.Rows && tm.IsSolid(tm.Grid[ny+1][nx]) && rand.Float64() < 0.03 {
					tm.Grid[ny][nx] = ID_Chest
				}
			}
		}
	}
}

func generateUndergroundLake(tm *Tilemap, startX, startY int) {
	radius := 6 + rand.Intn(8)

	// Dig the lake area
	for rx := -radius; rx <= radius; rx++ {
		for ry := -radius / 2; ry <= radius/2; ry++ {
			distSq := float64(rx*rx)/float64(radius*radius) + float64(ry*ry)/float64((radius/2)*(radius/2))
			if distSq <= 1.0 {
				nx := startX + rx
				ny := startY + ry
				if nx > 0 && nx < tm.Cols-1 && ny > 0 && ny < tm.Rows-10 {
					tm.Grid[ny][nx] = 0
				}
			}
		}
	}

	// Fill bottom with water
	for rx := -radius + 1; rx < radius; rx++ {
		for ry := 0; ry <= radius/2; ry++ {
			distSq := float64(rx*rx)/float64(radius*radius) + float64(ry*ry)/float64((radius/2)*(radius/2))
			if distSq <= 0.9 {
				nx := startX + rx
				ny := startY + ry
				if nx > 0 && nx < tm.Cols-1 && ny > 0 && ny < tm.Rows-10 {
					if tm.Grid[ny][nx] == 0 {
						tm.Grid[ny][nx] = ID_Water
					}
				}
			}
		}
	}
}

func generateCactus(tm *Tilemap, x, rootY int) {
	height := 2 + rand.Intn(3)
	for h := 1; h <= height; h++ {
		if rootY-h >= 0 {
			tm.Grid[rootY-h][x] = ID_Log
		}
	}
	// Arms
	if height > 2 && rand.Float64() < 0.5 {
		armY := rootY - height/2
		if armY >= 0 {
			if x-1 >= 0 {
				tm.Grid[armY][x-1] = ID_Log
			}
			if x+1 < tm.Cols {
				tm.Grid[armY][x+1] = ID_Log
			}
		}
	}
}

func generateRockFormation(tm *Tilemap, x, rootY int) {
	// Small rock pile
	size := 2 + rand.Intn(3)
	for rx := -size / 2; rx <= size/2; rx++ {
		for ry := 0; ry < size-int(math.Abs(float64(rx))); ry++ {
			nx := x + rx
			ny := rootY - 1 - ry
			if nx >= 0 && nx < tm.Cols && ny >= 0 && ny < tm.Rows {
				if tm.Grid[ny][nx] == 0 {
					tm.Grid[ny][nx] = ID_Stone
				}
			}
		}
	}
}

func generateSkyIsland(tm *Tilemap, x, y int) {
	width := 15 + rand.Intn(20)
	height := 4 + rand.Intn(4)

	// Generate island shape using noise
	for ix := 0; ix < width; ix++ {
		// Parabolic depth
		distFromCenter := math.Abs(float64(ix) - float64(width)/2)
		maxDepth := height - int(distFromCenter*float64(height)/float64(width)*1.5)
		if maxDepth < 1 {
			maxDepth = 1
		}

		for iy := 0; iy < maxDepth; iy++ {
			nx := x + ix
			ny := y + iy
			if nx >= 0 && nx < tm.Cols && ny >= 0 && ny < tm.Rows {
				if iy == 0 {
					tm.Grid[ny][nx] = ID_SkyGrass
				} else {
					tm.Grid[ny][nx] = ID_Dirt
				}
			}
		}
	}

	// Add a tree or chest on top
	centerX := x + width/2
	if centerX >= 0 && centerX < tm.Cols && y-1 >= 0 {
		if rand.Float64() < 0.5 {
			generateTree(tm, centerX, y)
		} else {
			tm.Grid[y-1][centerX] = ID_Chest
		}
	}
}

func generateHouse(tm *Tilemap, x, floorY int) {
	width := 14
	height := 8

	// Foundation
	for hx := x; hx < x+width && hx < tm.Cols; hx++ {
		if floorY >= 0 && floorY < tm.Rows && hx >= 0 {
			tm.Grid[floorY][hx] = ID_Stone // Stone foundation
		}
	}

	// Walls
	for hx := x; hx < x+width && hx < tm.Cols; hx++ {
		for hy := floorY - height; hy < floorY && hy < tm.Rows; hy++ {
			if hx >= 0 && hy >= 0 {
				isLeftWall := hx == x
				isRightWall := hx == x+width-1
				isCeiling := hy == floorY-height

				if isLeftWall || isRightWall || isCeiling {
					tm.Grid[hy][hx] = ID_Planks
				} else {
					tm.Grid[hy][hx] = 0 // Clear interior
				}
			}
		}
	}

	// Roof
	roofTop := floorY - height
	roofWidth := width + 4 // Overhang
	roofStartX := x - 2
	roofHeight := 5

	for level := 0; level < roofHeight; level++ {
		roofY := roofTop - 1 - level
		leftX := roofStartX + level
		rightX := roofStartX + roofWidth - 1 - level

		if roofY >= 0 && roofY < tm.Rows {
			for rx := leftX; rx <= rightX && rx < tm.Cols; rx++ {
				if rx >= 0 {
					if rx == leftX || rx == rightX || level == roofHeight-1 {
						tm.Grid[roofY][rx] = ID_Log // Roof edge
					}
				}
			}
		}
	}

	// Door
	doorX := x + 2
	if doorX < tm.Cols && doorX >= 0 {
		if floorY-1 >= 0 {
			tm.Grid[floorY-1][doorX] = 0
		}
		if floorY-2 >= 0 {
			tm.Grid[floorY-2][doorX] = 0
		}
		// Door frame
		if floorY-3 >= 0 {
			tm.Grid[floorY-3][doorX] = ID_Log
		}
	}

	// Windows
	windowX := x + width - 4
	windowY := floorY - 4
	for wx := 0; wx < 2; wx++ {
		for wy := 0; wy < 2; wy++ {
			winX := windowX + wx
			winY := windowY + wy
			if winX >= 0 && winX < tm.Cols && winY >= 0 && winY < tm.Rows {
				tm.Grid[winY][winX] = 0 // Window opening
			}
		}
	}

	// Interior
	// Furnace
	if floorY-1 >= 0 && x+width-2 >= 0 && x+width-2 < tm.Cols {
		tm.Grid[floorY-1][x+width-2] = ID_Furnace
	}

	// Chest
	if floorY-1 >= 0 && x+width/2 >= 0 && x+width/2 < tm.Cols {
		tm.Grid[floorY-1][x+width/2] = ID_Chest
	}

	// Table
	tableX := x + 4
	if floorY-1 >= 0 && tableX >= 0 && tableX < tm.Cols {
		tm.Grid[floorY-1][tableX] = ID_Planks
	}
	if floorY-1 >= 0 && tableX+1 >= 0 && tableX+1 < tm.Cols {
		tm.Grid[floorY-1][tableX+1] = ID_Planks
	}

	// Chimney
	chimneyX := x + width - 3
	chimneyBase := floorY - height - 3
	for cy := 0; cy < 4; cy++ {
		chimneyY := chimneyBase - cy
		if chimneyY >= 0 && chimneyY < tm.Rows && chimneyX >= 0 && chimneyX < tm.Cols {
			tm.Grid[chimneyY][chimneyX] = ID_Stone
		}
	}
}

func generateDungeon(tm *Tilemap, x, y int) {
	// Small dungeon room
	roomW := 10 + rand.Intn(8)
	roomH := 6 + rand.Intn(4)

	// Clear room
	for rx := 0; rx < roomW; rx++ {
		for ry := 0; ry < roomH; ry++ {
			nx := x + rx
			ny := y + ry
			if nx >= 0 && nx < tm.Cols && ny >= 0 && ny < tm.Rows-10 {
				if rx == 0 || rx == roomW-1 || ry == 0 || ry == roomH-1 {
					tm.Grid[ny][nx] = ID_Planks // Walls
				} else {
					tm.Grid[ny][nx] = 0 // Interior
				}
			}
		}
	}

	// Entrance
	entranceX := x + roomW/2
	if entranceX < tm.Cols && y >= 0 {
		tm.Grid[y][entranceX] = 0
		tm.Grid[y][entranceX+1] = 0
	}

	// Chest inside
	chestX := x + 2 + rand.Intn(roomW-4)
	chestY := y + roomH - 2
	if chestX >= 0 && chestX < tm.Cols && chestY >= 0 && chestY < tm.Rows {
		tm.Grid[chestY][chestX] = ID_Chest
	}
}

// Get biome at X
func GetBiomeAt(x int) int {
	if worldBiomeData == nil || x < 0 || x >= len(worldBiomeData.Biomes) {
		return BiomePlains
	}
	return worldBiomeData.Biomes[x]
}
