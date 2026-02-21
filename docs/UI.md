# Goduino IDE — Frontend UI Specification

> A browser-based IDE similar to VS Code for writing Arduino firmware in Go.  
> Powered by **Monaco Editor**, backed by a **Go LSP server** and the
> **Rust transpiler core**.

---

## Table of Contents
1. [Technology stack](#technology-stack)
2. [Architecture](#architecture)
3. [Screen layout](#screen-layout)
4. [Features](#features)
5. [Monaco editor integration](#monaco-editor-integration)
6. [Language Server Protocol (LSP)](#language-server-protocol-lsp)
7. [Live preview / serial monitor](#live-preview--serial-monitor)
8. [Project & file explorer](#project--file-explorer)
9. [Build panel](#build-panel)
10. [Settings & theming](#settings--theming)
11. [Implementation roadmap](#implementation-roadmap)

---

## Technology stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Shell** | [Tauri](https://tauri.app) v2 (Rust) | Cross-platform desktop wrapper |
| **Frontend** | React 18 + TypeScript | SPA inside the Tauri WebView |
| **Code editor** | [Monaco Editor](https://microsoft.github.io/monaco-editor/) | Same engine as VS Code |
| **Styling** | Tailwind CSS + Radix UI | |
| **LSP backend** | Go (`golang.org/x/tools/gopls` + custom overlay) | Language intelligence |
| **Transpiler** | Rust `goduino_core` crate | Called via Tauri commands |
| **Serial** | `serialport` crate (Rust/Tauri) | Serial monitor |
| **State management** | Zustand | |
| **IPC (frontend ↔ Rust)** | Tauri `invoke` commands | |

### Alternative: Pure-web / Electron

If distributing as a web app instead of desktop:
- Replace Tauri with Electron or a local HTTP server (`gin`).
- Use WebAssembly build of `goduino_core` for the transpiler.
- Replace native serial with Web Serial API (Chrome 89+).

---

## Architecture

```
┌───────────────────────────────────────────────────────────┐
│  Tauri shell (Rust)                                        │
│  ┌──────────────────────┐   ┌──────────────────────────┐  │
│  │  WebView (React/TS)  │   │  Tauri commands           │  │
│  │  ─────────────────── │   │  ─────────────────────── │  │
│  │  Monaco Editor        │◄──►  transpile(src) → cpp    │  │
│  │  File Explorer        │   │  compile(cpp)  → hex     │  │
│  │  Build Panel          │   │  flash(port)             │  │
│  │  Serial Monitor       │   │  list_ports()            │  │
│  │  Board Selector       │   │  serial_read/write()     │  │
│  └──────────────────────┘   └──────────────────────────┘  │
│                                     │                       │
│                          ┌──────────┴───────────┐          │
│                          │  goduino_core (Rust)  │          │
│                          │  + arduino-cli        │          │
│                          └───────────────────────┘          │
└───────────────────────────────────────────────────────────┘
```

---

## Screen layout

```
┌──────────────────────────────────────────────────────────────────────┐
│ ● ● ●  Goduino IDE                                    [_][□][X]      │
├──────────────────────────────────────────────────────────────────────┤
│ [FILE] [EDIT] [VIEW] [BUILD] [FLASH] [HELP]          Board: [Uno ▾]  │
├──────────────────────────────────────────────────────────────────────┤
│        │                                      │                       │
│ EXPLOR │   src/main.go               ×        │  OUTPUT / MONITOR     │
│ ├─ src │ ─────────────────────────────────── │                       │
│ │  └ main.go│ 1  package main                │  ✓ Transpile  0.12s   │
│ ├─build │ 2                                  │  ✓ Compile    3.41s   │
│ └ goduino│ 3  import (                       │  ✓ Flash      1.87s   │
│   .json │ 4      "arduino"                   │                       │
│         │ 5      "fmt"                        │ ─── SERIAL ────────   │
│         │ 6  )                                │ [9600 ▾] [Open]       │
│         │ 7                                   │                       │
│         │ 8  const ledPin = 13                │ > Blink ready!        │
│         │ 9                                   │ > Blink ready!        │
│         │10  func setup() {                   │                       │
│         │11      arduino.pinMode(             │ [─────────────] [↵]   │
│         │12          ledPin, arduino.OUTPUT)  │                       │
│         │13      fmt.Println("Blink ready!")  ├───────────────────────┤
│         │14  }                                │ STATUS: Connected     │
│         │                                     │ Port: /dev/ttyUSB0   │
└─────────┴─────────────────────────────────────┴───────────────────────┘
```

### Panels
| Panel | Default | Description |
|-------|---------|-------------|
| **File Explorer** | Left sidebar | Project tree, right-click context menu |
| **Editor** | Center | Monaco with Go syntax + custom completions |
| **Output/Monitor** | Right / bottom | Build log + serial monitor (tabs) |
| **Status bar** | Bottom | Board, port, cursor pos, error count |

---

## Features

### Core editor features
- [ ] Syntax highlighting for Go (Monaco built-in language)
- [ ] Custom token coloring for Arduino-specific identifiers
- [ ] Auto-indent, bracket matching, code folding
- [ ] Multi-cursor editing
- [ ] Find & Replace (including regex)
- [ ] Go-to-definition (via LSP)
- [ ] Hover documentation (function signatures + Arduino pin descriptions)
- [ ] Inline diagnostics (red squiggles from LSP)
- [ ] Auto-import suggestions
- [ ] Code snippets (`setup`, `loop`, `for`, `if`, …)

### Build features
- [ ] One-click **Transpile** (Go → C++)
- [ ] One-click **Compile** (C++ → .hex via arduino-cli)
- [ ] One-click **Flash** (upload to board)
- [ ] Combined **Build & Flash** button (Ctrl+B)
- [ ] Output panel with color-coded log
- [ ] Error links: click on error → jump to source line

### Board management
- [ ] Dropdown to select target board
- [ ] Auto-detect connected boards
- [ ] Board info tooltip (flash, RAM, clock)
- [ ] Port selector with refresh

### Serial monitor
- [ ] Open/close serial connection
- [ ] Baud rate selector
- [ ] Line-ending selector (LF / CR / CRLF)
- [ ] Autoscroll toggle
- [ ] Send text / raw bytes
- [ ] Timestamp column
- [ ] Export log to file
- [ ] ASCII / HEX display toggle

### Project management
- [ ] New project wizard (board + template)
- [ ] Open folder
- [ ] File tree CRUD (new file, rename, delete)
- [ ] Recent projects list
- [ ] `goduino.json` visual editor (form-based)

---

## Monaco editor integration

### Register Go language

```typescript
// src/editor/GoLanguage.ts
import * as monaco from 'monaco-editor'

monaco.languages.register({ id: 'go' })

monaco.languages.setMonarchTokensProvider('go', {
  keywords: [
    'package','import','func','var','const','type',
    'if','else','for','range','switch','case','default',
    'return','break','continue','goto','defer','go',
    'struct','interface','map','chan','select',
    'true','false','nil',
  ],
  // ... full monarch grammar
})
```

### Arduino-aware completions

```typescript
// src/editor/ArduinoCompletions.ts

const ARDUINO_COMPLETIONS: monaco.languages.CompletionItem[] = [
  {
    label: 'arduino.digitalWrite',
    kind: monaco.languages.CompletionItemKind.Function,
    insertText: 'arduino.digitalWrite(${1:pin}, ${2:value})',
    insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    documentation: 'Write HIGH or LOW to a digital pin.',
  },
  {
    label: 'arduino.analogRead',
    kind: monaco.languages.CompletionItemKind.Function,
    insertText: 'arduino.analogRead(${1:pin})',
    insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
    documentation: 'Read an analog value (0–1023) from the given pin.',
  },
  // ... all mapped functions + constants
]

monaco.languages.registerCompletionItemProvider('go', {
  provideCompletionItems: (model, position) => ({
    suggestions: ARDUINO_COMPLETIONS,
  }),
})
```

### Live transpile preview (split view)

Optionally show C++ preview alongside Go:

```typescript
// After every debounced edit (500ms), call Tauri transpile command
const cpp = await invoke<string>('transpile', { source, board })
cppPreviewEditor.setValue(cpp)
```

---

## Language Server Protocol (LSP)

Run `gopls` (the official Go language server) with a custom layer that
understands Goduino's virtual packages (`"arduino"`, `"wire"`, etc.).

### Architecture

```
Monaco Editor (LSP client — vscode-languageclient)
      │  JSON-RPC over stdio or WebSocket
      ▼
goduino-lsp (Go binary)
      ├── Proxy all requests to gopls for standard Go intelligence
      └── Override/augment:
            ├── textDocument/completion  → inject arduino.* completions
            ├── textDocument/hover       → Arduino pin/function docs
            └── textDocument/definition → map to runtime source
```

### Synthetic package stubs

Generate Go stub files for each virtual package so `gopls` can
provide real type information:

```go
// ~/.cache/goduino/stubs/arduino/arduino.go  (auto-generated)
package arduino

// pinMode sets the mode of a digital pin.
func pinMode(pin int, mode int) {}

// digitalWrite writes HIGH or LOW to a digital pin.
func digitalWrite(pin int, value int) {}

// HIGH represents a logic-high voltage level (typically 5V or 3.3V).
const HIGH = 1
const LOW  = 0
const OUTPUT = 1
const INPUT  = 0
// ...
```

Point `GOPATH`/`GOROOT` at these stubs when launching `gopls`.

---

## Live preview / serial monitor

### Tauri serial command (Rust side)

```rust
// src-tauri/src/serial.rs
use tauri::State;
use serialport::SerialPort;

#[tauri::command]
fn list_ports() -> Vec<String> {
    serialport::available_ports()
        .unwrap_or_default()
        .into_iter()
        .map(|p| p.port_name)
        .collect()
}

#[tauri::command]
async fn serial_open(port: String, baud: u32, app: tauri::AppHandle) -> Result<(), String> {
    let mut sp = serialport::new(&port, baud).open()?;
    // spawn a task reading bytes and emitting tauri events
    tauri::async_runtime::spawn(async move {
        let mut buf = [0u8; 256];
        loop {
            match sp.read(&mut buf) {
                Ok(n) => app.emit_all("serial-data", &buf[..n]).ok(),
                Err(_) => break,
            };
        }
    });
    Ok(())
}
```

### React serial monitor component

```tsx
// src/components/SerialMonitor.tsx
import { useEffect, useState } from 'react'
import { invoke, listen } from '@tauri-apps/api'

export function SerialMonitor() {
  const [log, setLog] = useState<string[]>([])
  const [input, setInput] = useState('')

  useEffect(() => {
    const unlisten = listen<Uint8Array>('serial-data', (event) => {
      const text = new TextDecoder().decode(event.payload)
      setLog(prev => [...prev, text])
    })
    return () => { unlisten.then(f => f()) }
  }, [])

  const send = () => {
    invoke('serial_write', { data: input + '\n' })
    setInput('')
  }

  return (
    <div className="font-mono text-sm bg-black text-green-400 h-full overflow-auto p-2">
      {log.map((line, i) => <div key={i}>{line}</div>)}
      <div className="flex mt-2">
        <input value={input} onChange={e => setInput(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && send()}
          className="flex-1 bg-gray-900 text-white px-2 py-1 rounded-l" />
        <button onClick={send} className="bg-blue-600 px-3 py-1 rounded-r">↵</button>
      </div>
    </div>
  )
}
```

---

## Project & file explorer

```tsx
// src/components/FileExplorer.tsx

// Use @tauri-apps/api/fs for file operations
import { readDir, writeFile, removeFile, createDir } from '@tauri-apps/api/fs'

// Right-click context menu actions:
// - New File
// - New Folder
// - Rename (F2)
// - Delete (Delete key)
// - Copy / Paste
// - Reveal in Finder / Explorer
```

---

## Build panel

```tsx
// src/components/BuildPanel.tsx

type BuildStep = 'transpile' | 'compile' | 'flash'
type StepStatus = 'idle' | 'running' | 'success' | 'error'

interface BuildState {
  steps: Record<BuildStep, StepStatus>
  log:   string[]
}
```

**Keyboard shortcuts:**
| Shortcut | Action |
|----------|--------|
| `Ctrl+B` | Build & Flash |
| `Ctrl+Shift+B` | Build only |
| `Ctrl+Shift+T` | Transpile only |
| `Ctrl+Shift+U` | Flash only |
| `Ctrl+M` | Open serial monitor |

---

## Settings & theming

Store settings in Tauri's app data directory:

```json
// settings.json
{
  "theme": "dark",
  "font_size": 14,
  "font_family": "JetBrains Mono, Fira Code, monospace",
  "tab_size": 4,
  "auto_save": true,
  "auto_save_delay_ms": 1000,
  "default_board": "uno",
  "default_baud": 9600,
  "arduino_cli_path": "",
  "line_ending": "LF",
  "cpp_preview": false
}
```

Themes: Dark (default), Light, High Contrast, Dracula, Solarized.

---

## Implementation roadmap

### Phase 1 — MVP (minimum viable IDE)
- [ ] Tauri scaffold + React + Monaco
- [ ] File explorer (open folder, read files)
- [ ] Monaco with Go highlighting + Arduino completions
- [ ] Transpile button → call Rust core → show C++ in output tab
- [ ] Board selector dropdown

### Phase 2 — Build & Flash
- [ ] Compile via arduino-cli
- [ ] Flash via arduino-cli
- [ ] Board auto-detection
- [ ] Serial monitor (read-only)

### Phase 3 — Intelligence
- [ ] LSP integration (gopls proxy)
- [ ] Go-to-definition for arduino.* symbols
- [ ] Hover docs for Arduino functions/constants
- [ ] Inline error diagnostics

### Phase 4 — Polish
- [ ] Serial monitor (write + timestamps)
- [ ] C++ split-view preview
- [ ] Settings panel
- [ ] Multiple theme support
- [ ] Project templates wizard
- [ ] Keyboard shortcuts panel
- [ ] Tutorial / welcome screen

### Phase 5 — Web version
- [ ] Compile `goduino_core` to WASM
- [ ] Replace Tauri commands with WASM calls
- [ ] Use Web Serial API for serial monitor
- [ ] Deploy as hosted web app