# Raga: C++ Problem Generator and Tester

Raga is a command-line tool written in Go for generating and testing C++ competitive programming problem files. It streamlines creating problem files from templates and automates testing against predefined inputs and outputs.

## Features

- **Generate** problem files (`problem.cpp`, `problem-in.txt`, `problem-out.txt`) from templates
- **Test** C++ solutions automatically against expected I/O
- **Watch** mode — re-runs tests every time you save a file
- **Multiple test cases** — run all `problem-*.in` / `problem-*.out` pairs
- **Stress testing** — brute-force vs optimized with random generated inputs
- **Clean** — remove build artifacts (`.temp/`, `.stress-failures/`, binaries)
- **Config** — persistent defaults via `~/.raga.json`
- **Colored** output with spinner animation during compilation
- Configurable C++ standard and timeout

## Quick Start

```bash
# Generate a new problem from the default template
raga -new -name=TwoSum

# Edit TwoSum.cpp, fill TwoSum-in.txt and TwoSum-out.txt, then test
raga -test -name=TwoSum

# Watch — auto-test on every save
raga -test -name=TwoSum -watch
```

## Installation

```bash
go mod tidy
go build -o raga .
export PATH="$PATH:$(pwd)"
raga -version
```

## Usage

### Generate

```bash
raga -new -name=ProblemName [-template=default]
# Creates: ProblemName.cpp, ProblemName-in.txt, ProblemName-out.txt
```

### Test

```bash
# Single test
raga -test -name=ProblemName

# With custom standard and timeout
raga -test -name=ProblemName -std=c++17 -timeout=10

# Watch for changes (auto re-test on save)
raga -test -name=ProblemName -watch

# Run all numbered test cases (Problem-1.in -> Problem-1.out, etc.)
raga -test -name=Problem -cases
```

### Stress Test

Run an optimized solution against a brute-force with random generated inputs:

```bash
# You need three files: Brute.cpp, Fast.cpp, Generator.cpp
raga -stress -brute=Brute -optimized=Fast -gen=Generator -iterations=5000
```

On mismatch, raga saves the failing input/output to `.stress-failures/`.

### Utility

```bash
raga -list             # List available templates
raga -clean            # Remove .temp/, .stress-failures/, binaries
raga -config           # Show current config (~/.raga.json)
raga -version          # Show version
raga -help             # Full usage
```

## Configuration

Raga reads `~/.raga.json` on startup. Example:

```json
{
  "std": "c++20",
  "timeout": 30,
  "template": "default",
  "iterations": 5000
}
```

Flags override config values at runtime.

## Templates

| Template      | Description                                    |
|---------------|------------------------------------------------|
| `default`     | Full template with mint, debug macros, fast I/O |
| `simple`      | Minimal boilerplate with fast I/O              |
| `generator`   | Random test case generator for stress testing  |

Add your own `.cpp` files to `templates/`. Raga looks in the current directory first, then next to the executable.

## Project Structure

```
├── main.go              # CLI entry point
├── go.mod / go.sum      # Go module
├── .raga.json           # Default config (optional)
├── .gitignore
├── README.md
└── templates/
    ├── default.cpp      # Full-featured template
    ├── simple.cpp       # Minimal template
    └── generator.cpp    # Stress test generator template
```
