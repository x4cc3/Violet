package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 1280
	ScreenHeight = 720

	// Hitbox padding
	MonsterBodyHitboxPaddingX = 48
	MonsterBodyHitboxPaddingY = 80
)

type Game struct {
	// Game state
	gameState   GameState
	introScreen *IntroScreen
	menuScreen  *MenuScreen
	pauseScreen *PauseScreen
	deathScreen *DeathScreen

	// Pause buffer
	gameBuffer *ebiten.Image

	// Assets
	idleSpriteSheet       *ebiten.Image
	walkSpriteSheet       *ebiten.Image
	attackSpriteSheet     *ebiten.Image
	protectionSpriteSheet *ebiten.Image
	dialogueSpriteSheet   *ebiten.Image
	jumpSpriteSheet       *ebiten.Image
	fallSpriteSheet       *ebiten.Image
	currentSpriteSheet    *ebiten.Image

	// Background
	bgImage *ebiten.Image

	// Systems
	tilemap *Tilemap
	ui      *UI

	// Animation
	frameWidth, frameHeight                             int
	totalFrames, currentFrame, frameCounter, frameDelay int
	currentState                                        AnimationState

	// Physics
	x, y       float64
	vx, vy     float64
	speed      float64
	direction  float64
	isGrounded bool

	// Actions
	isAttacking, isProtecting, isDialogue bool

	// Camera
	cameraX, cameraY float64

	// Debug
	showDebug   bool
	showPalette bool

	coyoteTimer     int
	jumpBufferTimer int

	dialogueSystem *DialogueSystem

	// Monsters
	monsters []*Monster

	// Player stats
	PlayerHealth          int
	PlayerMaxHealth       int
	PlayerInvincibleTimer int

	// Audio
	audioContext      *audio.Context
	sounds            map[string]*audio.Player
	attackSoundPlayed bool
	isRunningPlaying  bool
	bgMusicPlaying    bool
	audioEnabled      bool // Toggle with M key
}

// Update game logic
func (g *Game) Update() error {
	switch g.gameState {
	case StateIntro:
		newState := g.introScreen.Update()
		if newState != StateIntro {
			g.gameState = newState
			g.menuScreen = NewMenuScreen()
		}

	case StateMenu:
		newState, shouldQuit := g.menuScreen.Update()
		if shouldQuit {
			return ebiten.Termination
		}
		if newState != StateMenu {
			g.gameState = newState
			if newState == StatePlaying {
				// Start background music
				g.startBackgroundMusic()
				// Show initial dialogue
				if g.dialogueSystem != nil && !g.dialogueSystem.Active {
					g.startIntroDialogue()
				}
			}
		}

	case StatePlaying:
		g.updatePlaying()

		// Check for pause
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.gameState = StatePaused
			g.pauseScreen = NewPauseScreen()
			g.pauseBackgroundMusic()
		}

		// Check for death
		if g.PlayerHealth <= 0 {
			g.gameState = StateDeath
			g.deathScreen = NewDeathScreen(g.ui.killCount)
			g.pauseBackgroundMusic()
		}

	case StatePaused:
		newState, restart := g.pauseScreen.Update()
		if newState != StatePaused {
			g.gameState = newState
			if newState == StatePlaying {
				g.resumeBackgroundMusic()
				if restart {
					g.restartGame()
				}
			} else if newState == StateMenu {
				g.menuScreen = NewMenuScreen()
				g.stopBackgroundMusic()
			}
		}

	case StateDeath:
		newState, restart := g.deathScreen.Update()
		if newState != StateDeath {
			g.gameState = newState
			if newState == StatePlaying && restart {
				g.restartGame()
				g.resumeBackgroundMusic()
			} else if newState == StateMenu {
				g.menuScreen = NewMenuScreen()
				g.stopBackgroundMusic()
			}
		}
	}

	return nil
}

