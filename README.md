# GoDotIno
Write in Go, Upload In C++

```

   ██████╗  ██████╗ ██████╗  ██████╗ ████████╗██╗███╗   ██╗ ██████╗
  ██╔════╝ ██╔═══██╗██╔══██╗██╔═══██╗╚══██╔══╝██║████╗  ██║██╔═══██╗
  ██║  ███╗██║   ██║██║  ██║██║   ██║   ██║   ██║██╔██╗ ██║██║   ██║
  ██║   ██║██║   ██║██║  ██║██║   ██║   ██║   ██║██║╚██╗██║██║   ██║
  ╚██████╔╝╚██████╔╝██████╔╝╚██████╔╝   ██║   ██║██║ ╚████║╚██████╔╝
   ╚═════╝  ╚═════╝ ╚═════╝  ╚═════╝    ╚═╝   ╚═╝╚═╝  ╚═══╝ ╚═════╝

godotino lets you write Arduino firmware in Go and transpiles it to C++.

Run 'godotino <command> --help' for details on each command.

Usage:
  godotino [command]

Available Commands:
  boards      List and detect supported boards
  build       Transpile Go sources to C++ (and optionally compile)
  check       Validate source files for errors and warnings (no output produced)
  clean       Remove the build/ directory
  completion  Generate the autocompletion script for the specified shell
  config      Get or set CLI configuration
  help        Help about any command
  init        Initialize a new godotino project
  upload      Upload compiled firmware to a connected board
  version     Print version information

Flags:
  -h, --help       help for godotino
      --no-color   disable colored output
  -v, --verbose    verbose output

Use "godotino [command] --help" for more information about a command.
```


GoDotIno is a framework designed for adapt your go code into c++ code, and upload it into your favourite arduino devices

## Get Started

### Linux (Recommended)
```bash
# Build core + cli
make all

# Install cli tool
sudo make install
```

## Supported Go subset

| Feature | Status |
|---------|--------|
| Variables (`var`, `:=`) | ✅ |
| Constants (`const`) | ✅ |
| Functions + methods | ✅ |
| Structs + type aliases | ✅ |
| `if / else` | ✅ |
| `for` (C-style, while-style) | ✅ |
| `for … range` over arrays | ✅ |
| `switch / case` | ✅ |
| All operators | ✅ |
| String literals | ✅ |
| `import` + package calls | ✅ |
| Goroutines (`go`) | ⚠️ stub (comment emitted) |
| `defer` | ⚠️ stub (comment emitted) |
| Channels (`chan`) | ❌ not supported |
| Interfaces | ⚠️ type-only |
| Closures / lambdas | ⚠️ skeleton only |
| Multiple return values | ⚠️ struct pack |
| Generics | ❌ not planned |
| `map` type | ⚠️ void* stub |
| Garbage collection | ❌ (Arduino has no heap GC) |

## Mapped packages

| Go import | Maps to |
|-----------|---------|
| `"arduino"` | Arduino.h builtins |
| `"fmt"` | `Serial.print/println` |
| `"time"` | `delay / millis` |
| `"math"` | `<math.h>` functions |
| `"strconv"` | `String::to…` methods |
| `"wire"` / `"Wire"` | Wire.h (I2C) |
| `"spi"` / `"SPI"` | SPI.h |
| `"serial"` / `"Serial"` | Serial object |
| `"Servo"` | Servo.h |
| `"LiquidCrystal"` | LiquidCrystal.h |

## Supported boards

Run `goduino boards` for a full list:

| ID | Name | CPU |
|----|------|-----|
| `uno` | Arduino Uno | ATmega328P |
| `nano` | Arduino Nano | ATmega328P |
| `mega` | Arduino Mega 2560 | ATmega2560 |
| `esp32` | ESP32 Dev Module | Xtensa LX6 |
| `esp8266` | ESP8266 NodeMCU | ESP8266 |
| `pico` | Raspberry Pi Pico | RP2040 |
| `due` | Arduino Due | AT91SAM3X8E |
| … | … | … |