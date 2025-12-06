# Violet

A minimal Terraria-style 2D platformer written in Go + Ebitengine.

## Controls
- **WASD / Arrows**: Move & Jump
- **Space**: Jump
- **Enter**: Attack
- **Shift**: Shield
- **E**: Interact / Talk
- **M**: Toggle Audio
- **F1 / F2**: Debug / Tile Palette

## Run Native (Linux/Mac/PC)
```bash
go run .
```

## Run Web (WASM)
Build and start a local server:
```bash
./build_wasm.sh
cd dist
python3 -m http.server 8080
```
Open [http://localhost:8080](http://localhost:8080).

>> **Note:** Cleared browser cache (Ctrl+Shift+R) if updates don't appear.
>> Also not fully completed
