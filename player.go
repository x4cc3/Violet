package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Physics
const (
	Gravity       = 0.5
	JumpStrength  = -15.0
	MaxFallSpeed  = 14.0
	PlayerHitboxW = 20
	PlayerHitboxH = 64

	// Movement
	GroundAccel    = 0.8
	AirAccel       = 0.5
	GroundFriction = 0.85
	AirFriction    = 0.95
	MaxSpeedX      = 6.0

	// Jump
	JumpCutMultiplier = 1.0
	FallGravityMult   = 1.4
	CoyoteFrames      = 6
	JumpBufferFrames  = 8
)

type AnimationState int

const (
	StateIdle AnimationState = iota
	StateWalk
	StateAttack
	StateProtection
	StateDialogue
	StateJump
	StateFall
)

func (g *Game) updatePlayer() {
	// Dialogue check
	if g.dialogueSystem != nil && g.dialogueSystem.Active {
		g.dialogueSystem.Update()
		return
	}

	g.handleActionStates()

	// Coyote
	if g.isGrounded {
		g.coyoteTimer = CoyoteFrames
	} else if g.coyoteTimer > 0 {
		g.coyoteTimer--
	}

	// Buffer
	if g.jumpBufferTimer > 0 {
		g.jumpBufferTimer--
	}

	// Move X
	inputX := 0.0
	if !g.isAttacking && !g.isProtecting && !g.isDialogue {
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			inputX = -1
			g.direction = -1
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
			inputX = 1
			g.direction = 1
		}

		// Buffer jump
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.jumpBufferTimer = JumpBufferFrames
		}

		// Execute jump
		if g.jumpBufferTimer > 0 && g.coyoteTimer > 0 {
			g.vy = JumpStrength
			g.isGrounded = false
			g.coyoteTimer = 0
			g.jumpBufferTimer = 0
		}

		// Variable jump height
		if inpututil.IsKeyJustReleased(ebiten.KeySpace) && g.vy < 0 {
			g.vy *= JumpCutMultiplier
		}

		// Actions
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.isAttacking = true
			g.attackSoundPlayed = false
			g.resetAnim()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) {
			g.isProtecting = true
			g.resetAnim()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) {
			g.isDialogue = true
			g.resetAnim()
		}
	}

	// Acceleration
	accel := GroundAccel
	friction := GroundFriction
	if !g.isGrounded {
		accel = AirAccel
		friction = AirFriction
	}

	if inputX != 0 {
		g.vx += inputX * accel
		// Clamp to max speed
		if g.vx > MaxSpeedX {
			g.vx = MaxSpeedX
		}
		if g.vx < -MaxSpeedX {
			g.vx = -MaxSpeedX
		}
	} else {
		// Friction
		g.vx *= friction
		// Stop at low speed
		if g.vx > -0.1 && g.vx < 0.1 {
			g.vx = 0
		}
	}

	// Horizontal move
	g.x += g.vx
	g.handleCollisionsX(g.vx)

	gravMult := 1.0
	if g.vy > 0 {
		gravMult = FallGravityMult
	}
	g.vy += Gravity * gravMult

	if g.vy > MaxFallSpeed {
		g.vy = MaxFallSpeed
	}

	g.y += g.vy
	g.handleCollisionsY()

	// World bounds
	mapW := float64(g.tilemap.Cols * g.tilemap.TileSize)
	mapH := float64(g.tilemap.Rows * g.tilemap.TileSize)
	if g.x < PlayerHitboxW/2 {
		g.x = PlayerHitboxW / 2
	}
	if g.x > mapW-PlayerHitboxW/2 {
		g.x = mapW - PlayerHitboxW/2
	}
	if g.y > mapH {
		g.y = mapH
		g.vy = 0
		g.isGrounded = true
	}

	g.updateAnimationState(g.vx != 0)
}

// Update anim
func (g *Game) updateAnimationState(isMoving bool) {
	var nextState AnimationState
	if g.isAttacking {
		nextState = StateAttack
	} else if g.isProtecting {
		nextState = StateProtection
	} else if g.isDialogue {
		nextState = StateDialogue
	} else if g.vy < -0.5 {
		nextState = StateJump
	} else if !g.isGrounded && g.vy > 0.5 {
		nextState = StateFall
	} else if isMoving {
		nextState = StateWalk
	} else {
		nextState = StateIdle
	}

	if nextState != g.currentState {
		g.currentState = nextState
		g.resetAnim()
	}

	switch g.currentState {
	case StateAttack:
		g.currentSpriteSheet = g.attackSpriteSheet
		g.totalFrames = 6
		g.frameDelay = 5
	case StateProtection:
		g.currentSpriteSheet = g.protectionSpriteSheet
		g.totalFrames = 4
		g.frameDelay = 8
	case StateDialogue:
		g.currentSpriteSheet = g.dialogueSpriteSheet
		g.totalFrames = 4
		g.frameDelay = 8
	case StateWalk:
		g.currentSpriteSheet = g.walkSpriteSheet
		g.totalFrames = 7
		g.frameDelay = 4
	case StateJump:
		g.currentSpriteSheet = g.jumpSpriteSheet
		g.totalFrames = 6
		g.frameDelay = 8
		if g.currentFrame >= g.totalFrames {
			g.currentFrame = g.totalFrames - 1
		}
	case StateFall:
		g.currentSpriteSheet = g.fallSpriteSheet
		g.totalFrames = 3
		g.frameDelay = 8
		if g.currentFrame >= g.totalFrames {
			g.currentFrame = g.totalFrames - 1
		}
	case StateIdle:
		g.currentSpriteSheet = g.idleSpriteSheet
		g.totalFrames = 4
		g.frameDelay = 10
	}

	g.frameCounter++
	if g.frameCounter >= g.frameDelay {
		g.frameCounter = 0
		g.currentFrame++
		if g.currentFrame >= g.totalFrames {
			if g.currentState == StateWalk || g.currentState == StateIdle {
				g.currentFrame = 0
			} else {
				g.currentFrame = g.totalFrames - 1
			}
		}
	}
}