func (g *Game) updatePlaying() {
	g.handleDebugInputs()

	if g.showPalette {
		g.handlePaletteInput()
		return
	}

	g.updateInvincibility()
	g.updatePlayer()
	g.updateRunningSound()
	g.checkChestInteraction()
	g.updateMonstersAndCombat()
	g.updateCamera()

	// Update UI
	playerTileX := int(g.x / float64(g.tilemap.TileSize))
	biome := GetBiomeName(GetBiomeAt(playerTileX))
	g.ui.Update(g.PlayerHealth, g.PlayerMaxHealth, biome)
}

func (g *Game) startBackgroundMusic() {
	if IsEmbedded() || !g.audioEnabled {
		return
	}
	if g.sounds["background"] != nil && !g.bgMusicPlaying {
		g.sounds["background"].Rewind()
		g.sounds["background"].Play()
		g.bgMusicPlaying = true
	}
}

func (g *Game) pauseBackgroundMusic() {
	if g.sounds["background"] != nil {
		g.sounds["background"].Pause()
	}
}

func (g *Game) resumeBackgroundMusic() {
	if g.sounds["background"] != nil && g.bgMusicPlaying {
		g.sounds["background"].Play()
	}
}

func (g *Game) stopBackgroundMusic() {
	if g.sounds["background"] != nil {
		g.sounds["background"].Pause()
		g.sounds["background"].Rewind()
		g.bgMusicPlaying = false
	}
}

func (g *Game) updateRunningSound() {
	if IsEmbedded() || !g.audioEnabled {
		return
	}
	isMoving := g.isGrounded && (g.vx > 0.5 || g.vx < -0.5) && !g.isAttacking && !g.isProtecting

	if isMoving && !g.isRunningPlaying {
		if g.sounds["running"] != nil {
			g.sounds["running"].Rewind()
			g.sounds["running"].Play()
			g.isRunningPlaying = true
		}
	} else if !isMoving && g.isRunningPlaying {
		if g.sounds["running"] != nil {
			g.sounds["running"].Pause()
			g.isRunningPlaying = false
		}
	}

}

func (g *Game) restartGame() {
	// Reset player
	g.x = 1600
	g.y = 0
	g.vx = 0
	g.vy = 0
	g.PlayerHealth = g.PlayerMaxHealth

	// Regenerate world
	g.tilemap.GenerateTerrariaWorld()

	// Respawn monsters
	g.monsters = make([]*Monster, 0)
	mapWidth := g.tilemap.Cols * 16
	for i := 0; i < 15; i++ {
		spawnX := 200 + rand.Intn(mapWidth-400)
		r := rand.Float64()
		variant := SlimeGreen
		if r < 0.2 {
			variant = SlimeRed
		} else if r < 0.5 {
			variant = SlimeBlue
		}
		g.monsters = append(g.monsters, NewSlime(float64(spawnX), 0, variant))
	}

	// Reset UI
	g.ui = NewUI()
}

func (g *Game) startIntroDialogue() {
	g.dialogueSystem.Start([]DialogueLine{
		{Speaker: "???", Text: "Ugh... Where am I? My head is pounding...", Emotion: "sad"},
		{Speaker: "Violet", Text: "Wait... Violet!. That's my name. But how did I get here?", Emotion: "neutral"},
		{Speaker: "Narrator", Text: "You awaken in a mysterious procedurally generated world. Forests, mountains, deserts, and swamps stretch endlessly before you.", Emotion: "neutral"},
		{Speaker: "Violet", Text: "Those slimes... they don't look friendly. I need to defend myself!", Emotion: "angry"},
		{Speaker: "Tutorial", Text: "WASD or Arrow Keys to move. SPACE to jump. ENTER to attack. SHIFT to block. E to interact.", Emotion: "neutral"},
		{Speaker: "Violet", Text: "Alright! Time to explore and find out what happened to me!", Emotion: "excited"},
	})
}

