'use client'
import { useStore, BottomTab } from '@/lib/store'
import { useEffect, useRef, useState, useCallback } from 'react'
import { IconBtn } from '@/components/ui/primitives'
import { Trash2, GripHorizontal, AlertTriangle, Info, AlertCircle, Square } from 'lucide-react'
import { clsx } from 'clsx'
import { spawnProcess, type ProcessHandle } from '@/lib/tauri'

// ── Tab config ────────────────────────────────────────────────────────────────

const TABS: { id: BottomTab; label: string }[] = [
  { id: 'output',   label: 'Output'   },
  { id: 'problems', label: 'Problems' },
  { id: 'terminal', label: 'Terminal' },
]

// ── Resize handle ─────────────────────────────────────────────────────────────

function ResizeHandle() {
  const { setBottomHeight, bottomHeight } = useStore()
  const dragging = useRef(false)
  const startY   = useRef(0)
  const startH   = useRef(0)

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    dragging.current = true
    startY.current   = e.clientY
    startH.current   = bottomHeight
    document.body.style.cursor     = 'row-resize'
    document.body.style.userSelect = 'none'
  }, [bottomHeight])

  useEffect(() => {
    function onMove(e: MouseEvent) {
      if (!dragging.current) return
      setBottomHeight(startH.current + (startY.current - e.clientY))
    }
    function onUp() {
      dragging.current = false
      document.body.style.cursor     = ''
      document.body.style.userSelect = ''
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup',   onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup',   onUp)
    }
  }, [setBottomHeight])

  return (
    <div
      onMouseDown={onMouseDown}
      className="h-[3px] flex items-center justify-center cursor-row-resize border-t border-[var(--border)] hover:border-[var(--fg-faint)] group transition-colors flex-shrink-0"
    >
      <GripHorizontal size={12} className="text-[var(--fg-faint)] opacity-0 group-hover:opacity-100 transition-opacity" />
    </div>
  )
}

// ── Terminal line type ────────────────────────────────────────────────────────

interface TerminalLine {
  id:   number
  text: string
  type: 'out' | 'err' | 'info' | 'prompt'
}

let _lineId = 0

// ── Terminal component ────────────────────────────────────────────────────────

