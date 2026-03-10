# 🌐 DarkScraper - Dark Web Search Engine

**DarkScraper** (also known as **Darker**) is a high-performance, terminal-based search engine designed for the dark web. It transforms the fragmented ecosystem of `.onion` hidden services into a structured, searchable index via a premium "cyber-themed" terminal user interface (TUI).

![DarkScraper Preview](https://via.placeholder.com/800x400.png?text=DarkScraper+Cyber+TUI+Preview) *(Note: Replace with actual screenshot in production)*

## 🚀 Features

- **⚡ Full-Text Search**: Powered by SQLite FTS5 for lightning-fast keyword matching across indexed sites.
- **🛡️ Portable Tor integration**: Automatically detects your system Tor proxy. If missing, it downloads a portable **Tor Expert Bundle** and manages its lifecycle seamlessly.
- **🕷️ Resilience Crawler**: Automated discovery and indexing of hidden services with persistent storage and polite rate-limiting.
- **💎 Cyber Aesthetic**: A premium, high-contrast TUI built with `Bubble Tea` and `Lip Gloss`, featuring neon accents and status indicators.
- **📋 Easy Navigation**: Instantly copy `.onion` links to your clipboard with a single keystroke.

## 🏗️ Architecture

DarkScraper is built with a modular Go architecture:
- **`cmd/darker`**: The main entry point and lifecycle manager.
- **`internal/engine`**: The crawler and metadata extraction logic.
- **`internal/db`**: SQLite storage with FTS5 virtual tables and synchronization triggers.
- **`internal/ui`**: Interactive TUI components.
- **`pkg/tor`**: Cross-platform Tor process and proxy management.

## 🛠️ Installation

### Prerequisites
- [Go](https://golang.org/doc/install) 1.25 or higher.
- [Tor](https://www.torproject.org/download/) (Optional: DarkScraper can auto-download a portable version).

### Setup
```bash
# Clone the repository
git clone https://github.com/christian/darker.git
cd darker

# Build the application
go build -o darker ./cmd/darker/main.go
```

## 📖 Usage

Run the application:
```bash
./darker
```

### Keybindings
- **`Enter`**: Execute search query.
- **`Esc`**: Clear results and return to search bar.
- **`c`**: Copy the selected `.onion` URL to clipboard.
- **`q`** / **`Ctrl+C`**: Exit application safely.

## ⚖️ Ethical & Legal Disclaimer

DarkScraper is a tool designed for research and educational purposes. Accessing the dark web can expose you to illegal content or malicious services. 
- Use this tool responsibly.
- We do not condone or encourage illegal activities.
- Ensure you comply with your local laws and regulations.

## 📜 License
MIT License. See [LICENSE](LICENSE) for details.
