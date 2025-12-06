package main

import (
	"bytes"
	"image"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"

	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func loadImage(path string) *ebiten.Image {
	if IsEmbedded() {
		embeddedFS := GetEmbeddedFS()
		if embeddedFS != nil {
			data, err := fs.ReadFile(embeddedFS, path)
			if err != nil {
				log.Printf("Warning: Failed to load embedded image %s: %v", path, err)
				return nil
			}
			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				log.Printf("Warning: Failed to decode embedded image %s: %v", path, err)
				return nil
			}
			return ebiten.NewImageFromImage(img)
		}
	}
	// Fallback to filesystem
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		log.Printf("Warning: Failed to load image %s: %v", path, err)
		return nil
	}
	return img
}

// Audio loop wrapper
type InfiniteLoop struct {
	src    io.ReadSeeker
	length int64
}

func NewInfiniteLoop(src io.ReadSeeker, length int64) *InfiniteLoop {
	return &InfiniteLoop{
		src:    src,
		length: length,
	}
}

func (l *InfiniteLoop) Read(p []byte) (n int, err error) {
	n, err = l.src.Read(p)
	if err == io.EOF {
		_, err = l.src.Seek(0, io.SeekStart)
		if err != nil {
			return 0, err
		}
		n, err = l.src.Read(p)
	}
	return
}

func (l *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	return l.src.Seek(offset, whence)
}

func (l *InfiniteLoop) Length() int64 {
	return l.length
}

func fetchAudioData(path string) ([]byte, error) {
	if IsEmbedded() {
		embeddedFS := GetEmbeddedFS()
		if embeddedFS != nil {
			data, err := fs.ReadFile(embeddedFS, path)
			if err == nil {
				return data, nil
			}
			log.Printf("Warning: Failed to load embedded audio %s, trying HTTP: %v", path, err)
		}
	}

	resp, err := http.Get(path)
	if err != nil {
		// HTTP failed, try reading from file system (native build)
		return os.ReadFile(path)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// HTTP request failed, try file system
		return os.ReadFile(path)
	}

	return io.ReadAll(resp.Body)
}

func loadAudio(audioContext *audio.Context, path string) *audio.Player {
	data, err := fetchAudioData(path)
	if err != nil {
		log.Printf("Warning: Failed to load audio %s: %v", path, err)
		return nil
	}

	stream, err := mp3.DecodeWithSampleRate(44100, bytes.NewReader(data))
	if err != nil {
		log.Printf("Warning: Failed to decode audio %s: %v", path, err)
		return nil
	}

	player, err := audioContext.NewPlayer(stream)
	if err != nil {
		log.Printf("Warning: Failed to create player for %s: %v", path, err)
		return nil
	}

	return player
}

func loadLoopingAudio(audioContext *audio.Context, path string) *audio.Player {
	data, err := fetchAudioData(path)
	if err != nil {
		log.Printf("Warning: Failed to load audio %s: %v", path, err)
		return nil
	}

	stream, err := mp3.DecodeWithSampleRate(44100, bytes.NewReader(data))
	if err != nil {
		log.Printf("Warning: Failed to decode audio %s: %v", path, err)
		return nil
	}

	// Create an infinite loop
	loop := audio.NewInfiniteLoop(stream, stream.Length())

	player, err := audioContext.NewPlayer(loop)
	if err != nil {
		log.Printf("Warning: Failed to create player for %s: %v", path, err)
		return nil
	}

	return player
}

func NewGame() *Game {
	const assetBase = "assets/images/player/"
	const soundBase = "assets/sounds/"

	audioContext := audio.NewContext(44100)

	g := &Game{
		// Intro screen
		gameState:   StateIntro,
		introScreen: NewIntroScreen(),

		// Player sprites
		idleSpriteSheet:       loadImage(assetBase + "idle.png"),
		walkSpriteSheet:       loadImage(assetBase + "walk.png"),
		attackSpriteSheet:     loadImage(assetBase + "attack.png"),
		protectionSpriteSheet: loadImage(assetBase + "shield.png"),
		dialogueSpriteSheet:   loadImage(assetBase + "talking.png"),
		jumpSpriteSheet:       loadImage(assetBase + "jump.png"),
		fallSpriteSheet:       loadImage(assetBase + "fall.png"),

		frameWidth:  100,
		frameHeight: 64,

		// Start position
		x:         1600,
		y:         0,
		speed:     5.0,
		direction: 1,

		currentState: StateIdle,

		audioContext: audioContext,
		sounds:       make(map[string]*audio.Player),
		audioEnabled: true, // Audio enabled by default

		// UI system
		ui: NewUI(),
	}

	// Load BG image
	bgPath := "assets/images/backgrounds/bg.jpg"
	var err error
	g.bgImage, _, err = ebitenutil.NewImageFromFile(bgPath)
	if err != nil {
		log.Printf("Warning: Could not load background: %v (game will run without background)", err)
	}

	// Load tileset
	atlasImg := loadImage("assets/images/tiles/tiles.png")
	if atlasImg == nil {
		log.Fatal("Failed to load tiles.png - required for game")
	}

	// Create tilemap
	g.tilemap = NewTilemap(atlasImg, 16)

	// Generate level
	g.tilemap.GenerateTerrariaWorld()

	g.currentSpriteSheet = g.idleSpriteSheet
	g.totalFrames = 6
	g.frameDelay = 10

	// Init dialogue
	g.dialogueSystem = NewDialogueSystem()

	// Load sounds async
	log.Println("Loading audio files... (press M to toggle audio)")

	// BG music
	go func() {
		player := loadLoopingAudio(audioContext, soundBase+"Background.mp3")
		if player != nil {
			player.SetVolume(0.3)
			g.sounds["background"] = player
			log.Println("Background music loaded successfully")
		}
	}()

	// Running sound
	go func() {
		player := loadLoopingAudio(audioContext, soundBase+"running.mp3")
		if player != nil {
			player.SetVolume(0.5)
			g.sounds["running"] = player
			log.Println("Running sound loaded successfully")
		}
	}()

	// Attack sound
	go func() {
		player := loadAudio(audioContext, soundBase+"attack.mp3")
		if player != nil {
			g.sounds["attack"] = player
			log.Println("Attack sound loaded successfully")
		}
	}()

	// Chest sound
	go func() {
		player := loadAudio(audioContext, soundBase+"chest.mp3")
		if player != nil {
			g.sounds["chest"] = player
			log.Println("Chest sound loaded successfully")
		}
	}()

	// Slime sound
	go func() {
		player := loadAudio(audioContext, soundBase+"slime_monster_move.mp3")
		if player != nil {
			player.SetVolume(0.4)
			g.sounds["slime"] = player
			log.Println("Slime sound loaded successfully")
		}
	}()

	// Init player stats
	g.PlayerMaxHealth = 100
	g.PlayerHealth = 100

	// Spawn monsters
	mapWidth := g.tilemap.Cols * 16
	for i := 0; i < 5; i++ { // Reduced for profiling
		spawnX := 200 + rand.Intn(mapWidth-400) // Adjusted for smaller map

		r := rand.Float64()
		variant := SlimeGreen
		if r < 0.2 {
			variant = SlimeRed
		} else if r < 0.5 {
			variant = SlimeBlue
		}

		g.monsters = append(g.monsters, NewSlime(float64(spawnX), 0, variant))
	}

	log.Println("Game initialized successfully!")
	return g
}
