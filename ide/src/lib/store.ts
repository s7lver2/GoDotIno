'use client'
import { create } from 'zustand'

export type Screen = 'welcome' | 'ide' | 'settings'
export type SidebarTab = 'files' | 'git' | 'packages'
export type BottomTab = 'output' | 'problems' | 'terminal'
export type SettingsTab = 'cli' | 'defaults' | 'editor'

export interface FileNode {
  id: string
  name: string
  type: 'file' | 'dir'
  ext?: string
  content?: string
  git?: 'A' | 'M' | 'D'
  open?: boolean
  children?: string[]
}

export interface TabItem {
  fileId: string
  name: string
  ext: string
  content: string
  modified: boolean
}

export interface GitChange {
  letter: 'A' | 'M' | 'D'
  name: string
  path: string
}

export interface GitCommitNode {
  hash: string
  shortHash: string
  message: string
  author: string
  time: string
  branch?: string
  parents: string[]
  isMerge?: boolean
}

export interface LogLine {
  id: string
  type: 'ok' | 'err' | 'warn' | 'info'
  time: string
  msg: string
}

export interface Problem {
  id: string
  severity: 'error' | 'warning' | 'info'
  file: string
  line: number
  col: number
  message: string
}

export interface PackageEntry {
  name: string
  desc: string
  version: string
  installed: boolean
  installing?: boolean
}

export interface SettingsState {
  tsukiPath: string
  tsukiCorePath: string
  arduinoCliPath: string
  avrDudePath: string
  defaultBoard: string
  defaultBaud: string
  cppStd: string
  verbose: boolean
  autoDetect: boolean
  color: boolean
  libsDir: string
  registryUrl: string
  verifySignatures: boolean
  fontSize: number
  tabSize: number
  minimap: boolean
  wordWrap: boolean
  formatOnSave: boolean
  trimWhitespace: boolean
}

interface AppState {
  theme: 'dark' | 'light'
  toggleTheme: () => void
  screen: Screen
  setScreen: (s: Screen) => void
  projectName: string
  projectPath: string   // absolute path on disk, e.g. C:\Users\...\MyProject
  board: string
  backend: string
  gitInit: boolean
  setBoard: (b: string) => void
  setBackend: (b: string) => void
  setProjectPath: (p: string) => void
  loadProject: (name: string, board: string, template: string, backend?: string, gitInit?: boolean, path?: string) => void
  sidebarOpen: boolean
  sidebarTab: SidebarTab
  toggleSidebar: (tab: SidebarTab) => void
  bottomTab: BottomTab
  setBottomTab: (t: BottomTab) => void
  settingsTab: SettingsTab
  setSettingsTab: (t: SettingsTab) => void
  tree: FileNode[]
  openTabs: TabItem[]
  activeTabIdx: number
  openFile: (id: string) => void
  closeTab: (idx: number) => void
  updateTabContent: (idx: number, content: string) => void
  addFile: (name: string) => void
  addFolder: (name: string) => void
  deleteActive: () => void
  renameNode: (id: string, newName: string) => void
  gitChanges: GitChange[]
  gitBranch: string
  commitHistory: GitCommitNode[]
  doCommit: (msg: string) => void
  logs: LogLine[]
  addLog: (type: LogLine['type'], msg: string) => void
  clearLogs: () => void
  problems: Problem[]
  setProblems: (problems: Problem[]) => void
  bottomHeight: number
  setBottomHeight: (h: number) => void
  terminalLines: string[]
  addTerminalLine: (line: string) => void
  clearTerminal: () => void
  settings: SettingsState
  updateSetting: <K extends keyof SettingsState>(key: K, value: SettingsState[K]) => void
  packages: PackageEntry[]
  togglePackage: (name: string) => void
  setPackageInstalling: (name: string, installing: boolean) => void
}

