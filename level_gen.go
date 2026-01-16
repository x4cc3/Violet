package main

import (
	"math"
	"math/rand"

	"github.com/aquilax/go-perlin"
)

// Tile IDs
const (
	ID_Grass   = 92
	ID_Dirt    = 193
	ID_Stone   = 578
	ID_Log     = 220
	ID_Leaves  = 296
	ID_Planks  = 247
	ID_Furnace = 596
	ID_OreGold = 132
	ID_OreIron = 134
	ID_OreCoal = 127

	ID_Lava     = 162
	ID_Sand     = 582
	ID_Water    = 142 // placeholder; water generation removed
	ID_Chest    = 600
	ID_BigChest = 602 // Sacred chest for quest reward

	ID_Mushroom   = 191
	ID_Crystal    = 200
	ID_Flower     = 291
	ID_SnowGrass  = 570
	ID_MossyStone = 31
	ID_DarkStone  = 5
	ID_SkyGrass   = 558
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
		transitionWidth := 50 // Wider transitions for smoother biome blending
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

		// Enhanced neighbor smoothing for gentler terrain
		if x > 2 {
			// Weighted average with more neighbors
			height = (height*0.4 + float64(surfaceHeight[x-1])*0.3 + float64(surfaceHeight[x-2])*0.2 + float64(surfaceHeight[x-3])*0.1)
			// Strict slope limit - max 2 tiles difference
			prevH := float64(surfaceHeight[x-1])
			if height-prevH > 2 {
				height = prevH + 2
			}
			if prevH-height > 2 {
				height = prevH - 2
			}
		} else if x > 0 {
			height = (height + float64(surfaceHeight[x-1])) / 2
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

	// Spawn zone flattening around the start area
	spawnStart := 80
	spawnEnd := 140
	if spawnEnd >= tm.Cols {
		spawnEnd = tm.Cols - 1
	}
	if spawnStart < 2 {
		spawnStart = 2
	}
	// Flatten to mean height in window to ensure safe start
	sum := 0
	count := 0
	for x := spawnStart; x <= spawnEnd; x++ {
		sum += surfaceHeight[x]
		count++
	}
	if count > 0 {
		avg := sum / count
		for x := spawnStart; x <= spawnEnd; x++ {
			surfaceHeight[x] = avg
		}
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

				// Ores (single roll to prevent dense overlaps), with biome/depth bias
				if depth > 2 {
					r := rand.Float64()
					depthFactor := float64(depth) / float64(tm.Rows-yGeo)
					// Base chances
					coalChance := 0.010 * (1 + depthFactor)
					ironChance := 0.006 * (1 + depthFactor*0.5)
					goldChance := 0.0025 * (1 + depthFactor)

					// Biome tweaks
					if biomes[x] == BiomeMountains {
						ironChance *= 1.4
					}
					if biomes[x] == BiomeDesert {
						goldChance *= 1.25
					}
					if biomes[x] == BiomeForest || biomes[x] == BiomePlains {
						coalChance *= 1.15
					}

					if depth > 60 && r < goldChance {
						generateOreVein(tm, x, y, ID_OreGold)
					} else if depth > 30 && r < ironChance {
						generateOreVein(tm, x, y, ID_OreIron)
					} else if r < coalChance {
						generateOreVein(tm, x, y, ID_OreCoal)
					}
				}

			}
		}
	}

	// Lava band at bottom (keep a buffer above to avoid one-tile ceilings)
	for y := tm.Rows - 8; y < tm.Rows; y++ {
		for x := 0; x < tm.Cols; x++ {
			tm.Grid[y][x] = ID_Lava
		}
	}

	// Water disabled: no swamp fill

	// Enhanced cave system with more variety
	numCaves := tm.Cols / 12 // More caves
	if IsEmbedded() {
		numCaves = tm.Cols / 20
	}

	cavePositions := make([][2]int, 0) // Track cave positions for connections

	for i := 0; i < numCaves; i++ {
		cx := rand.Intn(tm.Cols)
		// Varied depth - some shallow, some deep
		minDepth := surfaceHeight[cx] + 15
		maxDepth := tm.Rows - 15
		if maxDepth <= minDepth {
			continue
		}

		// Bias towards mid-depth for larger caves
		depthRange := maxDepth - minDepth
		cy := minDepth + rand.Intn(depthRange)

		// More variety in cave types
		r := rand.Float64()
		if r < 0.3 {
			generateCavern(tm, cx, cy)
		} else if r < 0.5 {
			// Large cavern
			generateCavern(tm, cx, cy)
			generateCavern(tm, cx+rand.Intn(10)-5, cy+rand.Intn(10)-5)
		} else {
			generateSmootherCave(tm, cx, cy)
		}

		cavePositions = append(cavePositions, [2]int{cx, cy})
	}

	// Connect some nearby caves with tunnels
	for i := 0; i < len(cavePositions)-1; i++ {
		if rand.Float64() < 0.3 {
			c1, c2 := cavePositions[i], cavePositions[i+1]
			dist := math.Sqrt(float64((c1[0]-c2[0])*(c1[0]-c2[0]) + (c1[1]-c2[1])*(c1[1]-c2[1])))
			if dist < 50 {
				// Create connecting tunnel
				generateTunnel(tm, c1[0], c1[1], c2[0], c2[1])
			}
		}
	}

	// Sky islands with more variety
	numSkyIslands := 3 + rand.Intn(3)
	for i := 0; i < numSkyIslands; i++ {
		ix := 50 + rand.Intn(tm.Cols-100)
		iy := 10 + rand.Intn(30) // High in the sky
		generateSkyIsland(tm, ix, iy)
	}

	// Surface features with enhanced variety
	lastBuildingX := -50 // Track last building position for spacing
	for x := 5; x < tm.Cols-5; x++ {
		y := surfaceHeight[x]
		if y <= 0 || y >= tm.Rows {
			continue
		}

		surfaceTile := tm.Grid[y][x]

		// Trees with more variety
		if surfaceTile == ID_Grass {
			chance := 0.0
			switch biomes[x] {
			case BiomePlains:
				chance = 0.03 // Slightly more trees
			case BiomeForest:
				chance = 0.18 // Dense forests
			case BiomeMountains:
				chance = 0.008
			}

			if rand.Float64() < chance && tm.Grid[y-1][x] == 0 {
				generateTree(tm, x, y)
				x += 2 + rand.Intn(2) // Variable spacing
			} else if rand.Float64() < 0.08 {
				// Flowers and grass tufts
				if y-1 >= 0 && tm.Grid[y-1][x] == 0 {
					if rand.Float64() < 0.6 {
						tm.Grid[y-1][x] = ID_Flower
					} else {
						// Grass tuft (use leaves as grass)
						tm.Grid[y-1][x] = ID_Leaves
					}
				}
			}
		}

		// Swamp decorations: mushrooms and dead trees
		if biomes[x] == BiomeSwamp && surfaceTile == ID_Dirt {
			if rand.Float64() < 0.06 && y-1 >= 0 && tm.Grid[y-1][x] == 0 {
				tm.Grid[y-1][x] = ID_Mushroom
			} else if rand.Float64() < 0.02 && y-1 >= 0 && tm.Grid[y-1][x] == 0 {
				// Dead tree (just logs, no leaves)
				height := 3 + rand.Intn(2)
				for h := 1; h <= height; h++ {
					if y-h >= 0 {
						tm.Grid[y-h][x] = ID_Log
					}
				}
				x += 2
			}
		}

		// Desert: cacti and dead bushes
		if surfaceTile == ID_Sand {
			if rand.Float64() < 0.035 && tm.Grid[y-1][x] == 0 {
				generateCactus(tm, x, y)
				x += 2
			} else if rand.Float64() < 0.02 && y-1 >= 0 && tm.Grid[y-1][x] == 0 {
				// Desert rock/boulder
				tm.Grid[y-1][x] = ID_Stone
			}
		}

		// Mountains: more rock formations and crystals
		if biomes[x] == BiomeMountains {
			if rand.Float64() < 0.05 {
				generateRockFormation(tm, x, y)
			}
			if rand.Float64() < 0.01 && y-1 >= 0 && tm.Grid[y-1][x] == 0 {
				tm.Grid[y-1][x] = ID_Crystal
			}
		}

		// Decorative buildings (sparse, with spacing)
		if x-lastBuildingX > 50 && rand.Float64() < 0.02 {
			switch biomes[x] {
			case BiomePlains, BiomeForest:
				if surfaceTile == ID_Grass && y-6 >= 0 {
					generateCottage(tm, x, y)
					lastBuildingX = x
					x += 10
				}
			case BiomeDesert:
				if surfaceTile == ID_Sand && y-5 >= 0 {
					generateDesertHut(tm, x, y)
					lastBuildingX = x
					x += 8
				}
			case BiomeMountains:
				if y-7 >= 0 {
					generateMountainCabin(tm, x, y)
					lastBuildingX = x
					x += 12
				}
			case BiomeSwamp:
				if y-8 >= 0 {
					generateSwampHut(tm, x, y)
					lastBuildingX = x
					x += 8
				}
			}
		}
	}

	// Generate mountain encounter chamber
	generateMountainChamber(tm, surfaceHeight)

	// Place HP healing chests along path to chamber
	placePathChests(tm, surfaceHeight)
}

