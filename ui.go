package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// Pool sizes
const (
	MaxDamageNumbers = 32
	MaxNotifications = 8
)

// UI HUD
type UI struct {
	// Health anim
	displayHealth    float64
	healthFlashTimer int
	lastHealth       int

	// Kill counter
	killCount      int
	killFlashTimer int

	// Biome display
	currentBiome     string
	biomeChangeTimer int

	// Damage numbers
	damageNumbers     [MaxDamageNumbers]DamageNumber
	activeDamageCount int

	// Notifications
	notifications     [MaxNotifications]Notification
	activeNotifyCount int
}

type DamageNumber struct {
	X, Y   float64
	Amount int
	Timer  int
	VY     float64
	IsHeal bool
	Active bool // Pool
}

type Notification struct {
	Text   string
	Timer  int
	FadeIn bool
	Active bool // Pool
}

func NewUI() *UI {
	return &UI{
		displayHealth: 100,
	}
}

func (ui *UI) Update(currentHealth, maxHealth int, biome string) {
	// Smooth health bar animation
	targetHealth := float64(currentHealth)
	ui.displayHealth += (targetHealth - ui.displayHealth) * 0.1

	// Flash when damaged
	if currentHealth < ui.lastHealth {
		ui.healthFlashTimer = 20
	}
	ui.lastHealth = currentHealth

	if ui.healthFlashTimer > 0 {
		ui.healthFlashTimer--
	}

	// Biome change notification
	if biome != ui.currentBiome {
		ui.currentBiome = biome
		ui.biomeChangeTimer = 180 // 3 seconds
	}
	if ui.biomeChangeTimer > 0 {
		ui.biomeChangeTimer--
	}

	// Kill flash
	if ui.killFlashTimer > 0 {
		ui.killFlashTimer--
	}

	// Update damage numbers (in-place, no allocations)
	activeCount := 0
	for i := 0; i < ui.activeDamageCount; i++ {
		dn := &ui.damageNumbers[i]
		if !dn.Active {
			continue
		}
		dn.Timer--
		dn.Y += dn.VY
		dn.VY += 0.1 // Gravity
		if dn.Timer <= 0 {
			dn.Active = false
		} else {
			activeCount++
		}
	}
	// Compact array if many inactive (optional optimization)
	if activeCount < ui.activeDamageCount/2 && ui.activeDamageCount > 8 {
		ui.compactDamageNumbers()
	}

	// Update notifications (in-place, no allocations)
	for i := 0; i < ui.activeNotifyCount; i++ {
		n := &ui.notifications[i]
		if !n.Active {
			continue
		}
		n.Timer--
		if n.Timer <= 0 {
			n.Active = false
		}
	}
}

func (ui *UI) AddDamageNumber(x, y float64, amount int, isHeal bool) {
	// Find an inactive slot or use next slot
	if ui.activeDamageCount < MaxDamageNumbers {
		ui.damageNumbers[ui.activeDamageCount] = DamageNumber{
			X:      x,
			Y:      y,
			Amount: amount,
			Timer:  60,
			VY:     -3,
			IsHeal: isHeal,
			Active: true,
		}
		ui.activeDamageCount++
	} else {
		// Overwrite oldest (first slot)
		ui.damageNumbers[0] = DamageNumber{
			X:      x,
			Y:      y,
			Amount: amount,
			Timer:  60,
			VY:     -3,
			IsHeal: isHeal,
			Active: true,
		}
	}
}

func (ui *UI) AddKill() {
	ui.killCount++
	ui.killFlashTimer = 30
}

func (ui *UI) AddNotification(notificationText string) {
	if ui.activeNotifyCount < MaxNotifications {
		ui.notifications[ui.activeNotifyCount] = Notification{
			Text:   notificationText,
			Timer:  180,
			FadeIn: true,
			Active: true,
		}
		ui.activeNotifyCount++
	} else {
		// Overwrite oldest
		ui.notifications[0] = Notification{
			Text:   notificationText,
			Timer:  180,
			FadeIn: true,
			Active: true,
		}
	}
}

// compactDamageNumbers removes inactive entries (called sparingly)
func (ui *UI) compactDamageNumbers() {
	writeIdx := 0
	for i := 0; i < ui.activeDamageCount; i++ {
		if ui.damageNumbers[i].Active {
			if writeIdx != i {
				ui.damageNumbers[writeIdx] = ui.damageNumbers[i]
			}
			writeIdx++
		}
	}
	ui.activeDamageCount = writeIdx
}

