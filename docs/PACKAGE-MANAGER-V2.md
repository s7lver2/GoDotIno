# Tsuki v3 — Package Manager & Build System Specification

> **Estado:** Especificación técnica de implementación  
> **Versión objetivo:** tsuki 3.0  
> **Aplica a:** `tsuki-core`, `tsuki-flash`, y proyectos de usuario

---

## Tabla de Contenidos

1. [Visión General](#1-visión-general)
2. [Tipos de Proyecto](#2-tipos-de-proyecto)
3. [Archivo de Configuración `tsuki-config.toml`](#3-archivo-de-configuración-tsuki-configtoml)
4. [Archivo de Metadatos `tsuki-package.json`](#4-archivo-de-metadatos-tsuki-packagejson)
5. [Sistema de Dependencias — Local vs Global](#5-sistema-de-dependencias--local-vs-global)
6. [Registro de Paquetes](#6-registro-de-paquetes)
7. [Comandos de la CLI](#7-comandos-de-la-cli)
8. [Build & Release — Sin Docker](#8-build--release--sin-docker)
9. [Publicación Automática a GitHub Releases](#9-publicación-automática-a-github-releases)
10. [Estructura de Directorios de Instalación](#10-estructura-de-directorios-de-instalación)
11. [Cómo Empaquetar tsuki-core y tsuki-flash](#11-cómo-empaquetar-tsuki-core-y-tsuki-flash)
12. [Flujo Completo de Ejemplo](#12-flujo-completo-de-ejemplo)
13. [Guía de Implementación — Cambios en el Código](#13-guía-de-implementación--cambios-en-el-código)

---

## 1. Visión General

El nuevo sistema de gestión de paquetes de tsuki sustituye completamente al anterior basado en `goduino.json`. Se divide en tres capas:

```
tsuki-config.toml        ← configuración del proyecto (reemplaza goduino.json)
tsuki-package.json       ← metadatos adicionales para publicar en el registro
keys.json                ← lista de fuentes de paquetes (editable por el usuario)
packages.json            ← índice de paquetes de cada fuente
```

La CLI (`tsuki`) orquesta todo: inicializar proyectos, instalar dependencias, compilar, ejecutar y publicar releases.

---

## 2. Tipos de Proyecto

Al crear un proyecto con `tsuki init`, se elige entre dos tipos:

### `library` — Librería

- Produce un artefacto reutilizable (`.a` estático, `.so` dinámico, o conjunto de headers).
- No tiene entrypoint de ejecución.
- Se publica en el registro para que otros proyectos la instalen.
- Usa la sección `[[lib]]` en el config.

### `program` — Programa / Aplicación

- Produce uno o más binarios ejecutables.
- Tiene entrypoint(s) configurados (`entrypoint = "make all"` o un comando personalizado).
- Soporta `tsuki run` para ejecutar el binario principal.
- Soporta `tsuki push` para publicar releases en GitHub.
- Usa la sección `[[package]]` en el config.

> Los proyectos `tsuki-core` y `tsuki-flash` son de tipo **program**.

---

## 3. Archivo de Configuración `tsuki-config.toml`

Este archivo vive en la raíz del proyecto y reemplaza a `goduino.json` / `godotinolib.toml`.

### Esquema Completo

```toml
# ─── Metadatos del proyecto ───────────────────────────────────
[package]
name        = "mi-proyecto"
version     = "1.0.0"
edition     = "2024"          # año de convención — reservado para compatibilidad futura
description = "Descripción breve"
license     = "MIT"
authors     = ["s7lver", "tsuki Contributors"]
readme      = "README.md"
repository  = "https://github.com/s7lver/mi-proyecto"
keywords    = ["arduino", "embedded"]
type        = "program"       # "program" | "library"

# ─── Binarios (solo para type = "program") ────────────────────
# Puede haber varios [[package]] si el proyecto compila múltiples binarios
[[package]]
name       = "mi-proyecto"
path       = "src/main.rs"        # archivo de entrada (puede ser main.rs, main.go, main.cpp…)
entrypoint = "cargo run"          # comando a ejecutar con `tsuki run`

# Segundo binario opcional
[[package]]
name       = "mi-proyecto-daemon"
path       = "src/daemon.rs"
entrypoint = "cargo run --bin mi-proyecto-daemon"

# ─── Librerías (solo para type = "library") ───────────────────
[[lib]]
name = "mi-libreria"
path = "src/lib.rs"

# ─── Dependencias locales al proyecto ─────────────────────────
# Por defecto todas las dependencias son locales (en .tsuki/deps/)
[dependencies]
ws2812   = "1.0"
dht      = "^1.2"
u8g2     = { version = "2.0", features = ["ssd1306"] }

# ─── Dependencias de desarrollo (no se incluyen en release) ───
[dev-dependencies]
unity = "2.5"

# ─── Perfil de compilación release ───────────────────────────
[profile.release]
opt-level     = 3
lto           = true
codegen-units = 1
strip         = true

# ─── Configuración de publicación (solo program) ─────────────
[publish]
github_repo  = "s7lver/mi-proyecto"   # owner/repo
targets      = ["x86_64-linux", "x86_64-windows", "aarch64-linux", "x86_64-macos"]
pre_build    = "make clean"            # comando opcional antes de compilar
post_build   = "make test"             # comando opcional después de compilar
```

### Reglas de Parsing

- `[[package]]` y `[[lib]]` son arrays de tablas (igual que en Cargo).
- Si `type = "library"` no debe haber secciones `[[package]]`.
- Si `type = "program"` no debe haber secciones `[[lib]]`.
- El primer `[[package]]` es el binario principal (usado por `tsuki run` sin argumentos).

### Semver de Dependencias

| Especificador | Significado |
|---|---|
| `"1.0"` | `>=1.0.0, <2.0.0` |
| `"^1.2.3"` | `>=1.2.3, <2.0.0` |
| `"~1.2.3"` | `>=1.2.3, <1.3.0` |
| `"=1.2.3"` | exactamente 1.2.3 |
| `">=1.0, <2.0"` | rango explícito |

---

## 4. Archivo de Metadatos `tsuki-package.json`

Este archivo es **adicional** al `tsuki-config.toml` y contiene información específica para el registro de paquetes. Se genera automáticamente con `tsuki push` pero también puede mantenerse manualmente.

```json
{
  "name": "ws2812",
  "version": "1.0.0",
  "description": "WS2812 / NeoPixel LED driver",
  "author": "tsuki-team",
  "license": "MIT",
  "repository": "https://github.com/s7lver/tsuki",
  "keywords": ["led", "neopixel", "ws2812"],
  "type": "library",

  "cpp_header": "Adafruit_NeoPixel.h",
  "arduino_lib": "Adafruit NeoPixel",

  "files": [
    "tsuki-config.toml",
    "src/lib.rs",
    "README.md"
  ],

  "checksums": {
    "tsuki-config.toml": "sha256:abc123...",
    "src/lib.rs": "sha256:def456..."
  },

  "published_at": "2025-01-01T00:00:00Z",
  "download_url": "https://github.com/s7lver/tsuki/releases/download/ws2812-v1.0.0/ws2812-1.0.0.tar.gz"
}
```

### Diferencias entre `tsuki-config.toml` y `tsuki-package.json`

| Campo | `tsuki-config.toml` | `tsuki-package.json` |
|---|---|---|
| Propósito | Configurar el build local | Metadatos para el registro |
| Dependencias | Sí (para resolver localmente) | No |
| Perfil release | Sí | No |
| Entrypoint | Sí | No |
| Checksums | No | Sí |
| URL de descarga | No | Sí |
| Headers C++ | No | Sí (para librerías) |

---

## 5. Sistema de Dependencias — Local vs Global

### Comportamiento por Defecto: Local

Cada proyecto tiene su propio directorio de dependencias:

```
mi-proyecto/
├── tsuki-config.toml
├── tsuki-package.json
├── src/
│   └── main.rs
└── .tsuki/
    ├── deps/                    ← dependencias locales del proyecto
    │   ├── ws2812/
    │   │   └── 1.0.0/
    │   │       └── tsuki-config.toml
    │   └── dht/
    │       └── 1.2.3/
    │           └── tsuki-config.toml
    ├── lock.json                ← versiones concretas resueltas (como Cargo.lock)
    └── cache/                   ← archivos descargados sin desempaquetar
```

Esto garantiza que dos proyectos pueden usar versiones diferentes de la misma dependencia sin conflictos.

### Modo Global: `--global`

```bash
tsuki install ws2812 --global
```

Instala en el directorio global del sistema:

| SO | Ruta global |
|---|---|
| Linux | `~/.local/share/tsuki/global/deps/` |
| macOS | `~/Library/Application Support/tsuki/global/deps/` |
| Windows | `%APPDATA%\tsuki\global\deps\` |

Los paquetes globales son visibles para todos los proyectos que no tengan una versión local del mismo paquete. La resolución sigue este orden de prioridad:

```
.tsuki/deps/<pkg>/  →  (si no existe)  →  global deps/<pkg>/
```

### Archivo `lock.json`

Se genera automáticamente en `.tsuki/lock.json` y registra las versiones exactas resueltas:

```json
{
  "generated": "2025-01-01T00:00:00Z",
  "dependencies": [
    {
      "name": "ws2812",
      "requested": "1.0",
      "resolved": "1.0.2",
      "source": "tsuki-team@https://raw.githubusercontent.com/.../packages.json",
      "checksum": "sha256:abc123...",
      "scope": "local"
    }
  ]
}
```

---

## 6. Registro de Paquetes

### `keys.json` — Fuentes de Paquetes

Ubicado en el directorio de configuración de tsuki:

| SO | Ruta |
|---|---|
| Linux | `~/.config/tsuki/keys.json` |
| macOS | `~/.config/tsuki/keys.json` |
| Windows | `%APPDATA%\tsuki\keys.json` |

```json
{
  "_comment": "Lista de fuentes de paquetes. Editable por el usuario.",
  "registries": [
    {
      "id": "tsuki-team",
      "name": "Tsuki Official Registry",
      "packages_url": "https://raw.githubusercontent.com/s7lver/tsuki/main/pkg/packages.json",
      "trusted": true
    },
    {
      "id": "mi-equipo",
      "name": "Registro privado de mi empresa",
      "packages_url": "https://raw.githubusercontent.com/mi-empresa/tsuki-pkgs/main/packages.json",
      "trusted": false
    }
  ]
}
```

### `packages.json` — Índice de Paquetes

Cada entrada en `registries` apunta a un `packages.json` con esta estructura:

```json
{
  "_comment": "Índice de paquetes del registro tsuki-team",
  "packages": {
    "ws2812": {
      "description": "WS2812 / NeoPixel LED driver (Adafruit NeoPixel)",
      "author": "tsuki-team",
      "latest": "1.0.0",
      "versions": {
        "1.0.0": {
          "download_url": "https://github.com/s7lver/tsuki/releases/download/ws2812-v1.0.0/ws2812-1.0.0.tar.gz",
          "metadata_url": "https://raw.githubusercontent.com/s7lver/tsuki/main/pkg/ws2812/v1.0.0/tsuki-package.json",
          "checksum": "sha256:abc123...",
          "published_at": "2025-01-01T00:00:00Z"
        }
      }
    },
    "dht": {
      "description": "DHT11 / DHT22 temperature & humidity sensor",
      "author": "tsuki-team",
      "latest": "1.2.3",
      "versions": {
        "1.0.0": { "...": "..." },
        "1.2.3": { "...": "..." }
      }
    }
  }
}
```

### Sintaxis de Instalación con Fuente Explícita

```bash
# Formato: <registry-id>@<paquete>:<version>
tsuki install tsuki-team@ws2812:1.0.0
tsuki install tsuki-team@ws2812          # usa latest
tsuki install ws2812                     # busca en todos los registries por orden
tsuki install ws2812:1.0.0              # versión específica en cualquier registry
```

### Comando `tsuki updatedb`

Descarga todos los `packages.json` de las fuentes en `keys.json` y los cachea localmente en `~/.cache/tsuki/db/`. Esto permite resolver paquetes sin conexión.

```
~/.cache/tsuki/db/
├── tsuki-team.json        ← copia del packages.json de tsuki-team
└── mi-equipo.json         ← copia del packages.json de mi-equipo
```

---

## 7. Comandos de la CLI

### Gestión de Paquetes

#### `tsuki install`

Instala todas las dependencias declaradas en `tsuki-config.toml`:

```bash
tsuki install                     # instala deps localmente en .tsuki/deps/
tsuki install --global            # instala deps globalmente
tsuki install --frozen            # usa lock.json sin resolver versiones nuevas
```

#### `tsuki install <paquete>`

Instala un paquete y lo añade a `[dependencies]` en `tsuki-config.toml`:

```bash
tsuki install ws2812                        # versión latest del primer registry que lo tenga
tsuki install ws2812:1.0.0                  # versión específica
tsuki install tsuki-team@ws2812:1.0.0       # registry + paquete + versión
tsuki install ws2812 --global               # instalar globalmente
tsuki install ws2812 --dev                  # añadir a [dev-dependencies]
```

**Flujo interno:**

1. Consulta `~/.cache/tsuki/db/` (o descarga si no existe).
2. Resuelve la versión según semver.
3. Descarga el tarball desde `download_url`.
4. Verifica checksum SHA-256.
5. Desempaqueta en `.tsuki/deps/<pkg>/<version>/`.
6. Actualiza `tsuki-config.toml` y `.tsuki/lock.json`.

#### `tsuki updatedb`

```bash
tsuki updatedb          # actualiza todos los registries de keys.json
tsuki updatedb tsuki-team  # actualiza solo un registry
```

---

### Comandos de Build y Proyecto

#### `tsuki init`

Crea un proyecto nuevo interactivo:

```bash
tsuki init                          # wizard interactivo
tsuki init mi-proyecto              # nombre predefinido
tsuki init mi-proyecto --type program
tsuki init mi-libreria --type library
```

Genera la siguiente estructura:

```
mi-proyecto/
├── tsuki-config.toml
├── tsuki-package.json
├── src/
│   └── main.rs           # (program) o lib.rs (library)
├── .tsuki/
│   └── .gitignore        # ignora deps/ y cache/
└── README.md
```

#### `tsuki run`

Ejecuta el entrypoint del primer `[[package]]` (solo para proyectos tipo `program`):

```bash
tsuki run                            # ejecuta el primer [[package]]
tsuki run --bin mi-proyecto-daemon   # ejecuta un [[package]] específico por nombre
tsuki run -- --arg1 --arg2           # pasa argumentos al binario
```

Internamente ejecuta el comando especificado en `entrypoint` del `[[package]]` correspondiente.

#### `tsuki pull`

Equivalente a instalar todas las dependencias del `lock.json` (si existe) o del `tsuki-config.toml`. Similar a `git pull` en concepto: sincroniza el entorno de dependencias.

```bash
tsuki pull              # instala/actualiza deps según lock.json
tsuki pull --update     # actualiza deps a las versiones semver compatibles más recientes
```

#### `tsuki push`

Compila el proyecto para todos los targets definidos en `[publish.targets]` y sube los binarios a GitHub Releases (ver Sección 9).

```bash
tsuki push                         # publica versión actual (lee version del [package])
tsuki push --tag v2.1.0            # fuerza una tag específica
tsuki push --draft                 # crea release como draft
tsuki push --prerelease            # marca como pre-release
tsuki push --target x86_64-linux   # solo un target
```

---

## 8. Build & Release — Sin Docker

En lugar de compilación cruzada con Docker, el sistema usa las siguientes estrategias según el target:

### Estrategia 1: Compilación Nativa en CI (Recomendada)

Usar **GitHub Actions** con runners nativos para cada plataforma. Cada runner compila para su OS propio, sin emulación ni cross-compilation:

```yaml
# .github/workflows/release.yml
jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            target: x86_64-unknown-linux-gnu
            artifact: tsuki-core-linux-x86_64
          - os: ubuntu-latest
            target: aarch64-unknown-linux-gnu
            artifact: tsuki-core-linux-aarch64
            use_cross: true          # solo aarch64 necesita cross
          - os: windows-latest
            target: x86_64-pc-windows-msvc
            artifact: tsuki-core-windows-x86_64.exe
          - os: macos-latest
            target: x86_64-apple-darwin
            artifact: tsuki-core-macos-x86_64
          - os: macos-latest
            target: aarch64-apple-darwin
            artifact: tsuki-core-macos-aarch64
```

### Estrategia 2: `cross` para Linux ARM (Solo cuando sea necesario)

Para `aarch64-linux` desde un runner x86_64, usar [`cross`](https://github.com/cross-rs/cross) que es mucho más ligero que Docker personalizado:

```bash
cargo install cross
cross build --target aarch64-unknown-linux-gnu --release
```

`cross` descarga imágenes Docker preconfiguradas solo para el caso específico de cross-compilation de Rust. No requiere Dockerfile propio.

### Estrategia 3: `cargo-zigbuild` (Alternativa sin Docker)

Para cross-compilation sin Docker en absoluto:

```bash
cargo install cargo-zigbuild
pip install ziglang
cargo zigbuild --target aarch64-unknown-linux-gnu --release
```

`cargo-zigbuild` usa el compilador Zig como linker alternativo, que soporta múltiples targets sin necesitar contenedores.

### Tabla de Estrategias por Target

| Target | Estrategia | Herramienta |
|---|---|---|
| `x86_64-linux` | Runner nativo Ubuntu | `cargo build` |
| `aarch64-linux` | Cross en Ubuntu x86_64 | `cross` o `cargo-zigbuild` |
| `x86_64-windows` | Runner nativo Windows | `cargo build` |
| `x86_64-macos` | Runner nativo macOS | `cargo build` |
| `aarch64-macos` (Apple Silicon) | Runner nativo macOS latest | `cargo build` |

### Formato de Artefactos

Cada artefacto es un archivo comprimido que contiene el binario + README + LICENSE:

```
tsuki-core-v3.0.0-x86_64-linux.tar.gz
  ├── tsuki-core               ← binario
  ├── tsuki-flash              ← binario
  ├── README.md
  └── LICENSE

tsuki-core-v3.0.0-x86_64-windows.zip
  ├── tsuki-core.exe
  ├── tsuki-flash.exe
  ├── README.md
  └── LICENSE
```

---

## 9. Publicación Automática a GitHub Releases

### Flujo de `tsuki push`

```
tsuki push
    │
    ├── 1. Lee tsuki-config.toml → obtiene name, version, targets
    ├── 2. Ejecuta pre_build (si está definido)
    ├── 3. Para cada target en [publish.targets]:
    │       ├── Compila con el método apropiado (nativo / cross / zigbuild)
    │       ├── Empaqueta binario + assets en .tar.gz o .zip
    │       └── Calcula SHA-256 del artefacto
    ├── 4. Genera checksums.txt con todos los hashes
    ├── 5. Ejecuta post_build (si está definido)
    ├── 6. Crea tag git: v{version}
    ├── 7. Crea GitHub Release via API:
    │       ├── Título: "{name} v{version}"
    │       ├── Body: CHANGELOG.md o generado automáticamente desde commits
    │       └── Adjunta todos los artefactos + checksums.txt
    └── 8. Actualiza packages.json en el registry (si está configurado)
```

### Configuración de Credenciales

`tsuki push` requiere un token de GitHub. Se configura de estas formas (por prioridad):

```bash
# Variable de entorno (recomendado para CI)
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Archivo de configuración del usuario
~/.config/tsuki/auth.toml
```

```toml
# ~/.config/tsuki/auth.toml
[github]
token = "ghp_xxxxxxxxxxxx"
```

### Workflow de GitHub Actions Completo

Este workflow se activa automáticamente al hacer `git push --tags` o al ejecutar `tsuki push`:

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to release'
        required: true

jobs:
  build:
    name: Build ${{ matrix.artifact }}
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            target: x86_64-unknown-linux-gnu
            artifact: tsuki-linux-x86_64
            ext: tar.gz

          - os: ubuntu-latest
            target: aarch64-unknown-linux-gnu
            artifact: tsuki-linux-aarch64
            ext: tar.gz
            use_cross: true

          - os: windows-latest
            target: x86_64-pc-windows-msvc
            artifact: tsuki-windows-x86_64
            ext: zip

          - os: macos-latest
            target: x86_64-apple-darwin
            artifact: tsuki-macos-x86_64
            ext: tar.gz

          - os: macos-latest
            target: aarch64-apple-darwin
            artifact: tsuki-macos-aarch64
            ext: tar.gz

    steps:
      - uses: actions/checkout@v4

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable
        with:
          targets: ${{ matrix.target }}

      - name: Install cross (if needed)
        if: matrix.use_cross
        run: cargo install cross

      - name: Build
        run: |
          if [ "${{ matrix.use_cross }}" = "true" ]; then
            cross build --release --target ${{ matrix.target }}
          else
            cargo build --release --target ${{ matrix.target }}
          fi

      - name: Package (Linux/macOS)
        if: matrix.os != 'windows-latest'
        run: |
          mkdir -p dist
          cp target/${{ matrix.target }}/release/tsuki-core dist/
          cp target/${{ matrix.target }}/release/tsuki-flash dist/
          cp README.md LICENSE dist/
          cd dist && tar czf ../${{ matrix.artifact }}.tar.gz .

      - name: Package (Windows)
        if: matrix.os == 'windows-latest'
        shell: pwsh
        run: |
          New-Item -ItemType Directory -Force dist
          Copy-Item "target/${{ matrix.target }}/release/tsuki-core.exe" dist/
          Copy-Item "target/${{ matrix.target }}/release/tsuki-flash.exe" dist/
          Copy-Item README.md, LICENSE dist/
          Compress-Archive -Path dist/* -DestinationPath "${{ matrix.artifact }}.zip"

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.artifact }}
          path: ${{ matrix.artifact }}.${{ matrix.ext }}

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts/

      - name: Generate checksums
        run: |
          cd artifacts
          find . -name "*.tar.gz" -o -name "*.zip" | sort | xargs sha256sum > checksums.txt

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            artifacts/**/*.tar.gz
            artifacts/**/*.zip
            artifacts/checksums.txt
          generate_release_notes: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 10. Estructura de Directorios de Instalación

### Instalación del Sistema (tsuki CLI + core + flash)

```
/usr/local/bin/
├── tsuki           ← CLI principal (Go)
├── tsuki-core      ← motor de transpilación (Rust)
└── tsuki-flash     ← motor de flashing (Rust)
```

### Datos de Usuario

```
~/.config/tsuki/
├── keys.json          ← fuentes de paquetes
└── auth.toml          ← credenciales (GitHub token, etc.)

~/.cache/tsuki/
├── db/
│   ├── tsuki-team.json    ← copia local del packages.json de cada registry
│   └── mi-equipo.json
└── downloads/             ← tarballs descargados sin desempaquetar

~/.local/share/tsuki/
└── global/
    └── deps/
        ├── ws2812/
        │   └── 1.0.0/
        │       ├── tsuki-config.toml
        │       └── tsuki-package.json
        └── dht/
            └── 1.2.3/
                ├── tsuki-config.toml
                └── tsuki-package.json
```

### Estructura de un Proyecto

```
mi-proyecto/
├── tsuki-config.toml      ← configuración principal
├── tsuki-package.json     ← metadatos para el registro (si se publica)
├── src/
│   └── main.rs
├── README.md
├── LICENSE
└── .tsuki/
    ├── deps/              ← dependencias locales
    │   └── <pkg>/<ver>/
    ├── lock.json          ← versiones resueltas
    ├── cache/             ← tarballs cacheados localmente
    └── .gitignore         ← ignora deps/ y cache/
```

El `.tsuki/.gitignore` debe contener:

```
deps/
cache/
```

---

## 11. Cómo Empaquetar tsuki-core y tsuki-flash

Estos dos binarios son los únicos que requieren empaquetarse para distribución. El proceso a partir de v3 es:

### 1. Configurar `tsuki-config.toml` en la raíz del repo

```toml
[package]
name        = "tsuki"
version     = "3.0.0"
edition     = "2024"
description = "Arduino Framework — Write in Go, Upload in C++"
license     = "MIT"
authors     = ["tsuki Contributors", "s7lver"]
readme      = "README.md"
repository  = "https://github.com/s7lver/tsuki"
type        = "program"

[[package]]
name       = "tsuki-core"
path       = "src/main.rs"
entrypoint = "cargo run --bin tsuki-core"

[[package]]
name       = "tsuki-flash"
path       = "flash/main.rs"
entrypoint = "cargo run --bin tsuki-flash"

[lib]
name = "tsuki_core"
path = "src/lib.rs"

[dependencies]
thiserror  = "1.0"
anyhow     = "1.0"
serde      = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
toml       = "0.8"
clap       = { version = "4.5", features = ["derive", "color"] }
rayon      = "1.10"
sha2       = "0.10"
hex        = "0.4"
colored    = "2.1"
walkdir    = "2.5"
ureq       = { version = "2.9", features = ["json"] }
zip        = { version = "0.6", default-features = false, features = ["deflate"] }

[dev-dependencies]
pretty_assertions = "1.4"

[profile.release]
opt-level     = 3
lto           = true
codegen-units = 1
strip         = true

[publish]
github_repo = "s7lver/tsuki"
targets = [
  "x86_64-linux",
  "aarch64-linux",
  "x86_64-windows",
  "x86_64-macos",
  "aarch64-macos"
]
pre_build  = "cargo test"
```

### 2. Para publicar una nueva versión

```bash
# Desde la raíz del repo de tsuki
tsuki push                      # compila para todos los targets y sube a GitHub Releases
tsuki push --draft              # crear borrador primero para revisar
tsuki push --tag v3.1.0         # override de la versión
```

### 3. Actualizar el `packages.json` del registry

Después de publicar, actualizar `pkg/packages.json` para incluir la nueva versión de las librerías tsuki si aplica.

---

## 12. Flujo Completo de Ejemplo

### Crear y publicar una librería

```bash
# 1. Crear el proyecto
tsuki init mi-sensor --type library

# 2. Desarrollar...
# (editar src/lib.rs y tsuki-config.toml)

# 3. Instalar dependencias del proyecto
tsuki pull

# 4. Añadir una dependencia
tsuki install ws2812:1.0.0

# 5. Añadir una dependencia global
tsuki install some-tool --global

# 6. Actualizar el índice de paquetes
tsuki updatedb

# 7. Publicar
tsuki push --tag v1.0.0
```

### Crear y ejecutar un programa

```bash
# 1. Crear el proyecto
tsuki init mi-app --type program

# 2. Instalar dependencias
tsuki install dht
tsuki install u8g2:2.0

# 3. Ejecutar localmente
tsuki run

# 4. Ejecutar un binario específico si hay varios [[package]]
tsuki run --bin mi-app-daemon

# 5. Publicar release
tsuki push
```

---

## 13. Guía de Implementación — Cambios en el Código

### Cambios Necesarios en `tsuki-core` (Rust)

Los cambios principales afectan a `src/runtime/pkg_manager.rs` y `src/runtime/pkg_loader.rs`:

**`src/runtime/pkg_manager.rs`** — Debe ser capaz de:
- Parsear `tsuki-config.toml` (reemplaza parsing de `goduino.json`)
- Resolver dependencias con semver desde `packages.json`
- Manejar el scope local vs global (`--global`)
- Generar y leer `.tsuki/lock.json`
- Implementar `updatedb`: descargar y cachear todos los registries de `keys.json`

**`src/transpiler/config.rs`** — Actualizar para leer el nuevo formato TOML en lugar del JSON anterior.

**Nuevo: `src/publish/`** — Módulo nuevo para `tsuki push`:
- Compilar para múltiples targets
- Empaquetar artefactos
- Interactuar con la GitHub Releases API
- Actualizar `packages.json` del registry

### Cambios en la CLI Go (`cli/`)

**`cli/internal/config/config.go`** — Actualizar estructuras para el nuevo `tsuki-config.toml`:

```go
type TsukiConfig struct {
    Package      PackageMeta     `toml:"package"`
    Packages     []BinaryTarget  `toml:"package,omitempty"` // [[package]]
    Lib          *LibTarget      `toml:"lib,omitempty"`
    Dependencies map[string]any  `toml:"dependencies"`
    DevDeps      map[string]any  `toml:"dev-dependencies"`
    Profile      map[string]Profile `toml:"profile"`
    Publish      *PublishConfig  `toml:"publish,omitempty"`
}

type PackageMeta struct {
    Name        string   `toml:"name"`
    Version     string   `toml:"version"`
    Edition     string   `toml:"edition"`
    Description string   `toml:"description"`
    License     string   `toml:"license"`
    Authors     []string `toml:"authors"`
    Readme      string   `toml:"readme"`
    Repository  string   `toml:"repository"`
    Type        string   `toml:"type"` // "program" | "library"
}

type BinaryTarget struct {
    Name       string `toml:"name"`
    Path       string `toml:"path"`
    Entrypoint string `toml:"entrypoint"`
}

type PublishConfig struct {
    GithubRepo string   `toml:"github_repo"`
    Targets    []string `toml:"targets"`
    PreBuild   string   `toml:"pre_build"`
    PostBuild  string   `toml:"post_build"`
}
```

### Nuevos Comandos en la CLI

Añadir a la estructura de comandos Cobra (`cmd/`):

```
cmd/
├── root.go
├── init.go        ← MODIFICADO: añadir --type flag
├── run.go         ← NUEVO: tsuki run
├── pull.go        ← NUEVO: tsuki pull
├── push.go        ← NUEVO: tsuki push
├── install.go     ← MODIFICADO: soporte --global, --dev, <registry>@<pkg>:<ver>
└── updatedb.go    ← NUEVO: tsuki updatedb
```

### Migración desde `goduino.json`

Para proyectos existentes, proveer un comando de migración:

```bash
tsuki migrate     # convierte goduino.json → tsuki-config.toml automáticamente
```

---

## Apéndice A: Targets Disponibles en `[publish.targets]`

| Identificador tsuki | Target Rust |
|---|---|
| `x86_64-linux` | `x86_64-unknown-linux-gnu` |
| `aarch64-linux` | `aarch64-unknown-linux-gnu` |
| `x86_64-windows` | `x86_64-pc-windows-msvc` |
| `aarch64-windows` | `aarch64-pc-windows-msvc` |
| `x86_64-macos` | `x86_64-apple-darwin` |
| `aarch64-macos` | `aarch64-apple-darwin` |

## Apéndice B: Códigos de Error de la CLI

| Código | Significado |
|---|---|
| `E001` | `tsuki-config.toml` no encontrado |
| `E002` | Error de parseo en `tsuki-config.toml` |
| `E003` | Paquete no encontrado en ningún registry |
| `E004` | Conflicto de versiones en dependencias |
| `E005` | Checksum inválido al descargar paquete |
| `E006` | `GITHUB_TOKEN` no configurado |
| `E007` | Fallo en compilación para un target |
| `E008` | Registry no accesible (sin red o URL inválida) |

---

*Documento generado para tsuki v3.0 — Última actualización: Febrero 2026*