func generateBiomeHeight(p *perlin.Perlin, x int, biome int) float64 {
	baseHeight := 55.0

	switch biome {
	case BiomePlains:
		// Rolling gentle hills
		h1 := p.Noise1D(float64(x)*0.01) * 3.0
		h2 := p.Noise1D(float64(x)*0.05+100) * 1.0
		return baseHeight + h1 + h2
	case BiomeForest:
		// Gentle hills with occasional rises
		h1 := p.Noise1D(float64(x)*0.008) * 4.0
		h2 := p.Noise1D(float64(x)*0.03+200) * 2.0
		h3 := p.Noise1D(float64(x)*0.1+250) * 0.5
		return baseHeight + h1 + h2 + h3 - 3
	case BiomeMountains:
		// Dramatic peaks and valleys
		h1 := p.Noise1D(float64(x)*0.006) * 8.0
		h2 := p.Noise1D(float64(x)*0.02+300) * 4.0
		h3 := p.Noise1D(float64(x)*0.08+350) * 1.0
		return baseHeight + h1 + h2 + h3 - 8
	case BiomeDesert:
		// Sand dunes - gentle waves
		h1 := p.Noise1D(float64(x)*0.015) * 2.5
		h2 := p.Noise1D(float64(x)*0.08+400) * 1.0
		h3 := math.Sin(float64(x)*0.1) * 1.5 // Regular dune pattern
		return baseHeight + h1 + h2 + h3 + 5
	case BiomeSwamp:
		// Low and flat with occasional bumps
		h1 := p.Noise1D(float64(x)*0.012) * 2.0
		h2 := p.Noise1D(float64(x)*0.04+500) * 1.0
		return baseHeight + h1 + h2 - 8
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

// Generate a connecting tunnel between two points
func generateTunnel(tm *Tilemap, x1, y1, x2, y2 int) {
	steps := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
	if steps == 0 {
		return
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(float64(x1)*(1-t) + float64(x2)*t)
		y := int(float64(y1)*(1-t) + float64(y2)*t)

		// Carve a 2-tile radius tunnel
		for rx := -2; rx <= 2; rx++ {
			for ry := -2; ry <= 2; ry++ {
				if rx*rx+ry*ry <= 4 {
					nx := x + rx
					ny := y + ry
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

// Decorative cottage for plains/forest biomes
func generateCottage(tm *Tilemap, x, groundY int) {
	width := 6 + rand.Intn(3)  // 6-8 wide
	height := 4 + rand.Intn(2) // 4-5 tall

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Foundation and walls (solid planks)
	for hx := 0; hx < width; hx++ {
		for hy := 0; hy < height; hy++ {
			setTile(x+hx, groundY-1-hy, ID_Planks)
		}
	}

	// Log corners for detail
	for hy := 0; hy < height; hy++ {
		setTile(x, groundY-1-hy, ID_Log)
		setTile(x+width-1, groundY-1-hy, ID_Log)
	}

	// Pitched roof (logs)
	roofBase := groundY - height
	roofWidth := width + 2
	roofStart := x - 1
	for level := 0; level <= (roofWidth/2)+1; level++ {
		for rx := level; rx < roofWidth-level; rx++ {
			setTile(roofStart+rx, roofBase-level, ID_Log)
		}
		if level >= roofWidth/2 {
			break
		}
	}

	// Small window (decorative - just different tile)
	windowX := x + width/2
	windowY := groundY - height/2 - 1
	if windowY >= 0 {
		setTile(windowX, windowY, ID_Stone) // Stone as window frame
	}
}

// Decorative hut for desert biome
func generateDesertHut(tm *Tilemap, x, groundY int) {
	width := 5 + rand.Intn(3)  // 5-7 wide
	height := 3 + rand.Intn(2) // 3-4 tall

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Sandstone walls (using stone with sand accents)
	for hx := 0; hx < width; hx++ {
		for hy := 0; hy < height; hy++ {
			setTile(x+hx, groundY-1-hy, ID_Stone)
		}
	}

	// Sand trim at top
	for hx := 0; hx < width; hx++ {
		setTile(x+hx, groundY-height, ID_Sand)
	}

	// Flat roof extends slightly
	roofY := groundY - height - 1
	for hx := -1; hx <= width; hx++ {
		setTile(x+hx, roofY, ID_Stone)
	}

	// Decorative pot (stone block) beside hut
	if rand.Float64() < 0.5 {
		setTile(x-1, groundY-1, ID_Stone)
	}
}

// Decorative cabin for mountain biome
func generateMountainCabin(tm *Tilemap, x, groundY int) {
	width := 7 + rand.Intn(3)  // 7-9 wide
	height := 4 + rand.Intn(2) // 4-5 tall

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Stone walls
	for hx := 0; hx < width; hx++ {
		for hy := 0; hy < height; hy++ {
			setTile(x+hx, groundY-1-hy, ID_Stone)
		}
	}

	// Dark stone corners
	for hy := 0; hy < height; hy++ {
		setTile(x, groundY-1-hy, ID_DarkStone)
		setTile(x+width-1, groundY-1-hy, ID_DarkStone)
	}

	// Steep pitched roof (dark stone)
	roofBase := groundY - height
	for level := 0; level <= width/2+1; level++ {
		for rx := level; rx < width-level; rx++ {
			setTile(x+rx, roofBase-level, ID_DarkStone)
		}
		if level >= width/2 {
			break
		}
	}

	// Chimney
	chimneyX := x + width - 2
	chimneyTop := roofBase - width/2 - 1
	for chy := chimneyTop; chy <= roofBase; chy++ {
		setTile(chimneyX, chy, ID_Stone)
	}
}

// Decorative stilted structure for swamp biome
func generateSwampHut(tm *Tilemap, x, groundY int) {
	width := 5 + rand.Intn(2)  // 5-6 wide
	height := 3                // 3 tall cabin
	stilts := 2 + rand.Intn(2) // 2-3 tall stilts

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Stilts (logs)
	floorY := groundY - stilts - 1
	setTile(x, floorY+1, ID_Log)
	setTile(x+width-1, floorY+1, ID_Log)
	for sy := 0; sy < stilts; sy++ {
		setTile(x, groundY-1-sy, ID_Log)
		setTile(x+width-1, groundY-1-sy, ID_Log)
	}

	// Platform and walls (planks)
	for hx := 0; hx < width; hx++ {
		setTile(x+hx, floorY, ID_Planks) // floor
		for hy := 1; hy <= height; hy++ {
			setTile(x+hx, floorY-hy, ID_Planks)
		}
	}

	// Simple flat roof
	roofY := floorY - height - 1
	for hx := -1; hx <= width; hx++ {
		setTile(x+hx, roofY, ID_Log)
	}
}

// Mountain encounter chamber - returns center X,Y for slime spawning
var MountainChamberX, MountainChamberY float64

func generateMountainChamber(tm *Tilemap, surfaceHeight []int) {
	// Chamber location: x ~ 280-320 tiles (far from spawn at x~100)
	chamberX := 300
	if chamberX >= tm.Cols-40 {
		chamberX = tm.Cols - 50
	}

	chamberWidth := 40 // 40 tiles wide arena

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Flatten the ground in the chamber area
	avgHeight := 0
	for x := chamberX; x < chamberX+chamberWidth && x < tm.Cols; x++ {
		avgHeight += surfaceHeight[x]
	}
	avgHeight /= chamberWidth

	// Set chamber floor height
	for x := chamberX; x < chamberX+chamberWidth && x < tm.Cols; x++ {
		surfaceHeight[x] = avgHeight
	}

	// Clear the arena (air above ground)
	arenaHeight := 12 // Clear 12 tiles high
	for x := chamberX; x < chamberX+chamberWidth && x < tm.Cols; x++ {
		groundY := avgHeight
		// Set ground
		setTile(x, groundY, ID_Stone)
		// Clear above
		for y := groundY - arenaHeight; y < groundY; y++ {
			if y >= 0 {
				setTile(x, y, 0) // Air
			}
		}
		// Fill below with stone
		for y := groundY + 1; y < tm.Rows-10; y++ {
			setTile(x, y, ID_Stone)
		}
	}

	// Enhanced arena walls with pillars
	wallHeight := 15
	pillarSpacing := 8

	// Side walls
	for y := avgHeight - wallHeight; y <= avgHeight; y++ {
		if y >= 0 {
			setTile(chamberX-1, y, ID_DarkStone)
			setTile(chamberX-2, y, ID_DarkStone)
			setTile(chamberX+chamberWidth, y, ID_DarkStone)
			setTile(chamberX+chamberWidth+1, y, ID_DarkStone)
		}
	}

	// Decorative pillars inside arena
	for px := chamberX + pillarSpacing; px < chamberX+chamberWidth-pillarSpacing; px += pillarSpacing {
		for py := avgHeight - 6; py < avgHeight; py++ {
			setTile(px, py, ID_Stone)
		}
		// Crystal on top of pillars
		setTile(px, avgHeight-7, ID_Crystal)
	}

	// Entrance ramps (gradual slopes at edges)
	for i := 0; i < 4; i++ {
		rampY := avgHeight - i
		if rampY >= 0 {
			setTile(chamberX-3-i, rampY, ID_Stone)
			setTile(chamberX+chamberWidth+2+i, rampY, ID_Stone)
		}
	}

	// Store chamber center for slime spawning (world coordinates)
	MountainChamberX = float64((chamberX + chamberWidth/2) * tm.TileSize)
	MountainChamberY = float64((avgHeight - 2) * tm.TileSize) // Slightly above ground
}

// Place HP healing chests along the path from spawn to mountain chamber
func placePathChests(tm *Tilemap, surfaceHeight []int) {
	// Place 6 chests between spawn (x~100) and chamber (x~300)
	chestPositions := []int{115, 140, 170, 200, 235, 270}

	for _, cx := range chestPositions {
		if cx >= tm.Cols {
			continue
		}
		groundY := surfaceHeight[cx]
		chestY := groundY - 1 // One tile above ground

		// Verify there's air above and it's on solid ground
		if chestY >= 0 && chestY < tm.Rows && tm.Grid[chestY][cx] == 0 && tm.Grid[groundY][cx] != 0 {
			tm.Grid[chestY][cx] = ID_Chest
		}
	}
}

func generateSkyIsland(tm *Tilemap, x, y int) {
	// Varied island sizes
	islandType := rand.Intn(3)
	var width, height int

	switch islandType {
	case 0: // Small island
		width = 10 + rand.Intn(8)
		height = 3 + rand.Intn(2)
	case 1: // Medium island
		width = 18 + rand.Intn(12)
		height = 4 + rand.Intn(3)
	case 2: // Large island
		width = 25 + rand.Intn(15)
		height = 5 + rand.Intn(4)
	}

	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}

	// Generate island shape using noise
	for ix := 0; ix < width; ix++ {
		// Parabolic depth for natural shape
		distFromCenter := math.Abs(float64(ix) - float64(width)/2)
		maxDepth := height - int(distFromCenter*float64(height)/float64(width)*1.8)
		if maxDepth < 1 {
			maxDepth = 1
		}

		for iy := 0; iy < maxDepth; iy++ {
			nx := x + ix
			ny := y + iy
			if iy == 0 {
				setTile(nx, ny, ID_SkyGrass)
			} else if iy == maxDepth-1 && rand.Float64() < 0.3 {
				// Occasional mossy stone underneath
				setTile(nx, ny, ID_MossyStone)
			} else {
				setTile(nx, ny, ID_Dirt)
			}
		}

		// Hanging vines/crystals underneath some islands
		if rand.Float64() < 0.15 && maxDepth > 1 {
			vineLength := 2 + rand.Intn(3)
			for v := 0; v < vineLength; v++ {
				setTile(x+ix, y+maxDepth+v, ID_Leaves)
			}
		}
	}

	// Add decorations on top
	centerX := x + width/2

	// Trees (more on larger islands)
	numTrees := 1
	if islandType >= 1 {
		numTrees = 2 + rand.Intn(2)
	}

	for t := 0; t < numTrees; t++ {
		treeX := x + 3 + rand.Intn(width-6)
		if treeX >= 0 && treeX < tm.Cols && y-1 >= 0 {
			if tm.Grid[y][treeX] == ID_SkyGrass {
				generateTree(tm, treeX, y)
			}
		}
	}

	// Chest on medium/large islands
	if islandType >= 1 && centerX >= 0 && centerX < tm.Cols && y-1 >= 0 {
		setTile(centerX, y-1, ID_Chest)
	}

	// Flowers scattered on top
	for fx := x + 1; fx < x+width-1; fx++ {
		if rand.Float64() < 0.15 && fx >= 0 && fx < tm.Cols && y-1 >= 0 {
			if tm.Grid[y][fx] == ID_SkyGrass && tm.Grid[y-1][fx] == 0 {
				setTile(fx, y-1, ID_Flower)
			}
		}
	}

	// Floating crystal near some islands
	if rand.Float64() < 0.4 {
		crystalX := x + width + 3
		crystalY := y - 2 + rand.Intn(4)
		if crystalX < tm.Cols && crystalY >= 0 {
			setTile(crystalX, crystalY, ID_Crystal)
		}
	}
}

func generateHouse(tm *Tilemap, x, floorY int) {
	width := 16
	mainHeight := 6
	basementDepth := 5

	// Helper to safely set tile
	setTile := func(tx, ty, id int) {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			tm.Grid[ty][tx] = id
		}
	}
	getTile := func(tx, ty int) int {
		if tx >= 0 && tx < tm.Cols && ty >= 0 && ty < tm.Rows {
			return tm.Grid[ty][tx]
		}
		return 0
	}

	// Ground level is floorY (the grass/surface tile)
	// Interior floor will be at floorY (same level as outside ground)

	// Clear terrain around house first
	for hx := x - 3; hx < x+width+5; hx++ {
		for hy := floorY - mainHeight - 8; hy <= floorY+basementDepth+1; hy++ {
			// Clear trees/leaves above ground level
			if hy < floorY {
				tile := getTile(hx, hy)
				if tile == ID_Log || tile == ID_Leaves {
					setTile(hx, hy, 0)
				}
			}
		}
	}

	// ===== FOUNDATION =====
	// Solid stone foundation under the house
	for hx := x; hx < x+width; hx++ {
		setTile(hx, floorY, ID_Stone)
		setTile(hx, floorY+1, ID_Stone)
	}

	// ===== BASEMENT =====
	basementFloorY := floorY + basementDepth
	for hx := x; hx < x+width; hx++ {
		for hy := floorY + 2; hy <= basementFloorY; hy++ {
			isLeftWall := hx == x
			isRightWall := hx == x+width-1
			isFloor := hy == basementFloorY

			if isLeftWall || isRightWall || isFloor {
				setTile(hx, hy, ID_Stone)
			} else {
				setTile(hx, hy, 0) // Clear interior
			}
		}
	}

	// Basement stairs (right side, actual walkable stairs)
	stairX := x + width - 4
	for i := 0; i < basementDepth-1; i++ {
		stairY := floorY + 1 + i
		// Each stair step
		setTile(stairX-i, stairY, ID_Stone)
		// Clear above stairs
		for clearY := floorY + 1; clearY < stairY; clearY++ {
			setTile(stairX-i, clearY, 0)
		}
	}

	// Basement chests
	setTile(x+2, basementFloorY-1, ID_Chest)
	setTile(x+4, basementFloorY-1, ID_Chest)

	// Basement light
	setTile(x+1, basementFloorY-2, ID_Furnace)

	// ===== MAIN FLOOR WALLS =====
	mainCeiling := floorY - mainHeight
	for hx := x; hx < x+width; hx++ {
		for hy := mainCeiling; hy < floorY; hy++ {
			isLeftWall := hx == x
			isRightWall := hx == x+width-1
			isCeiling := hy == mainCeiling

			if isLeftWall || isRightWall || isCeiling {
				setTile(hx, hy, ID_Planks)
			} else {
				setTile(hx, hy, 0) // Clear interior
			}
		}
	}

	// ===== DOOR =====
	// Door at ground level (floorY is ground, so door opens at floorY-1, floorY-2, floorY-3)
	doorX := x + 4
	// Clear door opening (3 high)
	setTile(doorX, floorY-1, 0)
	setTile(doorX, floorY-2, 0)
	setTile(doorX, floorY-3, 0)
	setTile(doorX+1, floorY-1, 0)
	setTile(doorX+1, floorY-2, 0)
	setTile(doorX+1, floorY-3, 0)

	// Door frame
	setTile(doorX-1, floorY-1, ID_Log)
	setTile(doorX-1, floorY-2, ID_Log)
	setTile(doorX-1, floorY-3, ID_Log)
	setTile(doorX+2, floorY-1, ID_Log)
	setTile(doorX+2, floorY-2, ID_Log)
	setTile(doorX+2, floorY-3, ID_Log)
	// Top of door frame
	setTile(doorX, floorY-4, ID_Log)
	setTile(doorX+1, floorY-4, ID_Log)

	// Clear foundation under door for entry
	setTile(doorX, floorY, 0)
	setTile(doorX+1, floorY, 0)

	// ===== WINDOWS =====
	windowY := floorY - 3
	// Left window
	setTile(x+1, windowY, 0)
	setTile(x+1, windowY-1, 0)
	// Right window
	setTile(x+width-2, windowY, 0)
	setTile(x+width-2, windowY-1, 0)

	// ===== INTERIOR FURNITURE =====
	// Fireplace (back right)
	setTile(x+width-3, floorY-1, ID_Furnace)
	setTile(x+width-3, floorY-2, ID_Stone)
	setTile(x+width-3, floorY-3, ID_Stone)

	// Table
	setTile(x+7, floorY-1, ID_Planks)
	setTile(x+8, floorY-1, ID_Planks)

	// Chest
	setTile(x+2, floorY-1, ID_Chest)

	// Shelf
	setTile(x+1, floorY-2, ID_Planks)

	// ===== ROOF =====
	// Proper triangular roof - filled solid with hollow attic
	roofBaseY := mainCeiling - 1
	roofPeakHeight := 5
	roofOverhang := 2

	for level := 0; level <= roofPeakHeight; level++ {
		roofY := roofBaseY - level
		// Calculate width at this level
		leftX := x - roofOverhang + level
		rightX := x + width - 1 + roofOverhang - level

		if leftX > rightX {
			break
		}

		for rx := leftX; rx <= rightX; rx++ {
			if level == 0 {
				// Bottom row of roof - solid
				setTile(rx, roofY, ID_Log)
			} else if rx == leftX || rx == rightX {
				// Edges of roof
				setTile(rx, roofY, ID_Log)
			} else if leftX+1 >= rightX {
				// Peak
				setTile(rx, roofY, ID_Log)
			} else {
				// Interior - clear for attic
				setTile(rx, roofY, 0)
			}
		}
	}

	// ===== CHIMNEY =====
	chimneyX := x + width - 3
	chimneyBaseY := roofBaseY - roofPeakHeight
	for cy := roofBaseY; cy >= chimneyBaseY-2; cy-- {
		setTile(chimneyX, cy, ID_Stone)
	}

	// ===== FRONT PATH =====
	// Stone path leading to door
	for px := doorX - 3; px < doorX; px++ {
		setTile(px, floorY, ID_Stone)
	}

	// ===== GARDEN =====
	gardenX := x + width + 1
	for gx := gardenX; gx < gardenX+3 && gx < tm.Cols; gx++ {
		// Keep existing ground tile, just add flowers on top
		if rand.Float64() < 0.6 {
			setTile(gx, floorY-1, ID_Flower)
		}
	}

	// ===== FENCE =====
	// Simple fence around property
	fenceLeft := x - 2
	fenceRight := x + width + 4
	// Left fence post
	setTile(fenceLeft, floorY-1, ID_Log)
	setTile(fenceLeft, floorY-2, ID_Log)
	// Right fence post
	setTile(fenceRight, floorY-1, ID_Log)
	setTile(fenceRight, floorY-2, ID_Log)
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