// Toggle debug modes
func (g *Game) handleDebugInputs() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.showDebug = !g.showDebug
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.showPalette = !g.showPalette
	}
	// Toggle audio with M
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.audioEnabled = !g.audioEnabled
		if g.audioEnabled {
			log.Println("Audio ENABLED")
			g.startBackgroundMusic()
		} else {
			log.Println("Audio DISABLED")
			g.stopBackgroundMusic()
			if g.sounds["running"] != nil {
				g.sounds["running"].Pause()
			}
			g.isRunningPlaying = false
		}
	}
}

// Handle palette clicks
func (g *Game) handlePaletteInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		tileSize := 16
		padding := 1
		startX, startY := 20, 20

		relX := mx - startX
		relY := my - startY

		if relX >= 0 && relY >= 0 {
			col := relX / (tileSize + padding)
			row := relY / (tileSize + padding)

			inTileX := relX % (tileSize + padding)
			inTileY := relY % (tileSize + padding)

			if col < 24 && inTileX < tileSize && inTileY < tileSize {
				id := row*24 + col
				log.Printf("CLICKED TILE ID: %d", id)
			}
		}
	}
}

// Update invincibility
func (g *Game) updateInvincibility() {
	if g.PlayerInvincibleTimer > 0 {
		g.PlayerInvincibleTimer--
	}
}

// Check if tile breakable
func isBreakable(tileID int) bool {
	return tileID == ID_Stone || tileID == ID_Dirt || tileID == ID_Sand
}

// Break tiles in rect
func (g *Game) breakTilesInRect(rect image.Rectangle) {
	tileSize := g.tilemap.TileSize
	startX := rect.Min.X / tileSize
	endX := (rect.Max.X + tileSize - 1) / tileSize
	startY := rect.Min.Y / tileSize
	endY := (rect.Max.Y + tileSize - 1) / tileSize

	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			if x >= 0 && x < g.tilemap.Cols && y >= 0 && y < g.tilemap.Rows {
				tileID := g.tilemap.Grid[y][x]
				if isBreakable(tileID) {
					g.tilemap.Grid[y][x] = 0 // Break to air
				}
			}
		}
	}
}

// Handle chest interaction
func (g *Game) checkChestInteraction() {
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		playerTileX := int(g.x / float64(g.tilemap.TileSize))
		playerTileY := int(g.y / float64(g.tilemap.TileSize))
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				tx := playerTileX + dx
				ty := playerTileY + dy
				if tx >= 0 && tx < g.tilemap.Cols && ty >= 0 && ty < g.tilemap.Rows && g.tilemap.Grid[ty][tx] == ID_Chest {
					g.tilemap.Grid[ty][tx] = 0 // Remove chest
					healAmount := 20
					if g.PlayerHealth < g.PlayerMaxHealth {
						g.PlayerHealth += healAmount
						if g.PlayerHealth > g.PlayerMaxHealth {
							g.PlayerHealth = g.PlayerMaxHealth
						}
						g.ui.AddDamageNumber(g.x, g.y-50, healAmount, true)
						g.ui.AddNotification("Found healing herbs! +20 HP")
					} else {
						g.ui.AddNotification("Opened a chest!")
					}
					// Play sound
					if g.sounds["chest"] != nil {
						g.sounds["chest"].Rewind()
						g.sounds["chest"].Play()
					}
					return
				}
			}
		}
	}
}

// Update monsters and combat
func (g *Game) updateMonstersAndCombat() {
	// Remove dead monsters
	aliveMonsters := make([]*Monster, 0)
	for _, m := range g.monsters {
		if m.Health > 0 {
			aliveMonsters = append(aliveMonsters, m)
		}
	}
	g.monsters = aliveMonsters

	for _, m := range g.monsters {
		m.Update(g)

		// Player attack
		if g.isAttacking && g.currentFrame == 2 {
			if !g.attackSoundPlayed {
				if g.sounds["attack"] != nil {
					g.sounds["attack"].Rewind()
					g.sounds["attack"].Play()
				}
				g.attackSoundPlayed = true
			}
			if g.getPlayerAttackHitbox().Overlaps(m.getMonsterBodyHitbox()) {
				knockback := 5.0 * g.direction
				damage := 10
				m.TakeDamage(damage, knockback)
				g.ui.AddDamageNumber(m.X+float64(m.FrameWidth)/2, m.Y, damage, false)

				// Monster died
				if m.Health <= 0 {
					g.ui.AddKill()
					g.ui.AddNotification("Slime defeated!")
				}
			}
		}

		// Monster attack
		if m.Health > 0 {
			if g.getPlayerBodyHitbox().Overlaps(m.getMonsterBodyHitbox()) {
				dir := 1.0
				if m.X > g.x {
					dir = -1.0
				}
				g.PlayerTakeDamage(m.Damage, dir*8.0)
			}
		}
	}
}

