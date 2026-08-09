# Tools Collection

A collection of useful command-line tools and utilities for daily development and system administration tasks.

## Private infrastructure documentation

Operational documentation for hosts, network topology, tunnels, and recovery
procedures is maintained separately in the private
[`r4wm/infra-docs`](https://github.com/r4wm/infra-docs) repository. Do not add
infrastructure details or credentials to this repository.

## 📁 Repository Structure

```
tools/
├── bin/              # Executable binaries and shell scripts
├── src/              # Go source code
├── Makefile          # Build automation
└── README.md         # This file
```

## 🛠️ Tools Overview

### Bible Search Tools

#### `bs` (Bash Version)
Original bash script for searching Bible verses by text content using the PRSMUSA Bible API.

**Usage:**
```bash
./bin/bs "love one another"
./bin/bs "faith hope love"
./bin/bs "blessed are the meek"
```

**Features:**
- Searches for verses containing specific text/phrases
- URL-encodes search queries
- Makes HTTP requests to Bible search API
- Pretty-prints JSON responses using `jq`

#### `bsg` (Go Version)
Modern Go port of the Bible search tool with enhanced features for searching verse content.

**Usage:**
```bash
# Basic text search
./bin/bsg "love one another"
./bin/bsg "peace be with you"

# Verbose mode with debug logging
./bin/bsg -v "faith hope love"

# Show help
./bin/bsg -h
```

**Features:**
- ✅ Verbose logging with `-v` flag
- ✅ Built-in help with `-h` flag
- ✅ Proper error handling
- ✅ No external dependencies (unlike bash version)
- ✅ Cross-platform compatibility
- ✅ Pretty-printed JSON output

### System Utilities

#### `rm_swap.sh`
Refreshes system swap space by turning it off and back on.

**Usage:**
```bash
./bin/rm_swap.sh
```

**What it does:**
- Disables all swap partitions/files
- Re-enables all swap partitions/files
- Useful for clearing swap cache

#### `rm_tilde.sh`
Recursively removes all backup files (`*~`) from the current directory.

**Usage:**
```bash
cd /path/to/cleanup
~/bin/rm_tilde.sh
```

**Features:**
- Shows current directory being cleaned
- Fun ASCII art output
- Recursive cleanup

#### `xset_noblank.sh`
Disables screen blanking and power management for X11 sessions.

**Usage:**
```bash
./bin/xset_noblank.sh
```

**What it does:**
- Turns off screen saver (`xset s off`)
- Disables DPMS power management (`xset -dpms`)
- Prevents screen blanking (`xset s noblank`)

### Other Tools

#### `emojis`
*(Binary executable - functionality to be documented)*

#### `gac`
*(Binary executable - functionality to be documented)*

## 🚀 Getting Started

### Prerequisites

- **Go 1.19+** (for building Go tools)
- **bash** (for shell scripts)
- **Standard Unix tools**: `curl`, `jq` (for bash Bible search)

### Building from Source

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd tools
   ```

2. **Build Go binaries:**
   ```bash
   make
   ```

3. **Install to personal bin directory:**
   ```bash
   make install
   ```

### Manual Installation

Copy individual scripts to your `~/bin` directory:
```bash
cp bin/* ~/bin/
chmod +x ~/bin/*
```

### Setting Up PATH

To use the tools from anywhere without specifying full paths, add `~/bin` to your PATH:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Or for immediate use in current session:
export PATH="$HOME/bin:$PATH"
```

**After setting up PATH, you can use tools directly:**
```bash
# Instead of ./bin/bsg "search text"
bsg "search text"

# Instead of ~/bin/rm_tilde.sh
rm_tilde.sh
```

## 📋 Makefile Targets

| Target | Description |
|--------|-------------|
| `make` or `make all` | Build the Go binary (`bsg`) |
| `make clean` | Remove built binaries |
| `make install` | Install all binaries to `~/bin/` |
| `make deps` | Download Go module dependencies |
| `make test` | Run Go tests |
| `make fmt` | Format Go source code |
| `make vet` | Run Go static analysis |
| `make help` | Show available targets |

## 🔧 Development

### Project Structure

- **`src/`** - Go source files
- **`bin/`** - Compiled binaries and shell scripts
- **`Makefile`** - Build configuration

### Adding New Tools

1. **For Go tools:**
   ```bash
   # Create source file
   vim src/newtool.go
   
   # Update Makefile if needed
   # Build
   make
   ```

2. **For shell scripts:**
   ```bash
   # Create script
   vim bin/newtool.sh
   chmod +x bin/newtool.sh
   ```

### Code Style

- Go code follows standard `gofmt` formatting
- Shell scripts use bash best practices
- All executables should include help/usage information

## 📚 Dependencies

### Runtime Dependencies
- **Bible Search (bash)**: `curl`, `jq`, `python3`
- **Bible Search (Go)**: None (statically linked)
- **System utilities**: Standard Unix tools

### Build Dependencies
- **Go toolchain** (1.19+)
- **Make**

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📝 License

See [LICENSE](LICENSE) file for details.

## 🔗 API References

- **PRSMUSA Bible API**: `https://web.prsmusa.com/bible/search`
  - Parameters: `q` (query), `json=true`
  - Returns: JSON formatted Bible search results

## 💡 Tips

- **Add `~/bin` to your PATH** for easy access to installed tools from anywhere
- Use `bsg -v` for debugging Bible search queries  
- Run `make install` after updates to sync latest versions
- Check individual tool help with `-h` flag where available
- **Verify PATH setup**: Run `echo $PATH` to confirm `~/bin` is included

---

*Last updated: June 2025*