func (g *Game) resetAnim() {
	g.currentFrame = 0
	g.frameCounter = 0
}

func (g *Game) handleActionStates() {
	if g.isAttacking {
		if g.currentFrame >= g.totalFrames-1 {
			g.isAttacking = false
			g.resetAnim()
		}
	}
	if g.isProtecting {
		if !ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			g.isProtecting = false
			g.resetAnim()
		}
	}
	if g.isDialogue {
		// Controlled by dialogue system
		// Simple animation loop
		if g.dialogueSystem == nil || !g.dialogueSystem.Active {
			g.isDialogue = false
			g.resetAnim()
		}
	}
}

func (g *Game) handleCollisionsX(vx float64) {
	// AABB collision
	// Check corners
	// Hitbox

	left := int((g.x - PlayerHitboxW/2) / float64(g.tilemap.TileSize))
	right := int((g.x + PlayerHitboxW/2) / float64(g.tilemap.TileSize))
	top := int((g.y - PlayerHitboxH) / float64(g.tilemap.TileSize))
	bottom := int((g.y - 1) / float64(g.tilemap.TileSize))

	// Collisions
	if vx < 0 { // Moving Left
		if g.tilemap.IsSolid(g.tilemap.GetTile(left, top)) || g.tilemap.IsSolid(g.tilemap.GetTile(left, bottom)) || g.tilemap.IsSolid(g.tilemap.GetTile(left, (top+bottom)/2)) {
			g.x = float64(left+1)*float64(g.tilemap.TileSize) + PlayerHitboxW/2
			g.vx = 0
		}
	} else if vx > 0 { // Moving Right
		if g.tilemap.IsSolid(g.tilemap.GetTile(right, top)) || g.tilemap.IsSolid(g.tilemap.GetTile(right, bottom)) || g.tilemap.IsSolid(g.tilemap.GetTile(right, (top+bottom)/2)) {
			g.x = float64(right)*float64(g.tilemap.TileSize) - PlayerHitboxW/2
			g.vx = 0
		}
	}
}

func (g *Game) handleCollisionsY() {
	g.isGrounded = false
	tileSize := float64(g.tilemap.TileSize)
	hitboxW := float64(PlayerHitboxW)
	hitboxH := float64(PlayerHitboxH)

	// Left and right edges of the player
	// Use a small epsilon to avoid colliding with neighbors when exactly on the edge
	left := int((g.x - hitboxW/2 + 1) / tileSize)
	right := int((g.x + hitboxW/2 - 1) / tileSize)

	if g.vy < 0 {
		// Moving up - Check ceiling
		topY := g.y - hitboxH
		top := int(topY / tileSize)

		if g.tilemap.IsSolid(g.tilemap.GetTile(left, top)) || g.tilemap.IsSolid(g.tilemap.GetTile(right, top)) {
			// Check if we are really moving into it (topY is inside the tile)
			// Snap to bottom of the ceiling tile
			g.y = float64(top+1)*tileSize + hitboxH
			g.vy = 0
		}
	} else {
		// Moving down - Check floor
		// We use g.y because g.y is the bottom of the player
		bottom := int(g.y / tileSize)

		if g.tilemap.IsSolid(g.tilemap.GetTile(left, bottom)) || g.tilemap.IsSolid(g.tilemap.GetTile(right, bottom)) {
			// Landed
			g.y = float64(bottom) * tileSize
			g.vy = 0
			g.isGrounded = true
		} else {
			// Check if we are close to the ground (snap)
			// This helps with slopes or micro-gaps
			checkBelow := int((g.y + 2) / tileSize) // Check 2 pixels below
			if (g.tilemap.IsSolid(g.tilemap.GetTile(left, checkBelow)) || g.tilemap.IsSolid(g.tilemap.GetTile(right, checkBelow))) &&
				g.y > float64(checkBelow)*tileSize-2 { // Only snap if very close
				g.y = float64(checkBelow) * tileSize
				g.vy = 0
				g.isGrounded = true
			}
		}
	}
}