function Terminal() {
  const { projectPath, settings, addLog, setBottomTab } = useStore()

  const [lines,   setLines  ] = useState<TerminalLine[]>([])
  const [input,   setInput  ] = useState('')
  const [running, setRunning] = useState(false)
  const [history, setHistory] = useState<string[]>([])
  const [histIdx, setHistIdx] = useState(-1)

  const processRef = useRef<ProcessHandle | null>(null)
  const endRef     = useRef<HTMLDivElement>(null)
  const inputRef   = useRef<HTMLInputElement>(null)

  // ── Refs para callbacks estables ─────────────────────────────────────────
  // Guardamos los state-setters más recientes en refs para que spawnInTerminal
  // (que se expone en window) siempre tenga acceso al estado actual sin
  // necesitar recrear la función en cada render.
  const addLineRef    = useRef((text: string, type: TerminalLine['type'] = 'out') => {
    setLines(l => [...l, { id: _lineId++, text, type }])
  })
  const setRunningRef = useRef(setRunning)
  const addLogRef     = useRef(addLog)
  const projectPathRef = useRef(projectPath)
  const setBottomTabRef = useRef(setBottomTab)

  useEffect(() => { addLogRef.current = addLog },          [addLog])
  useEffect(() => { projectPathRef.current = projectPath }, [projectPath])
  useEffect(() => { setBottomTabRef.current = setBottomTab },[setBottomTab])

  // Auto-scroll
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [lines])

  // ── spawnInTerminal: función estable expuesta en window ───────────────────
  // Usamos useRef para que la referencia que se guarda en window.__terminalSpawn
  // nunca quede stale. El contenido se actualiza vía los refs de arriba.
  const spawnInTerminalRef = useRef(async (
    cmd: string,
    args: string[],
    cwd?: string,
  ): Promise<ProcessHandle> => {
    const addLine     = addLineRef.current
    const setRunning_ = setRunningRef.current
    const addLog_     = addLogRef.current
    const setBottomTab_ = setBottomTabRef.current
    const cwd_ = cwd ?? projectPathRef.current ?? undefined

    // Mata el proceso anterior si hay uno
    if (processRef.current) {
      await processRef.current.kill()
      processRef.current.dispose()
      processRef.current = null
    }

    addLine(`❯ ${[cmd, ...args].join(' ')}`, 'prompt')
    setRunning_(true)
    setBottomTab_('terminal')

    const handle = await spawnProcess(
      cmd, args, cwd_,
      (line, isErr) => {
        addLine(line, isErr ? 'err' : 'out')
        addLog_(isErr ? 'err' : 'ok', line)
      },
    )

    processRef.current = handle

    handle.done.then(code => {
      processRef.current = null
      setRunningRef.current(false)
      if (code !== 0 && code !== 130) {
        addLine(`[exit ${code}]`, 'err')
        addLog_('err', `process exited with code ${code}`)
      } else {
        addLine('', 'out')
      }
    })

    return handle
  })

  // Registrar en window UNA SOLA VEZ (el ref nunca cambia)
  useEffect(() => {
    ;(window as any).__terminalSpawn = (...a: Parameters<typeof spawnInTerminalRef.current>) =>
      spawnInTerminalRef.current(...a)
    return () => { delete (window as any).__terminalSpawn }
  }, []) // ← array vacío: sólo al montar/desmontar

  // ── Ejecutar comando escrito manualmente ──────────────────────────────────
  async function runCommand(raw: string) {
    const trimmed = raw.trim()
    if (!trimmed) return

    setHistory(h => [trimmed, ...h.slice(0, 49)])
    setHistIdx(-1)

    if (trimmed === 'clear' || trimmed === 'cls') { setLines([]); return }
    if (trimmed === 'help') {
      addLineRef.current(
        'Commands: tsuki build|build --compile|flash|check|monitor|pkg|deps|config, clear',
        'info',
      )
      return
    }

    const [exe, ...args] = trimmed.split(/\s+/)
    await spawnInTerminalRef.current(exe, args, projectPathRef.current ?? undefined)
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      if (running && processRef.current) {
        // Proceso interactivo — enviar input al stdin
        processRef.current.write(input).catch(() => {})
        addLineRef.current(input, 'prompt')
        setInput('')
      } else {
        const val = input
        setInput('')
        runCommand(val)
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const next = Math.min(histIdx + 1, history.length - 1)
      setHistIdx(next)
      setInput(history[next] ?? '')
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.max(histIdx - 1, -1)
      setHistIdx(next)
      setInput(next === -1 ? '' : history[next] ?? '')
    } else if (e.key === 'c' && e.ctrlKey) {
      if (processRef.current) {
        processRef.current.kill().then(() => {
          addLineRef.current('^C', 'info')
          setRunning(false)
          processRef.current = null
        })
      }
    }
  }

  async function stopProcess() {
    if (!processRef.current) return
    await processRef.current.kill()
    processRef.current.dispose()
    processRef.current = null
    addLineRef.current('^C  process terminated', 'info')
    setRunning(false)
  }

  const typeClass: Record<TerminalLine['type'], string> = {
    out:    'text-[var(--fg-muted)]',
    err:    'text-red-400',
    info:   'text-[var(--info,#60a5fa)]',
    prompt: 'text-[var(--fg)] font-medium',
  }

  return (
    <div
      className="flex-1 flex flex-col overflow-hidden font-mono text-xs select-text"
      onClick={() => inputRef.current?.focus()}
    >
      <div className="flex-1 overflow-y-auto px-3 py-2">
        {lines.map(l => (
          <div
            key={l.id}
            className={clsx('leading-[18px] whitespace-pre-wrap break-all', typeClass[l.type])}
          >
            {l.text}
          </div>
        ))}

        {/* Línea de input */}
        <div className="flex items-center gap-1.5 mt-0.5">
          <span className={clsx('flex-shrink-0 text-[10px]', running ? 'text-yellow-400' : 'text-green-400')}>
            {running ? '◉' : '❯'}
          </span>
          <input
            ref={inputRef}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={onKeyDown}
            className="flex-1 bg-transparent outline-none text-[var(--fg)] caret-[var(--fg)] border-0 font-mono text-xs"
            placeholder={running ? 'send input to process… (Ctrl+C to kill)' : ''}
            spellCheck={false}
            autoCorrect="off"
            autoCapitalize="off"
          />
          {running && (
            <button
              onClick={stopProcess}
              title="Kill process  (Ctrl+C)"
              className="flex items-center justify-center w-5 h-5 rounded text-red-400 hover:bg-[var(--hover)] border-0 bg-transparent cursor-pointer transition-colors"
            >
              <Square size={10} />
            </button>
          )}
        </div>

        <div ref={endRef} />
      </div>
    </div>
  )
}

