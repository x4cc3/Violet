package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

type GameState int

const (
	StateIntro GameState = iota
	StateMenu
	StatePlaying
	StatePaused
	StateDeath
)

// Intro
type IntroScreen struct {
	timer           int
	titleY          float64
	pressStartBlink int
	logoScale       float64
	fadeAlpha       float64
}

func NewIntroScreen() *IntroScreen {
	return &IntroScreen{
		timer:     0,
		titleY:    -100,
		logoScale: 0.5,
		fadeAlpha: 1.0,
	}
}

func (is *IntroScreen) Update() GameState {
	is.timer++

	// Animate title dropping in
	targetY := float64(ScreenHeight) / 3
	is.titleY += (targetY - is.titleY) * 0.05

	// Animate logo
	targetScale := 1.0
	is.logoScale += (targetScale - is.logoScale) * 0.03

	// Fade in
	if is.fadeAlpha > 0 {
		is.fadeAlpha -= 0.02
		if is.fadeAlpha < 0 {
			is.fadeAlpha = 0
		}
	}

	// Blink prompt
	is.pressStartBlink++

	// Allow skip
	if is.timer > 60 {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			return StateMenu
		}
	}

	return StateIntro
}

func (is *IntroScreen) Draw(screen *ebiten.Image) {
	// Dark gradient background
	screen.Fill(color.RGBA{15, 10, 25, 255})

	// Animated stars/particles in background
	for i := 0; i < 50; i++ {
		starX := float32((i*73 + is.timer/2) % ScreenWidth)
		starY := float32((i*47 + is.timer/3) % ScreenHeight)
		brightness := uint8(100 + (i*7)%155)
		size := float32(1 + (i % 3))
		vector.DrawFilledCircle(screen, starX, starY, size, color.RGBA{brightness, brightness, brightness, 200}, false)
	}

	// Title text with glow effect
	face := basicfont.Face7x13
	title := "VIOLET"
	subtitle := "A Terraria-Style Adventure"

	// Draw glow behind title
	titleWidth := len(title) * 7 * 4 // Approximate width
	glowX := float32(ScreenWidth/2 - titleWidth/2 - 20)
	glowY := float32(is.titleY - 20)
	glowW := float32(titleWidth + 40)
	glowH := float32(60)

	// Purple glow
	for i := 0; i < 5; i++ {
		alpha := uint8(30 - i*5)
		expand := float32(i * 5)
		vector.DrawFilledRect(screen, glowX-expand, glowY-expand, glowW+expand*2, glowH+expand*2,
			color.RGBA{128, 0, 255, alpha}, false)
	}

	// Draw title (scaled effect using multiple draws)
	drawScaledText(screen, title, ScreenWidth/2, int(is.titleY), is.logoScale*4, face, color.RGBA{200, 100, 255, 255})

	// Subtitle
	subtitleWidth := len(subtitle) * 7
	text.Draw(screen, subtitle, face, ScreenWidth/2-subtitleWidth/2, int(is.titleY)+50, color.RGBA{150, 150, 200, 255})

	// "Press Enter to Start" - blinking
	if is.timer > 90 && (is.pressStartBlink/30)%2 == 0 {
		prompt := "Press ENTER to Start"
		promptWidth := len(prompt) * 7
		text.Draw(screen, prompt, face, ScreenWidth/2-promptWidth/2, ScreenHeight*2/3, color.White)
	}

	// Credits at bottom
	credits := "Made with Ebitengine"
	creditsWidth := len(credits) * 7
	text.Draw(screen, credits, face, ScreenWidth/2-creditsWidth/2, ScreenHeight-30, color.RGBA{100, 100, 100, 255})

	// Fade overlay
	if is.fadeAlpha > 0 {
		alpha := uint8(is.fadeAlpha * 255)
		vector.DrawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.RGBA{0, 0, 0, alpha}, false)
	}
}

// Main menu
type MenuScreen struct {
	selectedOption int
	options        []string
	animTimer      int
}

func NewMenuScreen() *MenuScreen {
	return &MenuScreen{
		selectedOption: 0,
		options:        []string{"Start Game", "Controls", "Quit"},
		animTimer:      0,
	}
}

func (ms *MenuScreen) Update() (GameState, bool) {
	ms.animTimer++

	// Navigate menu
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ms.selectedOption--
		if ms.selectedOption < 0 {
			ms.selectedOption = len(ms.options) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		ms.selectedOption++
		if ms.selectedOption >= len(ms.options) {
			ms.selectedOption = 0
		}
	}

	// Select
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch ms.selectedOption {
		case 0: // Start Game
			return StatePlaying, false
		case 1: // Controls - just show for now, could add a controls screen
			return StateMenu, false
		case 2: // Quit
			return StateMenu, true // Signal to quit
		}
	}

	return StateMenu, false
}

