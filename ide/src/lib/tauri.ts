/**
 * Tauri v1 bridge.
 *
 * En Tauri v1 el webview expone window.__TAURI__ con la siguiente estructura:
 *   window.__TAURI__.invoke(cmd, args)   ← forma principal en v1.x
 *   window.__TAURI__.tauri.invoke(...)   ← alias en algunas versiones
 *
 * IMPORTANTE: La función se llama en runtime, no en import-time, para que
 * Next.js SSR no pete (window no existe en el servidor).
 */

declare global {
  interface Window {
    __TAURI__?: any
  }
}

// ── Detección robusta ─────────────────────────────────────────────────────────

function getInvoke(): ((cmd: string, args?: unknown) => Promise<unknown>) | null {
  if (typeof window === 'undefined') return null
  const t = window.__TAURI__
  if (!t) return null
  // Tauri v1: invoke está directamente en __TAURI__
  if (typeof t.invoke === 'function') return t.invoke.bind(t)
  // Algunas builds lo ponen en __TAURI__.tauri.invoke
  if (typeof t.tauri?.invoke === 'function') return t.tauri.invoke.bind(t.tauri)
  return null
}

function getListen(): ((event: string, cb: (e: { payload: unknown }) => void) => Promise<() => void>) | null {
  if (typeof window === 'undefined') return null
  const t = window.__TAURI__
  if (!t) return null
  if (typeof t.event?.listen === 'function') return t.event.listen.bind(t.event)
  return null
}

export function isTauri(): boolean {
  return getInvoke() !== null
}

// Debug: loguea en consola al cargar para diagnosticar en la app compilada
if (typeof window !== 'undefined') {
  setTimeout(() => {
    const t = window.__TAURI__
    if (t) {
      console.log('[tsuki-ide] Tauri detected. Keys:', Object.keys(t))
      console.log('[tsuki-ide] invoke:', typeof t.invoke, '| tauri.invoke:', typeof t.tauri?.invoke)
    } else {
      console.log('[tsuki-ide] No window.__TAURI__ — running in browser mode')
    }
  }, 0)
}

async function invoke<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const fn = getInvoke()
  if (!fn) throw new Error('[tsuki-ide] Tauri invoke not available')
  return fn(cmd, args) as Promise<T>
}

async function listen(event: string, cb: (payload: unknown) => void): Promise<() => void> {
  const fn = getListen()
  if (!fn) return () => {}
  return fn(event, (e) => cb(e.payload))
}

// ── Mock lines para dev en browser ───────────────────────────────────────────

