package main

import (
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// Portrait cache to avoid reloading images
var portraitCache = make(map[string]*ebiten.Image)

func loadCachedPortrait(path string) *ebiten.Image {
	if path == "" {
		return nil
	}
	if cached, ok := portraitCache[path]; ok {
		return cached
	}
	img := loadImage(path)
	portraitCache[path] = img
	return img
}

type DialogueLine struct {
	Speaker  string
	Text     string
	Portrait string // Portrait path, empty if none
	Emotion  string // e.g. excited, sad, neutral
}

type DialogueSystem struct {
	Active      bool
	Lines       []DialogueLine
	CurrentLine int

	// Typewriter
	CharIndex    int
	CharTimer    int
	CharsPerTick int // Speed: lower = faster

	// Box appearance
	BoxX, BoxY float32
	BoxW, BoxH float32
	Padding    float32

	// Animation
	AnimationTimer int
	SlideInSpeed   float32

	// Portraits
	PortraitImage *ebiten.Image
	PortraitSize  float32

	// Font
	Face font.Face

	// Colors
	BoxColor     color.RGBA
	BorderColor  color.RGBA
	TextColor    color.Color
	SpeakerColor color.Color

	// Input cooldown
	InputCooldown int
}

func NewDialogueSystem() *DialogueSystem {
	return &DialogueSystem{
		CharsPerTick:  2,
		BoxW:          1000,
		BoxH:          150,
		Padding:       20,
		SlideInSpeed:  10,
		PortraitSize:  100,
		Face:          basicfont.Face7x13, // Custom font later
		BoxColor:      color.RGBA{0, 0, 0, 200},
		BorderColor:   color.RGBA{255, 255, 255, 255},
		TextColor:     color.RGBA{255, 255, 255, 255},
		SpeakerColor:  color.RGBA{100, 200, 255, 255},
		InputCooldown: 0,
	}
}

// Start dialogue
func (ds *DialogueSystem) Start(lines []DialogueLine) {
	ds.Active = true
	ds.Lines = lines
	ds.CurrentLine = 0
	ds.CharIndex = 0
	ds.CharTimer = 0
	ds.AnimationTimer = 0
	ds.InputCooldown = 10 // Prevent instant skip

	// Load first portrait (cached)
	if len(lines) > 0 && lines[0].Portrait != "" {
		ds.PortraitImage = loadCachedPortrait(lines[0].Portrait)
		if ds.PortraitImage == nil {
			log.Printf("Warning: Failed to load portrait %s", lines[0].Portrait)
		}
	} else {
		ds.PortraitImage = nil
	}

	// Center box bottom, slide in
	ds.BoxX = float32(ScreenWidth-int(ds.BoxW)) / 2
	ds.BoxY = float32(ScreenHeight) + ds.BoxH // Off-screen initially
}

func (ds *DialogueSystem) Update() {
	if !ds.Active || len(ds.Lines) == 0 {
		return
	}

	if ds.InputCooldown > 0 {
		ds.InputCooldown--
	}

	currentText := ds.Lines[ds.CurrentLine].Text
	textLen := len(currentText)

	// Slide-in animation
	targetY := float32(ScreenHeight) - ds.BoxH - 30
	if ds.BoxY > targetY {
		ds.AnimationTimer++
		ds.BoxY -= ds.SlideInSpeed
		if ds.BoxY < targetY {
			ds.BoxY = targetY
		}
	}

	// Typewriter effect
	if ds.CharIndex < textLen {
		ds.CharTimer++
		if ds.CharTimer >= ds.CharsPerTick {
			ds.CharTimer = 0
			ds.CharIndex++
			if ds.CharIndex > textLen {
				ds.CharIndex = textLen
			}
			// Skip spaces
			for ds.CharIndex < textLen && currentText[ds.CharIndex] == ' ' {
				ds.CharIndex++
			}
		}
	}

	// Advance on input
	if ds.InputCooldown <= 0 && ds.BoxY <= targetY {
		if ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyEnter) ||
			ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {

			if ds.CharIndex < len(currentText) {
				// Skip to full text
				ds.CharIndex = len(currentText)
			} else {
				// Next line
				ds.CurrentLine++
				ds.CharIndex = 0
				ds.CharTimer = 0
				ds.InputCooldown = 10

				// Load next portrait (cached)
				if ds.CurrentLine < len(ds.Lines) && ds.Lines[ds.CurrentLine].Portrait != "" {
					ds.PortraitImage = loadCachedPortrait(ds.Lines[ds.CurrentLine].Portrait)
					if ds.PortraitImage == nil {
						log.Printf("Warning: Failed to load portrait %s", ds.Lines[ds.CurrentLine].Portrait)
					}
				} else {
					ds.PortraitImage = nil
				}

				if ds.CurrentLine >= len(ds.Lines) {
					ds.Active = false
				}
			}
		}
	}
}