func (ms *MenuScreen) Draw(screen *ebiten.Image) {
	// Background
	screen.Fill(color.RGBA{20, 15, 30, 255})

	// Animated pattern
	for i := 0; i < 30; i++ {
		x := float32((i*97+ms.animTimer)%(ScreenWidth+100) - 50)
		y := float32((i * 61) % ScreenHeight)
		size := float32(2 + math.Sin(float64(ms.animTimer+i*10)/30)*1)
		vector.DrawFilledCircle(screen, x, y, size, color.RGBA{80, 40, 120, 100}, false)
	}

	face := basicfont.Face7x13

	// Title
	title := "VIOLET"
	titleWidth := len(title) * 7 * 3
	drawScaledText(screen, title, ScreenWidth/2, 120, 3, face, color.RGBA{200, 100, 255, 255})
	_ = titleWidth

	// Menu options
	startY := ScreenHeight / 2
	for i, option := range ms.options {
		y := startY + i*50

		optionWidth := len(option) * 7
		x := ScreenWidth/2 - optionWidth/2

		// Highlight selected
		if i == ms.selectedOption {
			// Selection box
			boxPadding := 10
			wave := math.Sin(float64(ms.animTimer)/10) * 3
			vector.DrawFilledRect(screen,
				float32(x-boxPadding)+float32(wave), float32(y-15),
				float32(optionWidth+boxPadding*2), 25,
				color.RGBA{128, 50, 180, 200}, false)
			vector.StrokeRect(screen,
				float32(x-boxPadding)+float32(wave), float32(y-15),
				float32(optionWidth+boxPadding*2), 25,
				2, color.RGBA{200, 150, 255, 255}, false)

			// Arrow
			text.Draw(screen, ">", face, x-20+int(wave), y, color.RGBA{255, 200, 100, 255})
			text.Draw(screen, option, face, x, y, color.White)
		} else {
			text.Draw(screen, option, face, x, y, color.RGBA{150, 150, 150, 255})
		}
	}

	// Controls hint
	hint := "Arrow Keys: Navigate | Enter: Select"
	hintWidth := len(hint) * 7
	text.Draw(screen, hint, face, ScreenWidth/2-hintWidth/2, ScreenHeight-50, color.RGBA{100, 100, 120, 255})
}

// Pause menu
type PauseScreen struct {
	selectedOption int
	options        []string
}

func NewPauseScreen() *PauseScreen {
	return &PauseScreen{
		selectedOption: 0,
		options:        []string{"Resume", "Restart", "Quit to Menu"},
	}
}

func (ps *PauseScreen) Update() (GameState, bool) {
	// ESC resume
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return StatePlaying, false
	}

	// Navigate
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ps.selectedOption--
		if ps.selectedOption < 0 {
			ps.selectedOption = len(ps.options) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		ps.selectedOption++
		if ps.selectedOption >= len(ps.options) {
			ps.selectedOption = 0
		}
	}

	// Select
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch ps.selectedOption {
		case 0: // Resume
			return StatePlaying, false
		case 1: // Restart - signal restart
			return StatePlaying, true
		case 2: // Quit to Menu
			return StateMenu, false
		}
	}

	return StatePaused, false
}

func (ps *PauseScreen) Draw(screen *ebiten.Image, gameScreen *ebiten.Image) {
	// Dimmed game
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(0.3, 0.3, 0.3, 1)
	screen.DrawImage(gameScreen, op)

	// Overlay
	vector.DrawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.RGBA{0, 0, 0, 150}, false)

	face := basicfont.Face7x13

	// Paused title
	title := "PAUSED"
	titleWidth := len(title) * 7 * 2
	drawScaledText(screen, title, ScreenWidth/2, ScreenHeight/3, 2, face, color.RGBA{255, 255, 255, 255})
	_ = titleWidth

	// Menu options
	startY := ScreenHeight / 2
	for i, option := range ps.options {
		y := startY + i*40
		optionWidth := len(option) * 7
		x := ScreenWidth/2 - optionWidth/2

		if i == ps.selectedOption {
			vector.DrawFilledRect(screen, float32(x-10), float32(y-15), float32(optionWidth+20), 25, color.RGBA{100, 50, 150, 200}, false)
			text.Draw(screen, ">", face, x-15, y, color.RGBA{255, 200, 100, 255})
			text.Draw(screen, option, face, x, y, color.White)
		} else {
			text.Draw(screen, option, face, x, y, color.RGBA{150, 150, 150, 255})
		}
	}
}

// Death screen
type DeathScreen struct {
	timer          int
	fadeIn         float64
	selectedOption int
	options        []string
	killCount      int
}