// ── Problems tab ──────────────────────────────────────────────────────────────

function ProblemsTab() {
  const { problems } = useStore()

  if (problems.length === 0) {
    return (
      <div className="flex items-center gap-2 px-3 py-3 text-xs text-[var(--fg-faint)]">
        <span className="text-green-400">✓</span>No problems detected.
      </div>
    )
  }

  const icons = {
    error:   <AlertCircle   size={12} className="text-red-400  flex-shrink-0" />,
    warning: <AlertTriangle size={12} className="text-yellow-400 flex-shrink-0" />,
    info:    <Info          size={12} className="text-blue-400 flex-shrink-0" />,
  }

  return (
    <div className="flex-1 overflow-y-auto">
      {problems.map(p => (
        <div key={p.id} className="flex items-start gap-2 px-3 py-1.5 hover:bg-[var(--hover)]">
          {icons[p.severity]}
          <div className="flex-1 min-w-0">
            <span className="text-xs text-[var(--fg)]">{p.message}</span>
            <span className="text-2xs text-[var(--fg-faint)] font-mono ml-2">
              {p.file}:{p.line}:{p.col}
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}

// ── Main BottomPanel ──────────────────────────────────────────────────────────

export default function BottomPanel() {
  const {
    bottomTab, setBottomTab, logs, clearLogs,
    problems, bottomHeight,
  } = useStore()

  const [termKey, setTermKey] = useState(0)
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (bottomTab === 'output') endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs, bottomTab])

  const errCount  = problems.filter(p => p.severity === 'error').length
  const warnCount = problems.filter(p => p.severity === 'warning').length

  return (
    <div
      className="flex flex-col border-t border-[var(--border)] bg-[var(--surface-1)] flex-shrink-0"
      style={{ height: bottomHeight }}
    >
      <ResizeHandle />

      {/* Tab bar */}
      <div className="h-8 flex items-center px-2 gap-0.5 border-b border-[var(--border)] flex-shrink-0">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setBottomTab(t.id)}
            className={clsx(
              'px-3 py-1 rounded text-xs cursor-pointer border-0 bg-transparent transition-colors flex items-center gap-1.5',
              bottomTab === t.id
                ? 'text-[var(--fg)] bg-[var(--active)]'
                : 'text-[var(--fg-muted)] hover:text-[var(--fg)]',
            )}
          >
            {t.label}
            {t.id === 'problems' && (errCount + warnCount) > 0 && (
              <span className="flex items-center gap-1 text-2xs font-mono">
                {errCount  > 0 && <span className="text-red-400">{errCount}</span>}
                {warnCount > 0 && <span className="text-yellow-400">{warnCount}</span>}
              </span>
            )}
          </button>
        ))}
        <div className="flex-1" />
        {bottomTab === 'terminal' && (
          <IconBtn tooltip="Clear terminal" onClick={() => setTermKey(k => k + 1)}>
            <Trash2 size={11} />
          </IconBtn>
        )}
        {bottomTab === 'output' && (
          <IconBtn tooltip="Clear output" onClick={clearLogs}>
            <Trash2 size={11} />
          </IconBtn>
        )}
      </div>

      {/* Content */}
      {bottomTab === 'output' && (
        <div className="flex-1 overflow-y-auto px-3 py-2">
          {logs.length === 0 && (
            <span className="text-xs text-[var(--fg-faint)]">No output yet.</span>
          )}
          {logs.map(l => (
            <div key={l.id} className="flex gap-3 font-mono text-xs leading-[18px]">
              <span className="text-[var(--fg-faint)] flex-shrink-0 select-none">{l.time}</span>
              <span className={clsx({
                'text-green-400':          l.type === 'ok',
                'text-red-400':            l.type === 'err',
                'text-yellow-400':         l.type === 'warn',
                'text-[var(--fg-muted)]':  l.type === 'info',
              })}>
                {l.msg}
              </span>
            </div>
          ))}
          <div ref={endRef} />
        </div>
      )}

      {bottomTab === 'problems' && <ProblemsTab />}

      {/* Terminal — re-montado con key cuando se limpia */}
      {bottomTab === 'terminal' && <Terminal key={termKey} />}
    </div>
  )
}