const TEMPLATES: Record<string, string> = {
  blink: `package main\n\nimport "arduino"\n\nconst ledPin = 13\nconst interval = 500 // ms\n\nfunc setup() {\n    arduino.PinMode(ledPin, arduino.OUTPUT)\n    arduino.Serial.Begin(9600)\n    arduino.Serial.Println("Blink ready!")\n}\n\nfunc loop() {\n    arduino.DigitalWrite(ledPin, arduino.HIGH)\n    arduino.Delay(interval)\n    arduino.DigitalWrite(ledPin, arduino.LOW)\n    arduino.Delay(interval)\n}`,
  sensor: `package main\n\nimport (\n    "arduino"\n    "fmt"\n)\n\nfunc setup() {\n    arduino.Serial.Begin(9600)\n}\n\nfunc loop() {\n    val := arduino.AnalogRead(arduino.A0)\n    fmt.Println("sensor:", val)\n    arduino.Delay(500)\n}`,
  serial: `package main\n\nimport (\n    "arduino"\n    "fmt"\n)\n\nfunc setup() {\n    arduino.Serial.Begin(115200)\n    fmt.Println("Serial ready!")\n}\n\nfunc loop() {\n    if arduino.Serial.Available() > 0 {\n        b := arduino.Serial.Read()\n        fmt.Print(string(b))\n    }\n}`,
  servo: `package main\n\nimport (\n    "arduino"\n    "Servo"\n)\n\nvar s Servo.Servo\n\nfunc setup() {\n    s.Attach(9)\n}\n\nfunc loop() {\n    for pos := 0; pos <= 180; pos++ {\n        s.Write(pos)\n        arduino.Delay(15)\n    }\n    for pos := 180; pos >= 0; pos-- {\n        s.Write(pos)\n        arduino.Delay(15)\n    }\n}`,
  empty: `package main\n\nimport "arduino"\n\nfunc setup() {\n    // setup code here\n}\n\nfunc loop() {\n    // main loop\n}`,
}

function manifest(name: string, board: string, backend = 'tsuki-flash') {
  return JSON.stringify({ name, version: '0.1.0', board, backend, go_version: '1.21', packages: [] }, null, 2)
}

function ts() {
  return new Date().toTimeString().slice(0, 8)
}

let logId = 0

const DEFAULT_PACKAGES: PackageEntry[] = [
  { name: 'dht',           desc: 'DHT11/DHT22 temperature & humidity sensor', version: 'v1.0.0', installed: true  },
  { name: 'ws2812',        desc: 'NeoPixel / WS2812 LED strip driver',        version: 'v1.0.0', installed: true  },
  { name: 'u8g2',          desc: 'OLED / LCD display library (SSD1306, etc)', version: 'v1.0.0', installed: true  },
  { name: 'Servo',         desc: 'Servo motor control',                       version: 'v1.0.0', installed: false },
  { name: 'LiquidCrystal', desc: 'LCD display (parallel, HD44780)',           version: 'v1.0.0', installed: false },
  { name: 'IRremote',      desc: 'Infrared remote receiver/transmitter',      version: 'v1.0.0', installed: false },
  { name: 'RTClib',        desc: 'Real-time clock — DS1307 / DS3231',         version: 'v1.0.0', installed: false },
  { name: 'MFRC522',       desc: 'SPI RFID reader/writer',                    version: 'v1.0.0', installed: false },
  { name: 'Stepper',       desc: 'Stepper motor driver (4-wire)',              version: 'v1.0.0', installed: false },
  { name: 'Adafruit_GFX', desc: 'Adafruit graphics core library',            version: 'v1.0.0', installed: false },
]

const DEFAULT_SETTINGS: SettingsState = {
  tsukiPath: 'tsuki',
  tsukiCorePath: '',
  arduinoCliPath: 'arduino-cli',
  avrDudePath: '',
  defaultBoard: 'uno',
  defaultBaud: '9600',
  cppStd: 'c++17',
  verbose: false,
  autoDetect: true,
  color: true,
  libsDir: '~/.tsuki/libs',
  registryUrl: 'https://registry.goduino.dev/v1/index.json',
  verifySignatures: true,
  fontSize: 13,
  tabSize: 2,
  minimap: false,
  wordWrap: false,
  formatOnSave: true,
  trimWhitespace: true,
}