func (ds *DialogueSystem) Draw(screen *ebiten.Image) {
	if !ds.Active || len(ds.Lines) == 0 {
		return
	}

	// Shadow
	vector.DrawFilledRect(screen, ds.BoxX+5, ds.BoxY+5, ds.BoxW, ds.BoxH,
		color.RGBA{0, 0, 0, 100}, false)

	// Main box
	vector.DrawFilledRect(screen, ds.BoxX, ds.BoxY, ds.BoxW, ds.BoxH, ds.BoxColor, false)

	// Border
	vector.StrokeRect(screen, ds.BoxX, ds.BoxY, ds.BoxW, ds.BoxH, 2, ds.BorderColor, false)

	line := ds.Lines[ds.CurrentLine]

	// Portrait
	textStartX := int(ds.BoxX + ds.Padding)
	if ds.PortraitImage != nil {
		portraitX := ds.BoxX + ds.Padding
		portraitY := ds.BoxY + ds.Padding
		op := &ebiten.DrawImageOptions{}
		scale := float64(ds.PortraitSize) / float64(ds.PortraitImage.Bounds().Dx())
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(portraitX), float64(portraitY))
		screen.DrawImage(ds.PortraitImage, op)
		textStartX = int(ds.BoxX + ds.Padding + ds.PortraitSize + 10)
	}

	// Speaker name (if any)
	yOffset := int(ds.BoxY + ds.Padding + 13)
	if line.Speaker != "" {
		text.Draw(screen, line.Speaker+":", ds.Face, textStartX, yOffset, ds.SpeakerColor)
		yOffset += 20
	}

	// Typewriter text
	charIndex := ds.CharIndex
	if charIndex > len(line.Text) {
		charIndex = len(line.Text)
	}
	displayText := line.Text[:charIndex]
	textWidth := int(ds.BoxW - ds.Padding*2)
	if ds.PortraitImage != nil {
		textWidth -= int(ds.PortraitSize + 10)
	}
	wrapped := wrapText(displayText, textWidth, ds.Face)

	// Dynamic height
	lineHeight := 16
	minLines := 3
	actualLines := len(wrapped)
	if actualLines < minLines {
		actualLines = minLines
	}
	boxHeight := ds.Padding*2 + float32(actualLines*lineHeight+20) // Extra for speaker
	if boxHeight > ds.BoxH {
		ds.BoxH = boxHeight
	}

	// Emotion color
	textColor := ds.TextColor
	switch line.Emotion {
	case "excited":
		textColor = color.RGBA{255, 255, 0, 255} // Yellow
	case "sad":
		textColor = color.RGBA{0, 255, 255, 255} // Cyan
	case "angry":
		textColor = color.RGBA{255, 0, 0, 255} // Red
	default:
		textColor = ds.TextColor
	}

	for _, wline := range wrapped {
		text.Draw(screen, wline, ds.Face, textStartX, yOffset, textColor)
		yOffset += lineHeight
	}

	// Continue indicator
	if ds.CharIndex >= len(line.Text) {
		indicator := "▼"
		if (ds.CharTimer/10)%2 == 0 { // Blink
			text.Draw(screen, indicator, ds.Face,
				int(ds.BoxX+ds.BoxW-ds.Padding-10),
				int(ds.BoxY+ds.BoxH-ds.Padding), ds.TextColor)
		}
	}
}

// Word wrap
func wrapText(s string, maxWidth int, face font.Face) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var current string

	for _, word := range words {
		test := current
		if test != "" {
			test += " "
		}
		test += word

		bounds := text.BoundString(face, test)
		if bounds.Dx() > maxWidth && current != "" {
			lines = append(lines, current)
			current = word
		} else {
			current = test
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
