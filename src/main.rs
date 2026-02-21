// ─────────────────────────────────────────────────────────────────────────────
//  godotino  —  standalone binary
//  Usage: godotino <input.go> [output.cpp] [--board <id>] [--source-map]
//
//  This is intentionally minimal — the full CLI is a separate Go tool
//  (see docs/CLI.md) that calls into this binary or links godotino_core.
// ─────────────────────────────────────────────────────────────────────────────

use std::path::PathBuf;
use godotino_core::{Pipeline, TranspileConfig, Board};

fn main() {
    let args: Vec<String> = std::env::args().collect();

    // Quick --version / --help
    if args.iter().any(|a| a == "--version" || a == "-V") {
        println!("godotino {}", env!("CARGO_PKG_VERSION"));
        return;
    }
    if args.iter().any(|a| a == "--help" || a == "-h") || args.len() < 2 {
        print_help();
        return;
    }
    if args.iter().any(|a| a == "boards") {
        print_boards();
        return;
    }

    // Parse simple flags
    let input: PathBuf  = args[1].clone().into();
    let output: Option<PathBuf> = args.get(2).filter(|s| !s.starts_with('-'))
        .map(|s| s.clone().into());

    let board = flag_value(&args, "--board").unwrap_or_else(|| "uno".into());
    let source_map = args.iter().any(|a| a == "--source-map");

    let cfg = TranspileConfig {
        board,
        emit_source_map: source_map,
        ..Default::default()
    };

    // Read source
    let source = match std::fs::read_to_string(&input) {
        Ok(s) => s,
        Err(e) => { eprintln!("error: cannot read {}: {}", input.display(), e); std::process::exit(1); }
    };

    // Run pipeline
    let filename = input.to_string_lossy().into_owned();
    let result   = Pipeline::new(cfg).run(&source, &filename);

    match result {
        Ok(cpp) => {
            match output {
                Some(path) => {
                    if let Err(e) = std::fs::write(&path, &cpp) {
                        eprintln!("error: cannot write {}: {}", path.display(), e);
                        std::process::exit(1);
                    }
                    eprintln!("ok  {}", path.display());
                }
                None => print!("{}", cpp),
            }
        }
        Err(e) => {
            eprintln!("{}", godotino_core::pretty_error(&e, &source));
            std::process::exit(1);
        }
    }
}

fn flag_value(args: &[String], flag: &str) -> Option<String> {
    args.windows(2).find(|w| w[0] == flag).map(|w| w[1].clone())
}

fn print_help() {
    println!(
r#"godotino {} — Go-to-Arduino C++ transpiler

USAGE:
    godotino <input.go> [output.cpp] [FLAGS]

FLAGS:
    --board <id>     Target board (default: uno)
    --source-map     Emit #line pragmas for source mapping
    --version        Print version
    --help           Print help

COMMANDS:
    godotino boards   List supported boards

EXAMPLES:
    godotino src/main.go build/main.cpp --board esp32
    godotino src/main.go                           # print to stdout
"#,
    env!("CARGO_PKG_VERSION"));
}

fn print_boards() {
    println!("{:<15} {:<30} {:<8} {:<6}  {}", "ID", "NAME", "FLASH", "RAM", "FQBN");
    println!("{}", "-".repeat(85));
    for b in Board::catalog() {
        println!("{:<15} {:<30} {:>5}K  {:>4}K  {}",
            b.id, b.name, b.flash_kb, b.ram_kb, b.fqbn);
    }
}