function mockLines(args: string[]): string[] {
  const sub  = args[0] ?? ''
  const rest = args.slice(1)

  const boardIdx = rest.indexOf('--board')
  const board    = boardIdx >= 0 ? rest[boardIdx + 1] ?? 'uno' : 'uno'
  const baudIdx  = rest.indexOf('--baud')
  const baud     = baudIdx >= 0 ? rest[baudIdx + 1] ?? '9600' : '9600'

  // tsuki build  /  tsuki build --compile
  if (sub === 'build') {
    const compile = rest.includes('--compile')
    const cppsStd = rest.find(a => a.startsWith('--cpp-std'))?.split('=')[1] ?? 'c++11'
    const base = [
      `tsuki: reading tsuki_package.json`,
      `tsuki-core: parsing src/main.go…`,
      `tsuki-core: transpile OK  → build/main.cpp  (8 ms)`,
    ]
    if (compile) return [...base,
      `tsuki-flash: compiling for ${board}  std=${cppsStd}…`,
      `tsuki-flash: core.a cached (SDK 1.8.6)`,
      `tsuki-flash: sketch objects…`,
      `tsuki-flash: linking…`,
      `Sketch uses 1 284 bytes (3%) of program storage space. Maximum is 32 256 bytes.`,
      `Global variables use 188 bytes (9%) of dynamic memory, leaving 1 860 bytes.`,
      `✓  build/${board}.hex`,
    ]
    return [...base, `✓  build/main.cpp  (transpile only — use --compile for firmware)`]
  }

  // tsuki flash
  if (sub === 'flash') return [
    `tsuki: reading tsuki_package.json`,
    `tsuki-flash: scanning serial ports…`,
    `tsuki-flash: found  ${board}  COM3 / /dev/ttyUSB0  (1A86:7523)`,
    `tsuki-flash: uploading build/${board}.hex  (1 284 bytes)…`,
    `avrdude: AVR device initialized and ready to accept instructions`,
    `avrdude: 1 284 bytes of flash written`,
    `avrdude done.  Thank you.`,
    `✓  flash complete`,
  ]

  // tsuki check
  if (sub === 'check') return [
    `tsuki: reading tsuki_package.json`,
    `tsuki-core: parsing src/main.go…`,
    `✓  syntax OK`,
    `✓  setup() present`,
    `✓  loop() present`,
    `✓  0 errors, 0 warnings`,
  ]

  // tsuki monitor
  if (sub === 'monitor') return [
    `tsuki: opening serial monitor  ${baud} baud…`,
    `tsuki-flash: port  COM3 / /dev/ttyUSB0  detected`,
    `Connected. Ctrl+C to exit.`,
    `[00:00:01] Hello from tsuki!`,
    `[00:00:02] loop: tick`,
    `[00:00:03] loop: tick`,
  ]

  // tsuki pkg install <n>
  if (sub === 'pkg') {
    const pkgSub  = rest[0] ?? ''
    const pkgName = rest[1] ?? ''
    if (pkgSub === 'install') return [
      `Fetching ${pkgName} from registry…`,
      `Verifying SHA-256…`,
      `Installing → ~/.local/share/tsuki/libs/${pkgName}/1.0.0/godotinolib.toml`,
      `✓  ${pkgName} v1.0.0 installed`,
    ]
    if (pkgSub === 'list') return [
      `Installed packages:`,
      `  dht     1.0.0   DHT11/DHT22 temperature & humidity`,
      `  ws2812  1.0.0   NeoPixel / WS2812 LED strip`,
      `  u8g2    1.0.0   OLED / LCD display`,
    ]
    if (pkgSub === 'search') return [
      `Registry: https://raw.githubusercontent.com/s7lver2/GoDotIno/main/pkg/packages.json`,
      `  dht       v1.0.0  DHT11/DHT22 sensor`,
      `  ws2812    v1.0.0  NeoPixel LED strip`,
      `  u8g2      v1.0.0  OLED display (SSD1306)`,
      `  Servo     v1.0.0  Servo motor`,
      `  IRremote  v1.0.0  IR receiver/transmitter`,
      `  bme280    v1.0.0  BME280 pressure/humidity/temp`,
    ]
  }

  // tsuki deps add/remove/list
  if (sub === 'deps') {
    const depSub  = rest[0] ?? ''
    const depName = rest[1] ?? ''
    if (depSub === 'add') return [
      `Resolving ${depName}…`,
      `✓  Added ${depName} ^1.0.0  → tsuki_package.json`,
    ]
    if (depSub === 'remove') return [`✓  Removed ${depName}  → tsuki_package.json updated`]
    if (depSub === 'list') return [
      `Dependencies in tsuki_package.json:`,
      `  dht    ^1.0.0`,
      `  ws2812 ^1.0.0`,
    ]
  }

  // tsuki config
  if (sub === 'config') {
    const cfgSub = rest[0] ?? ''
    if (cfgSub === 'list') return [
      `libs_dir         ~/.local/share/tsuki/libs`,
      `registry_url     https://raw.githubusercontent.com/s7lver2/GoDotIno/main/pkg/packages.json`,
      `verify_sigs      false`,
    ]
    if (cfgSub === 'set') return [`✓  ${rest[1]} = ${rest[2]}`]
  }

  // tsuki clean
  if (sub === 'clean') return [`Removing build/…`, `✓  clean`]

  // tsuki init
  if (sub === 'init') return [
    `Creating ${rest[0] ?? 'project'}/…`,
    `Writing tsuki_package.json`,
    `Writing src/main.go`,
    `✓  project ready`,
  ]

  return [`[mock] tsuki ${args.join(' ')}`]
}

