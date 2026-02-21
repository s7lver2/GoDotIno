// ─────────────────────────────────────────────────────────────────────────────
//  godotino_core  —  public library API
//
//  External crates (CLI in Go via FFI, IDE plugin, WASM build) should
//  consume this crate through the types and functions exported here.
// ─────────────────────────────────────────────────────────────────────────────

pub mod error;
pub mod lexer;
pub mod parser;
pub mod runtime;
pub mod transpiler;

pub use error::{GodotinoError, Result, Span};
pub use transpiler::TranspileConfig;
pub use runtime::Board;

// ── Pipeline ──────────────────────────────────────────────────────────────────

/// One-shot: Go source text → Arduino C++ source text.
///
/// ```rust
/// use godotino_core::{Pipeline, TranspileConfig};
///
/// let src = r#"
///     package main
///     import "arduino"
///     func setup() { arduino.pinMode(13, arduino.OUTPUT) }
///     func loop()  { arduino.digitalWrite(13, arduino.HIGH); arduino.delay(500) }
/// "#;
///
/// let cpp = Pipeline::new(TranspileConfig::default()).run(src, "main.go").unwrap();
/// assert!(cpp.contains("pinMode(13, OUTPUT)"));
/// ```
pub struct Pipeline {
    cfg: TranspileConfig,
}

impl Pipeline {
    pub fn new(cfg: TranspileConfig) -> Self { Self { cfg } }

    pub fn run(&self, source: &str, filename: &str) -> Result<String> {
        // 1. Lex
        let tokens = lexer::Lexer::new(source, filename).tokenize()?;

        // 2. Parse
        let prog = parser::Parser::new(tokens).parse_program()?;

        // 3. Generate
        let mut gen = transpiler::Transpiler::new(self.cfg.clone());
        gen.generate(&prog)
    }
}

// ── Diagnostics helper ────────────────────────────────────────────────────────

/// Format a pipeline error with full source context (suitable for CLI output).
pub fn pretty_error(err: &GodotinoError, source: &str) -> String {
    err.pretty(source)
}