func NewDeathScreen(killCount int) *DeathScreen {
	return &DeathScreen{
		timer:          0,
		fadeIn:         0,
		selectedOption: 0,
		options:        []string{"Respawn", "Quit to Menu"},
		killCount:      killCount,
	}
}

func (ds *DeathScreen) Update() (GameState, bool) {
	ds.timer++

	// Fade in
	if ds.fadeIn < 1 {
		ds.fadeIn += 0.02
		if ds.fadeIn > 1 {
			ds.fadeIn = 1
		}
	}

	// Wait for input
	if ds.timer < 60 {
		return StateDeath, false
	}

	// Navigate
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ds.selectedOption--
		if ds.selectedOption < 0 {
			ds.selectedOption = len(ds.options) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		ds.selectedOption++
		if ds.selectedOption >= len(ds.options) {
			ds.selectedOption = 0
		}
	}

	// Select
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch ds.selectedOption {
		case 0: // Respawn
			return StatePlaying, true // true = restart
		case 1: // Quit to Menu
			return StateMenu, false
		}
	}

	return StateDeath, false
}

func (ds *DeathScreen) Draw(screen *ebiten.Image) {
	// Red overlay
	alpha := uint8(ds.fadeIn * 200)
	vector.DrawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, color.RGBA{30, 0, 0, alpha}, false)

	face := basicfont.Face7x13

	// Died title shake
	shake := 0.0
	if ds.timer < 30 {
		shake = math.Sin(float64(ds.timer)*0.5) * float64(30-ds.timer) / 5
	}

	title := "YOU DIED"
	drawScaledText(screen, title, ScreenWidth/2+int(shake), ScreenHeight/3, 3, face, color.RGBA{200, 50, 50, uint8(ds.fadeIn * 255)})

	// Stats
	if ds.timer > 30 {
		statsY := ScreenHeight/2 - 30
		stats := []string{
			"Slimes Defeated: " + intToString(ds.killCount),
		}
		for i, stat := range stats {
			statWidth := len(stat) * 7
			text.Draw(screen, stat, face, ScreenWidth/2-statWidth/2, statsY+i*20, color.RGBA{200, 200, 200, uint8(ds.fadeIn * 255)})
		}
	}

	// Options
	if ds.timer > 60 {
		startY := ScreenHeight*2/3 - 20
		for i, option := range ds.options {
			y := startY + i*35
			optionWidth := len(option) * 7
			x := ScreenWidth/2 - optionWidth/2

			if i == ds.selectedOption {
				text.Draw(screen, "> "+option+" <", face, x-14, y, color.RGBA{255, 100, 100, 255})
			} else {
				text.Draw(screen, option, face, x, y, color.RGBA{150, 100, 100, 255})
			}
		}
	}
}

// Draw scaled text
func drawScaledText(screen *ebiten.Image, s string, cx, y int, scale float64, face font.Face, clr color.Color) {
	if scale <= 0 {
		return
	}

	// Text dims
	charWidth := 7   // basicfont character width
	charHeight := 13 // basicfont character height
	textWidth := len(s) * charWidth

	// Temp image
	textImg := ebiten.NewImage(textWidth+4, charHeight+4)

	// Draw text
	text.Draw(textImg, s, face, 2, charHeight, clr)

	// Scaled dims
	scaledWidth := float64(textWidth) * scale
	scaledHeight := float64(charHeight) * scale

	// Shadow
	shadowOp := &ebiten.DrawImageOptions{}
	shadowOp.GeoM.Scale(scale, scale)
	shadowOp.GeoM.Translate(float64(cx)-scaledWidth/2+3, float64(y)-scaledHeight+3)
	shadowOp.ColorM.Scale(0, 0, 0, 0.5)
	shadowOp.Filter = ebiten.FilterLinear // Smooth scaling
	screen.DrawImage(textImg, shadowOp)

	// Main text
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(cx)-scaledWidth/2, float64(y)-scaledHeight)
	op.Filter = ebiten.FilterLinear // Smooth scaling
	screen.DrawImage(textImg, op)
}

// Draw large text
func drawLargeText(screen *ebiten.Image, s string, cx, y int, size int, face font.Face, clr color.Color) {
	// Render then scale
	charWidth := 7
	charHeight := 13
	textWidth := len(s) * charWidth

	// Create buffer
	buf := ebiten.NewImage(textWidth+4, charHeight+4)
	text.Draw(buf, s, face, 2, charHeight, clr)

	// Scale
	scale := float64(size) / float64(charHeight)
	scaledWidth := float64(textWidth) * scale

	// Draw with smooth scaling
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(cx)-scaledWidth/2, float64(y))
	op.Filter = ebiten.FilterLinear // Smooth scaling
	screen.DrawImage(buf, op)
}

// Int to string
func intToString(n int) string {
	// Zero
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