func (ui *UI) Draw(screen *ebiten.Image, currentHealth, maxHealth int, cameraX, cameraY float64) {
	face := basicfont.Face7x13

	// ===== HEALTH BAR =====
	ui.drawHealthBar(screen, currentHealth, maxHealth, face)

	// ===== KILL COUNTER =====
	ui.drawKillCounter(screen, face)

	// ===== BIOME INDICATOR =====
	if ui.biomeChangeTimer > 0 {
		ui.drawBiomeIndicator(screen, face)
	}

	// ===== DAMAGE NUMBERS =====
	ui.drawDamageNumbers(screen, face, cameraX, cameraY)

	// ===== NOTIFICATIONS =====
	ui.drawNotifications(screen, face)

	// ===== CONTROLS HINT =====
	ui.drawControlsHint(screen, face)
}

func (ui *UI) drawHealthBar(screen *ebiten.Image, currentHealth, maxHealth int, face font.Face) {
	barX, barY := 20.0, 20.0
	barW, barH := 220.0, 28.0

	// Background with gradient effect
	vector.DrawFilledRect(screen, float32(barX-2), float32(barY-2), float32(barW+4), float32(barH+4), color.RGBA{0, 0, 0, 200}, false)
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barW), float32(barH), color.RGBA{40, 20, 20, 255}, false)

	// Health fill with gradient
	healthPct := ui.displayHealth / float64(maxHealth)
	if healthPct < 0 {
		healthPct = 0
	}
	if healthPct > 1 {
		healthPct = 1
	}

	fillW := barW * healthPct

	// Color based on health level
	var healthColor color.RGBA
	if healthPct > 0.6 {
		healthColor = color.RGBA{50, 200, 80, 255} // Green
	} else if healthPct > 0.3 {
		healthColor = color.RGBA{220, 180, 50, 255} // Yellow
	} else {
		healthColor = color.RGBA{220, 50, 50, 255} // Red
	}

	// Flash effect when damaged
	if ui.healthFlashTimer > 0 && ui.healthFlashTimer%4 < 2 {
		healthColor = color.RGBA{255, 255, 255, 255}
	}

	// Draw health fill
	vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(fillW), float32(barH), healthColor, false)

	// Health bar segments (like hearts)
	segmentWidth := barW / 10
	for i := 0; i < 10; i++ {
		x := barX + float64(i)*segmentWidth
		vector.StrokeLine(screen, float32(x), float32(barY), float32(x), float32(barY+barH), 1, color.RGBA{0, 0, 0, 100}, false)
	}

	// Border
	vector.StrokeRect(screen, float32(barX), float32(barY), float32(barW), float32(barH), 2, color.RGBA{200, 200, 200, 255}, false)

	// Heart icon
	drawHeart(screen, int(barX)-15, int(barY)+int(barH)/2, healthPct > 0.3)

	// Health text
	healthText := intToString(currentHealth) + "/" + intToString(maxHealth)
	textWidth := len(healthText) * 7
	text.Draw(screen, healthText, face, int(barX)+int(barW)/2-textWidth/2, int(barY)+int(barH)/2+5, color.White)
}

func drawHeart(screen *ebiten.Image, cx, cy int, full bool) {
	// Simple heart shape using circles
	size := 6.0

	var clr color.RGBA
	if full {
		clr = color.RGBA{220, 50, 80, 255}
	} else {
		clr = color.RGBA{100, 30, 40, 255}
	}

	// Two circles for top of heart
	vector.DrawFilledCircle(screen, float32(cx)-float32(size)/2, float32(cy)-float32(size)/3, float32(size)/2, clr, false)
	vector.DrawFilledCircle(screen, float32(cx)+float32(size)/2, float32(cy)-float32(size)/3, float32(size)/2, clr, false)

	// Rectangle for bottom
	vector.DrawFilledRect(screen, float32(cx)-float32(size), float32(cy)-float32(size)/3, float32(size)*2, float32(size), clr, false)
}

func (ui *UI) drawKillCounter(screen *ebiten.Image, face font.Face) {
	x := ScreenWidth - 150
	y := 35

	// Background
	vector.DrawFilledRect(screen, float32(x-10), float32(y-20), 140, 30, color.RGBA{0, 0, 0, 150}, false)
	vector.StrokeRect(screen, float32(x-10), float32(y-20), 140, 30, 1, color.RGBA{100, 100, 100, 200}, false)

	// Skull icon (simplified)
	vector.DrawFilledCircle(screen, float32(x), float32(y-5), 8, color.RGBA{200, 200, 200, 255}, false)

	// Kill count text
	killText := "Kills: " + intToString(ui.killCount)
	if ui.killCount < 7 {
		killText += " / 7"
	}

	textColor := color.RGBA{255, 255, 255, 255}
	if ui.killFlashTimer > 0 {
		textColor = color.RGBA{255, 255, 100, 255}
	}

	text.Draw(screen, killText, face, x+15, y, textColor)
}

