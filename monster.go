package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type MonsterType int

const (
	MonsterSlime MonsterType = iota
)

type MonsterState int

const (
	MStateIdle MonsterState = iota
	MStateJump
	MStateFall
	MStateLand
)

type Monster struct {
	Type  MonsterType
	State MonsterState

	X, Y   float64
	VX, VY float64

	// Stats
	Health    int
	MaxHealth int
	Damage    int

	Speed        float64
	JumpStrength float64

	// Animation
	IdleSheet *ebiten.Image
	JumpSheet *ebiten.Image

	CurrentSheet *ebiten.Image

	FrameWidth   int
	FrameHeight  int
	TotalFrames  int
	CurrentFrame int
	FrameDelay   int
	FrameCounter int

	FacingRight bool

	// AI
	JumpTimer int

	// Combat
	InvincibleTimer int
	KnockbackVX     float64
}

const (
	SlimeGreen = iota
	SlimeBlue
	SlimeRed
)

func NewSlime(x, y float64, variant int) *Monster {
	var idlePath, jumpPath string
	var health, damage int
	var speedX, jumpY float64

	switch variant {
	case SlimeBlue:
		idlePath = "assets/images/monsters/blue_slime/idle.png"
		jumpPath = "assets/images/monsters/blue_slime/jump.png"
		health = 15
		damage = 8
		speedX = 3.5 // Faster
		jumpY = -7.0
	case SlimeRed:
		idlePath = "assets/images/monsters/red_slime/idle.png"
		jumpPath = "assets/images/monsters/red_slime/jump.png"
		health = 40 // Tanky
		damage = 15
		speedX = 1.5 // Slower
		jumpY = -5.0
	default: // Green
		idlePath = "assets/images/monsters/green_slime/idle.png"
		jumpPath = "assets/images/monsters/green_slime/jump.png"
		health = 20
		damage = 10
		speedX = 2.0
		jumpY = -6.0
	}

	idle := loadImage(idlePath)
	jump := loadImage(jumpPath)

	m := &Monster{
		Type:         MonsterSlime,
		State:        MStateIdle,
		X:            x,
		Y:            y,
		Health:       health,
		MaxHealth:    health,
		Damage:       damage,
		Speed:        speedX,
		JumpStrength: jumpY,
		IdleSheet:    idle,
		JumpSheet:    jump,
		CurrentSheet: idle,
		FrameWidth:   128,
		FrameHeight:  128,
		TotalFrames:  8,
		FrameDelay:   6,
		FacingRight:  true,

		// Variant stats
		// Hardcode for now
		// Add fields later
	}
	// Temp storage
	// Use struct fields
	return m
}

func (m *Monster) Update(g *Game) {
	if m.Health <= 0 {
		return // Dead
	}

	// Cull distant AI
	distToCamera := math.Abs(m.X - g.cameraX - ScreenWidth/2)
	isNearCamera := distToCamera < 1500 // Update near camera

	if m.InvincibleTimer > 0 {
		m.InvincibleTimer--
	}

	// Knockback decay
	if m.KnockbackVX != 0 {
		m.VX = m.KnockbackVX
		m.KnockbackVX *= 0.9
		if math.Abs(m.KnockbackVX) < 0.5 {
			m.KnockbackVX = 0
		}
	} else if isNearCamera { // Near camera AI

		// AI hop
		distX := (g.x + float64(g.frameWidth)/2) - (m.X + float64(m.FrameWidth)/2)
		distY := (g.y + float64(g.frameHeight)/2) - (m.Y + float64(m.FrameHeight)/2)
		dist := math.Sqrt(distX*distX + distY*distY)

		if m.State == MStateIdle {
			m.VX = 0 // Reset VX idle
			m.JumpTimer++
			if m.JumpTimer > 60 && dist < 500 { // Jump near player
				m.JumpTimer = 0
				m.State = MStateJump
				m.VY = m.JumpStrength
				m.CurrentSheet = m.JumpSheet
				m.TotalFrames = 8
				m.CurrentFrame = 0

				// Jump to player
				if distX > 0 {
					m.VX = m.Speed
					m.FacingRight = true
				} else {
					m.VX = -m.Speed
					m.FacingRight = false
				}

				if dist < 300 {
					g.soundMutex.RLock()
					if g.sounds != nil && g.sounds["slime"] != nil {
						g.sounds["slime"].Rewind()
						g.sounds["slime"].Play()
					}
					g.soundMutex.RUnlock()
				}
			}
		}
		// Keep VX in jump
	}

	// Physics
	m.VY += 0.25 // Gravity

	// Horizontal move
	m.X += m.VX
	if isNearCamera { // Near collisions
		m.handleCollisionsX(g.tilemap)
	}

	// Vertical move
	m.Y += m.VY
	m.handleCollisionsY(g.tilemap) // Always vertical

	// Animation
	if isNearCamera {
		m.FrameCounter++
		if m.FrameCounter >= m.FrameDelay {
			m.FrameCounter = 0
			m.CurrentFrame++
			if m.CurrentFrame >= m.TotalFrames {
				m.CurrentFrame = 0
			}
		}
	}
}