export const useStore = create<AppState>((set, get) => ({
  theme: 'dark',
  toggleTheme: () => {
    const next = get().theme === 'dark' ? 'light' : 'dark'
    set({ theme: next })
    document.documentElement.className = next
    try { localStorage.setItem('gdi-theme', next) } catch {}
  },

  screen: 'welcome',
  setScreen: (screen) => set({ screen }),

  projectName: '',
  projectPath: '',
  board: 'uno',
  backend: 'tsuki-flash',
  gitInit: true,
  setBoard: (board) => set({ board }),
  setBackend: (backend) => set({ backend }),
  setProjectPath: (projectPath) => set({ projectPath }),

  loadProject: (name, board, template, backend = 'tsuki-flash', gitInit = true, path = '') => {
    const tree: FileNode[] = [
      { id: 'root',      name,                type: 'dir',  open: true,  children: ['manifest', 'src', 'build', 'gitignore'] },
      { id: 'manifest',  name: 'goduino.json', type: 'file', ext: 'json', content: manifest(name, board, backend), git: 'A' },
      { id: 'src',       name: 'src',           type: 'dir',  open: true,  children: ['main'] },
      { id: 'main',      name: 'main.go',        type: 'file', ext: 'go',   content: TEMPLATES[template] || TEMPLATES.blink, git: 'A' },
      { id: 'build',     name: 'build',          type: 'dir',  open: false, children: [] },
      { id: 'gitignore', name: '.gitignore',     type: 'file', ext: 'txt',  content: 'build/\n*.hex\n*.bin\n*.elf\n', git: 'A' },
    ]
    const gitChanges: GitChange[] = [
      { letter: 'A', name: 'main.go',       path: 'src/main.go'   },
      { letter: 'A', name: 'goduino.json',  path: 'goduino.json'  },
      { letter: 'A', name: '.gitignore',    path: '.gitignore'    },
    ]
    set({
      projectName: name, projectPath: path, board, backend, gitInit, tree, gitChanges,
      commitHistory: [], openTabs: [], activeTabIdx: -1,
      screen: 'ide', logs: [], terminalLines: [],
    })
    setTimeout(() => get().openFile('main'), 50)
    get().addLog('info', `Project "${name}" loaded · Board: ${board} · Backend: ${backend}`)
    get().addLog('ok', gitInit ? 'Git repo initialized · Ready.' : 'Ready (no git). Run tsuki check to validate.')
  },

  sidebarOpen: true,
  sidebarTab: 'files',
  toggleSidebar: (tab) => {
    const { sidebarOpen, sidebarTab } = get()
    if (sidebarOpen && sidebarTab === tab) set({ sidebarOpen: false })
    else set({ sidebarOpen: true, sidebarTab: tab })
  },

  bottomTab: 'output',
  setBottomTab: (bottomTab) => set({ bottomTab }),

  settingsTab: 'cli',
  setSettingsTab: (settingsTab) => set({ settingsTab }),

  tree: [],
  openTabs: [],
  activeTabIdx: -1,

  openFile: (id) => {
    const node = get().tree.find(n => n.id === id)
    if (!node || node.type === 'dir') return
    const existing = get().openTabs.findIndex(t => t.fileId === id)
    if (existing >= 0) { set({ activeTabIdx: existing }); return }
    const tab: TabItem = { fileId: id, name: node.name, ext: node.ext || '', content: node.content || '', modified: false }
    const tabs = [...get().openTabs, tab]
    set({ openTabs: tabs, activeTabIdx: tabs.length - 1 })
  },

  closeTab: (idx) => {
    const tabs = get().openTabs.filter((_, i) => i !== idx)
    let active = get().activeTabIdx
    if (active >= tabs.length) active = tabs.length - 1
    set({ openTabs: tabs, activeTabIdx: active })
  },

  updateTabContent: (idx, content) => {
    const tabs = [...get().openTabs]
    const tree = [...get().tree]
    tabs[idx] = { ...tabs[idx], content, modified: true }
    const nodeIdx = tree.findIndex(n => n.id === tabs[idx].fileId)
    if (nodeIdx >= 0) tree[nodeIdx] = { ...tree[nodeIdx], content, git: tree[nodeIdx].git || 'M' }
    set({ openTabs: tabs, tree })
  },

  addFile: (name) => {
    const id = 'f_' + Date.now()
    const ext = name.split('.').pop() || 'txt'
    const node: FileNode = { id, name, type: 'file', ext, content: '', git: 'A' }
    const tree = [...get().tree, node]
    const src = tree.find(n => n.id === 'src')
    if (src) src.children = [...(src.children || []), id]
    const gitChanges = [...get().gitChanges, { letter: 'A' as const, name, path: `src/${name}` }]
    set({ tree, gitChanges })
    get().openFile(id)
  },

  addFolder: (name) => {
    const id = 'd_' + Date.now()
    const node: FileNode = { id, name, type: 'dir', open: false, children: [] }
    const tree = [...get().tree, node]
    const root = tree.find(n => n.id === 'root')
    if (root) root.children = [...(root.children || []), id]
    set({ tree })
  },

  deleteActive: () => {
    const { activeTabIdx, openTabs, tree } = get()
    if (activeTabIdx < 0) return
    const tab = openTabs[activeTabIdx]
    const newTree = tree.filter(n => n.id !== tab.fileId).map(n => ({ ...n, children: n.children?.filter(c => c !== tab.fileId) }))
    get().closeTab(activeTabIdx)
    set({ tree: newTree })
  },

  renameNode: (id, newName) => {
    const tree = get().tree.map(n => n.id === id ? { ...n, name: newName, git: n.git || 'M' as const } : n)
    const openTabs = get().openTabs.map(t => t.fileId === id ? { ...t, name: newName } : t)
    set({ tree, openTabs })
  },

  gitChanges: [],
  gitBranch: 'main',
  commitHistory: [],

  doCommit: (msg) => {
    const tree = get().tree.map(n => ({ ...n, git: undefined }))
    const changedFiles = get().gitChanges.length
    const hash = Math.random().toString(16).slice(2, 9)
    const timeStr = new Date().toLocaleTimeString('en', { hour: '2-digit', minute: '2-digit' })
    const newCommit: GitCommitNode = {
      hash, shortHash: hash.slice(0, 7), message: msg, author: 'you', time: timeStr,
      branch: get().gitBranch, parents: get().commitHistory.length > 0 ? [get().commitHistory[0].hash] : [],
    }
    set({ gitChanges: [], tree, commitHistory: [newCommit, ...get().commitHistory] })
    get().addLog('ok', `[${get().gitBranch}] ${hash.slice(0,7)} ${msg} (${changedFiles} file${changedFiles !== 1 ? 's' : ''})`)
  },

  logs: [],
  addLog: (type, msg) => {
    const line: LogLine = { id: String(logId++), type, time: ts(), msg }
    set({ logs: [...get().logs, line] })
  },
  clearLogs: () => set({ logs: [] }),

  problems: [],
  setProblems: (problems) => set({ problems }),

  bottomHeight: 200,
  setBottomHeight: (h) => set({ bottomHeight: Math.max(80, Math.min(h, 600)) }),

  terminalLines: [],
  addTerminalLine: (line) => set((s) => ({ terminalLines: [...s.terminalLines, line] })),
  clearTerminal: () => set({ terminalLines: [] }),

  settings: DEFAULT_SETTINGS,
  updateSetting: (key, value) => {
    set((s) => {
      const next = { ...s.settings, [key]: value }
      // Persist asynchronously — import is at top level to avoid circular dep issues
      import('@/lib/tauri').then(({ saveSettings }) => saveSettings(next)).catch(() => {})
      return { settings: next }
    })
  },

  packages: DEFAULT_PACKAGES,
  togglePackage: (name) => set((s) => ({
    packages: s.packages.map(p => p.name === name ? { ...p, installed: !p.installed } : p)
  })),
  setPackageInstalling: (name, installing) => set((s) => ({
    packages: s.packages.map(p => p.name === name ? { ...p, installing } : p)
  })),
}))