func (g *Game) getPlayerAttackHitbox() image.Rectangle {
	if !g.isAttacking || g.currentFrame != 2 {
		return image.Rect(0, 0, 0, 0)
	}
	var x1, x2 int
	if g.direction > 0 {
		x1 = int(g.x)
		x2 = int(g.x) + 40
	} else {
		x1 = int(g.x) - 40
		x2 = int(g.x)
	}
	y1 := int(g.y) - 60
	y2 := int(g.y)
	return image.Rect(x1, y1, x2, y2)
}

func (g *Game) getPlayerBodyHitbox() image.Rectangle {
	w := 20
	h := 64
	if g.isProtecting {
		w /= 2
	}
	x1 := int(g.x) - w/2
	x2 := int(g.x) + w/2
	y1 := int(g.y) - h
	y2 := int(g.y)
	return image.Rect(x1, y1, x2, y2)
}

func (g *Game) PlayerTakeDamage(amount int, knockbackX float64) {
	if g.PlayerInvincibleTimer > 0 {
		return
	}

	if g.isProtecting {
		amount /= 5
		knockbackX /= 2
	}

	g.PlayerHealth -= amount
	g.ui.AddDamageNumber(g.x, g.y-30, amount, false)

	if g.PlayerHealth < 0 {
		g.PlayerHealth = 0
	}

	g.PlayerInvincibleTimer = 60
	g.vx = knockbackX
	g.vy = -5
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.gameState {
	case StateIntro:
		g.introScreen.Draw(screen)

	case StateMenu:
		g.menuScreen.Draw(screen)

	case StatePlaying:
		g.drawGame(screen)

	case StatePaused:
		// Draw game to buffer first
		if g.gameBuffer == nil {
			g.gameBuffer = ebiten.NewImage(ScreenWidth, ScreenHeight)
		}
		g.gameBuffer.Clear()
		g.drawGame(g.gameBuffer)
		g.pauseScreen.Draw(screen, g.gameBuffer)

	case StateDeath:
		g.drawGame(screen)
		g.deathScreen.Draw(screen)
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {
	if g.showPalette {
		g.drawPalette(screen)
		return
	}

	// Draw background
	g.drawBackground(screen)

	// Draw world
	g.tilemap.Draw(screen, g.cameraX, g.cameraY)

	// Draw monsters
	for _, m := range g.monsters {
		m.Draw(screen, g.cameraX, g.cameraY)
	}

	// Draw player
	g.drawPlayer(screen)

	// Draw UI
	g.ui.Draw(screen, g.PlayerHealth, g.PlayerMaxHealth, g.cameraX, g.cameraY)

	// Draw debug
	if g.showDebug {
		g.drawDebug(screen)
	}

	// Draw dialogue
	if g.dialogueSystem != nil {
		g.dialogueSystem.Draw(screen)
	}
}

func (g *Game) drawPalette(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20, 20, 20, 255})

	tileSize := 16
	padding := 1
	startX, startY := 20, 20

	cols := 24

	for i, img := range g.tilemap.TileImages {
		if img == nil {
			continue
		}

		c := i % cols
		r := i / cols

		x := startX + c*(tileSize+padding)
		y := startY + r*(tileSize+padding)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, op)

		mx, my := ebiten.CursorPosition()
		if mx >= x && mx < x+tileSize && my >= y && my < y+tileSize {
			vector.StrokeRect(screen, float32(x), float32(y), float32(tileSize), float32(tileSize), 1, color.RGBA{255, 255, 0, 255}, false)
		}
	}
}

