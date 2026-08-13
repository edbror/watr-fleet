# Publicar fleet — 10 minutos, una sola vez

## 1. Repos en GitHub (web o gh CLI)
```bash
gh repo create edbror/watr-fleet --public --description "A beautiful terminal dashboard for your fleet of AI coding agents"
gh repo create edbror/homebrew-tap --public --description "WATR Homebrew tap"
```


## 2. Token para el tap
GitHub → Settings → Developer settings → Fine-grained token con permiso `contents: write` SOLO sobre `edbror/homebrew-tap`.
Luego: repo `edbror/watr-fleet` → Settings → Secrets → Actions → nuevo secret `HOMEBREW_TAP_TOKEN` con ese token.

## 3. Push (desde esta carpeta, que ya trae git con historia)
```bash
git remote add origin git@github.com:edbror/watr-fleet.git
git push -u origin main
```

## 4. Release
```bash
git tag v0.10.0
git push origin v0.10.0
```
La GitHub Action corre GoReleaser: binarios darwin/linux (arm64+amd64) en Releases, y la fórmula `fleet` publicada en el tap automáticamente.

## 5. Verifica
```bash
brew install edbror/tap/watr-fleet
fleet --demo
```

## Releases siguientes
Solo: commit → `git tag v0.11.0` → `git push origin main v0.11.0`. Todo lo demás es automático.