func (m *Monster) getMonsterBodyHitbox() image.Rectangle {
	x1 := int(m.X) + MonsterBodyHitboxPaddingX
	y1 := int(m.Y) + MonsterBodyHitboxPaddingY
	x2 := int(m.X) + m.FrameWidth - MonsterBodyHitboxPaddingX
	y2 := int(m.Y) + m.FrameHeight

	// Validate to prevent inverted hitboxes
	if x1 > x2 {
		x1 = x2
	}
	if y1 > y2 {
		y1 = y2
	}

	return image.Rect(x1, y1, x2, y2)
}

func (m *Monster) TakeDamage(amount int, knockbackX float64) {
	if m.InvincibleTimer > 0 {
		return
	}
	m.Health -= amount
	m.InvincibleTimer = 30 // Invincibility
	m.KnockbackVX = knockbackX
	m.VY = -4 // Pop up
}

func (m *Monster) handleCollisionsX(tm *Tilemap) {
	paddingX := 48.0
	paddingTop := 80.0

	left := int((m.X + paddingX) / float64(tm.TileSize))
	right := int((m.X + float64(m.FrameWidth) - paddingX) / float64(tm.TileSize))
	top := int((m.Y + paddingTop) / float64(tm.TileSize))
	bottom := int((m.Y + float64(m.FrameHeight) - 1) / float64(tm.TileSize))

	if m.VX < 0 { // Moving Left
		if tm.IsSolid(tm.GetTile(left, top)) || tm.IsSolid(tm.GetTile(left, bottom)) {
			m.X = float64(left+1)*float64(tm.TileSize) - paddingX
			m.VX = 0
		}
	} else if m.VX > 0 { // Moving Right
		if tm.IsSolid(tm.GetTile(right, top)) || tm.IsSolid(tm.GetTile(right, bottom)) {
			m.X = float64(right)*float64(tm.TileSize) - float64(m.FrameWidth) + paddingX
			m.VX = 0
		}
	}
}

func (m *Monster) handleCollisionsY(tm *Tilemap) {
	paddingX := 48.0
	paddingTop := 80.0

	left := int((m.X + paddingX) / float64(tm.TileSize))
	right := int((m.X + float64(m.FrameWidth) - paddingX) / float64(tm.TileSize))
	top := int((m.Y + paddingTop) / float64(tm.TileSize))
	bottom := int((m.Y + float64(m.FrameHeight) - 1) / float64(tm.TileSize))

	// Ground snap
	bottomFeet := int((m.Y + float64(m.FrameHeight)) / float64(tm.TileSize))

	if m.VY < 0 { // Moving up
		if tm.IsSolid(tm.GetTile(left, top)) || tm.IsSolid(tm.GetTile(right, top)) {
			m.Y = float64(top+1)*float64(tm.TileSize) - paddingTop
			m.VY = 0
		}
	} else if m.VY > 0 { // Moving down
		if tm.IsSolid(tm.GetTile(left, bottom)) || tm.IsSolid(tm.GetTile(right, bottom)) {
			m.Y = float64(bottom)*float64(tm.TileSize) - float64(m.FrameHeight)
			m.VY = 0

			// Land
			if m.State == MStateJump {
				m.State = MStateIdle
				m.CurrentSheet = m.IdleSheet
				m.TotalFrames = 8
			}
		} else if tm.IsSolid(tm.GetTile(left, bottomFeet)) || tm.IsSolid(tm.GetTile(right, bottomFeet)) {
			// Snap to ground
			m.Y = float64(bottomFeet)*float64(tm.TileSize) - float64(m.FrameHeight)
			m.VY = 0
			if m.State == MStateJump {
				m.State = MStateIdle
				m.CurrentSheet = m.IdleSheet
				m.TotalFrames = 8
			}
		}
	}
}

// Reusable opts
var monsterDrawOpts = &ebiten.DrawImageOptions{}

func (m *Monster) Draw(screen *ebiten.Image, cameraX, cameraY float64) {
	if m.Health <= 0 {
		return
	}

	// Cull off-screen
	screenX := m.X - cameraX
	screenY := m.Y - cameraY
	if screenX < -float64(m.FrameWidth) || screenX > ScreenWidth+float64(m.FrameWidth) ||
		screenY < -float64(m.FrameHeight) || screenY > ScreenHeight+float64(m.FrameHeight) {
		return
	}

	// Flicker invincible
	if m.InvincibleTimer > 0 && (m.InvincibleTimer/4)%2 == 0 {
		return
	}

	// Reuse opts
	monsterDrawOpts.GeoM.Reset()

	if !m.FacingRight {
		monsterDrawOpts.GeoM.Scale(-1, 1)
		monsterDrawOpts.GeoM.Translate(float64(m.FrameWidth), 0)
	}

	monsterDrawOpts.GeoM.Translate(screenX, screenY)

	if m.CurrentSheet == nil {
		return
	}

	sx := m.CurrentFrame * m.FrameWidth
	sy := 0

	// Bounds check
	if sx < 0 || sy < 0 || sx+m.FrameWidth > m.CurrentSheet.Bounds().Dx() || sy+m.FrameHeight > m.CurrentSheet.Bounds().Dy() {
		return
	}

	rect := image.Rect(sx, sy, sx+m.FrameWidth, sy+m.FrameHeight)
	screen.DrawImage(m.CurrentSheet.SubImage(rect).(*ebiten.Image), monsterDrawOpts)
}