func (ui *UI) drawBiomeIndicator(screen *ebiten.Image, face font.Face) {
	// Fade in/out
	alpha := 255
	if ui.biomeChangeTimer > 150 {
		alpha = int((180 - float64(ui.biomeChangeTimer)) / 30 * 255)
	} else if ui.biomeChangeTimer < 30 {
		alpha = int(float64(ui.biomeChangeTimer) / 30 * 255)
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 255 {
		alpha = 255
	}

	biomeText := "~ " + ui.currentBiome + " ~"
	textWidth := len(biomeText) * 7
	x := ScreenWidth/2 - textWidth/2
	y := 80

	// Background
	vector.DrawFilledRect(screen, float32(x-20), float32(y-15), float32(textWidth+40), 25, color.RGBA{0, 0, 0, uint8(alpha / 2)}, false)

	// Biome-specific color
	var biomeColor color.RGBA
	switch ui.currentBiome {
	case "Plains":
		biomeColor = color.RGBA{100, 200, 100, uint8(alpha)}
	case "Forest":
		biomeColor = color.RGBA{50, 150, 50, uint8(alpha)}
	case "Mountains":
		biomeColor = color.RGBA{150, 150, 180, uint8(alpha)}
	case "Desert":
		biomeColor = color.RGBA{220, 180, 100, uint8(alpha)}
	case "Swamp":
		biomeColor = color.RGBA{100, 130, 80, uint8(alpha)}
	default:
		biomeColor = color.RGBA{200, 200, 200, uint8(alpha)}
	}

	text.Draw(screen, biomeText, face, x, y, biomeColor)
}

func (ui *UI) drawDamageNumbers(screen *ebiten.Image, face font.Face, cameraX, cameraY float64) {
	for i := 0; i < ui.activeDamageCount; i++ {
		dn := &ui.damageNumbers[i]
		if !dn.Active {
			continue
		}

		screenX := int(dn.X - cameraX)
		screenY := int(dn.Y - cameraY)

		// Fade out
		alpha := uint8(255 * dn.Timer / 60)

		var clr color.RGBA
		var prefix string
		if dn.IsHeal {
			clr = color.RGBA{100, 255, 100, alpha}
			prefix = "+"
		} else {
			clr = color.RGBA{255, 100, 100, alpha}
			prefix = "-"
		}

		dmgText := prefix + intToString(dn.Amount)
		text.Draw(screen, dmgText, face, screenX, screenY, clr)
	}
}

func (ui *UI) drawNotifications(screen *ebiten.Image, face font.Face) {
	startY := ScreenHeight - 100
	drawnCount := 0
	for i := 0; i < ui.activeNotifyCount; i++ {
		n := &ui.notifications[i]
		if !n.Active {
			continue
		}

		y := startY - drawnCount*25
		drawnCount++

		// Fade based on timer
		alpha := 255
		if n.Timer < 30 {
			alpha = int(float64(n.Timer) / 30 * 255)
		}

		textWidth := len(n.Text) * 7
		x := ScreenWidth/2 - textWidth/2

		text.Draw(screen, n.Text, face, x, y, color.RGBA{255, 255, 200, uint8(alpha)})
	}
}

func (ui *UI) drawControlsHint(screen *ebiten.Image, face font.Face) {
	hints := "A/D: Move | Space: Jump | Enter: Attack | Shift: Block | E: Interact | ESC: Pause"
	textWidth := len(hints) * 7
	x := ScreenWidth/2 - textWidth/2
	y := ScreenHeight - 20

	text.Draw(screen, hints, face, x, y, color.RGBA{100, 100, 100, 150})
}

// GetBiomeName returns the name of a biome by ID
func GetBiomeName(biomeID int) string {
	switch biomeID {
	case BiomePlains:
		return "Plains"
	case BiomeForest:
		return "Forest"
	case BiomeMountains:
		return "Mountains"
	case BiomeDesert:
		return "Desert"
	case BiomeSwamp:
		return "Swamp"
	default:
		return "Unknown"
	}
}