// ── Public API ────────────────────────────────────────────────────────────────

export interface ProcessHandle {
  pid: number
  done: Promise<number>
  write: (line: string) => Promise<void>
  kill: () => Promise<void>
  dispose: () => void
}

/**
 * Spawn a process and stream its output line by line.
 *
 * En la app compilada (Tauri): lanza el proceso real vía spawn_process,
 * escucha los eventos proc://<id>:stdout|stderr|done.
 *
 * En browser dev: emite las líneas del mock con delays realistas.
 *
 * @param cmd   Ejecutable (p.ej. "tsuki" o el path completo)
 * @param args  Argumentos (p.ej. ["build", "--compile", "--board", "uno"])
 * @param cwd   Directorio del proyecto (root donde está tsuki_package.json)
 * @param onLine  Callback por cada línea — (text, isStderr)
 */
export async function spawnProcess(
  cmd: string,
  args: string[],
  cwd: string | undefined,
  onLine: (line: string, isErr: boolean) => void,
): Promise<ProcessHandle> {

  // ── browser / dev mock ────────────────────────────────────────────────────
  if (!isTauri()) {
    const lines = mockLines(args)  // args[0] = subcommand
    let disposed = false
    let resolveDone!: (code: number) => void
    const done = new Promise<number>(r => { resolveDone = r })

    lines.forEach((line, i) => {
      setTimeout(() => {
        if (disposed) return
        onLine(line, false)
        if (i === lines.length - 1) setTimeout(() => resolveDone(0), 60)
      }, 60 + i * 80)
    })

    return {
      pid: Math.floor(Math.random() * 90000) + 10000,
      done,
      write: async (line) => { if (!disposed) onLine(`← ${line}`, false) },
      kill:  async () => { disposed = true; resolveDone(130) },
      dispose: () => { disposed = true },
    }
  }

  // ── Tauri: proceso real con streaming ─────────────────────────────────────
  const eventId = `${Date.now()}-${Math.random().toString(36).slice(2)}`
  const unsubs: Array<() => void> = []

  const outU  = await listen(`proc://${eventId}:stdout`, l => onLine(l as string, false))
  const errU  = await listen(`proc://${eventId}:stderr`, l => onLine(l as string, true))
  unsubs.push(outU, errU)

  let resolveDone!: (code: number) => void
  const done = new Promise<number>(r => { resolveDone = r })
  const doneU = await listen(`proc://${eventId}:done`, code => resolveDone(code as number))
  unsubs.push(doneU)

  const pid = await invoke<number>('spawn_process', {
    cmd, args, cwd: cwd ?? null, eventId,
  })

  return {
    pid,
    done,
    write:   async (line) => invoke<void>('write_stdin', { pid, data: line }),
    kill:    async () => invoke<void>('kill_process', { pid }),
    dispose: () => unsubs.forEach(f => f()),
  }
}

// ── Utilidades simples ────────────────────────────────────────────────────────

export async function detectTool(name: string): Promise<string> {
  if (!isTauri()) return `${name} v0.4.2`
  return invoke<string>('detect_tool', { name })
}

export async function pickFolder(): Promise<string | null> {
  if (!isTauri()) return null
  return invoke<string | null>('pick_folder')
}

export async function readFile(path: string): Promise<string> {
  if (!isTauri()) return ''
  return invoke<string>('read_file', { path })
}

export async function writeFile(path: string, content: string): Promise<void> {
  if (!isTauri()) return
  return invoke<void>('write_file', { path, content })
}

export async function loadSettings(): Promise<string> {
  if (!isTauri()) {
    try { return localStorage.getItem('gdi-settings') ?? '{}' } catch { return '{}' }
  }
  return invoke<string>('load_settings')
}

export async function saveSettings(settings: unknown): Promise<void> {
  const json = JSON.stringify(settings, null, 2)
  if (!isTauri()) {
    try { localStorage.setItem('gdi-settings', json) } catch {}
    return
  }
  return invoke<void>('save_settings', { settings: json })
}