func (g *Game) drawBackground(screen *ebiten.Image) {
	if g.bgImage == nil {
		return
	}

	parallax := 0.5
	scale := float64(ScreenHeight) / float64(g.bgImage.Bounds().Dy())
	imgW := float64(g.bgImage.Bounds().Dx()) * scale

	offset := g.cameraX * parallax

	startPos := -float64(int(offset) % int(imgW))
	if startPos > 0 {
		startPos -= imgW
	}

	op := &ebiten.DrawImageOptions{}
	for x := startPos; x < ScreenWidth; x += imgW {
		op.GeoM.Reset()
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x, 0)
		screen.DrawImage(g.bgImage, op)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) updateCamera() {
	targetX := g.x - float64(ScreenWidth)/2.0
	targetY := g.y - float64(ScreenHeight)/2.0

	mapW := float64(g.tilemap.Cols * g.tilemap.TileSize)
	mapH := float64(g.tilemap.Rows * g.tilemap.TileSize)

	if targetX < 0 {
		targetX = 0
	}
	if targetX > mapW-float64(ScreenWidth) {
		targetX = mapW - float64(ScreenWidth)
	}

	if targetY < 0 {
		targetY = 0
	}
	if targetY > mapH-float64(ScreenHeight) {
		targetY = mapH - float64(ScreenHeight)
	}

	g.cameraX = targetX
	g.cameraY = targetY
}

// Reusable draw opts
var playerDrawOpts = &ebiten.DrawImageOptions{}
var playerFlashOpts = &ebiten.DrawImageOptions{}

func (g *Game) drawPlayer(screen *ebiten.Image) {
	if g.currentSpriteSheet == nil {
		return
	}

	playerDrawOpts.GeoM.Reset()
	playerDrawOpts.ColorScale.Reset()

	if g.direction == -1 {
		playerDrawOpts.GeoM.Scale(-1, 1)
		playerDrawOpts.GeoM.Translate(float64(g.frameWidth), 0)
	}

	playerDrawOpts.GeoM.Translate(g.x-g.cameraX-float64(g.frameWidth)/2, g.y-g.cameraY-float64(g.frameHeight))

	sx := g.currentFrame * g.frameWidth
	sy := 0
	rect := image.Rect(sx, sy, sx+g.frameWidth, sy+g.frameHeight)
	sprite := g.currentSpriteSheet.SubImage(rect).(*ebiten.Image)

	screen.DrawImage(sprite, playerDrawOpts)

	if g.PlayerInvincibleTimer > 0 && (g.PlayerInvincibleTimer/4)%2 == 0 {
		playerFlashOpts.GeoM.Reset()
		playerFlashOpts.ColorScale.Reset()
		if g.direction == -1 {
			playerFlashOpts.GeoM.Scale(-1, 1)
			playerFlashOpts.GeoM.Translate(float64(g.frameWidth), 0)
		}
		playerFlashOpts.GeoM.Translate(g.x-g.cameraX-float64(g.frameWidth)/2, g.y-g.cameraY-float64(g.frameHeight))
		playerFlashOpts.ColorScale.Scale(1.5, 0.25, 0.25, 0.6)
		screen.DrawImage(sprite, playerFlashOpts)
	}
}

func (g *Game) drawDebug(screen *ebiten.Image) {
	audioStatus := "OFF (M to toggle)"
	if g.audioEnabled {
		audioStatus = "ON (M to toggle)"
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f\nTPS: %0.2f\nX: %0.2f\nY: %0.2f\nVY: %0.2f\nGrounded: %v\nState: %d\nMonsters: %d\nAudio: %s", ebiten.CurrentFPS(), ebiten.CurrentTPS(), g.x, g.y, g.vy, g.isGrounded, g.currentState, len(g.monsters), audioStatus))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Violet - A Mystery Adventure")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		if err != ebiten.Termination {
			log.Fatal(err)
		}
	}
}
