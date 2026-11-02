<p align="center">
  <img src="./web/public/pwa-512.png" alt="InkDock icon" width="76" height="76">
  <br>
  <strong>InkDock · 墨栈 AI</strong>
</p>

<p align="center">
  <strong>Write stories, build worlds, and use AI from your own creative workspace.</strong>
</p>

<p align="center">
  English | <a href="README.md">中文</a>
</p>

<p align="center">
  <a href="https://github.com/vevila2020/writing_platform_ai/releases"><img alt="Release" src="https://img.shields.io/github/v/release/vevila2020/writing_platform_ai?style=flat-square"></a>
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/github/license/vevila2020/writing_platform_ai?style=flat-square"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26.9%2B-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="Node.js" src="https://img.shields.io/badge/Node.js-22.13%2B-5FA04E?style=flat-square&logo=nodedotjs&logoColor=white">
</p>

<p align="center">
  Upstream version: <strong>v0.6.3</strong> (released on 2026-10-20) · Beta · <a href="https://github.com/vevila2020/writing_platform_ai/releases">No InkDock release published yet</a>
</p>

InkDock is a modified, self-hosted creative workspace based on [Denova](https://github.com/alfredxw/denova). It retains novel writing, interactive stories, structured lore, AI Agents, image creation, and project version management.

The project plans to connect to a self-hosted Sub2API service and provide user registration and model access, reducing reliance on third-party API sites. **Sub2API registration is not included in the current code.**

> The screenshots below show the inherited interface. Product branding and visual assets will be updated as development continues.

![InkDock writing interface preview](./img/ide.png)

<details>
<summary>View more screenshots</summary>

### Game

![Interactive game interface preview](./img/interactive.png)

### Lore Library

![Lore library interface preview](./img/setting.png)

### Creative Setups

![Creative setup interface preview](./img/story-teller.png)

### Change review

![Change review interface preview](./img/review.png)

### Workspace

![Workspace interface preview](./img/workspace.png)

Supports LAN access, responsive mobile layouts, and PWA apps.

<img src="./img/mobile.png" alt="Mobile interface preview" width="360">

</details>

## Why InkDock

InkDock brings novel writing, interactive stories, structured lore, AI Agents, image generation, automation, and local version management into one workspace. It is designed for creative projects whose settings and content need to evolve over time.

Start a new book from an idea, import an existing novel to continue writing, or use character cards and lore to build a branching text adventure. Agents can read project content, use tools, and edit files while important changes remain reviewable, undoable, and recoverable.

## Core Features

- **Novel writing**: Markdown document and source editing, multiple tabs, find and replace, outlines and chapter plans, progress tracking, inline comments, change review, and novel import.
- **Creative Agents**: choose Native, Codex, or Claude Code to work with selections, project files, images, and lore, with multiple conversations, SubAgent collaboration, and task pause/resume.
- **Interactive games**: advance stories through player input, with branches, storyline switching, action suggestions, character and world state, rule checks, story planning, and read-aloud. Stage scenes with Lore backgrounds and character sprites, with presentation saved along story branches.
- **Lore**: organize settings with custom categories, document-style indexes, @ references, and backlinks; find text mentions in project files and a selected story branch, provide grouped context to Writing and Game Agents, and manage multiple images, audio, and covers for each entry.
- **Creative Setups and Skills**: reuse narrative styles, image styles, rules, and other creative configurations; extend Agent workflows with Skills, including a built-in novel-writing Skill for premises, outlines, scenes, and revision.
- **Resource Center**: discover, preview, and selectively import Creative Setups, Skills, Lore, plugins, and games; combine resources for export, track sources, review updates, and restore backups.
- **Plugins and game extensions**: install and manage plugins and custom games. Plugins can provide Agent tools, direct commands, and independent panels; games can define their own interface, gameplay, and save format. The workbench supports creation, builds, isolated previews, and export.
- **Image creation**: generate book covers, chapter illustrations, game backgrounds, and character artwork, then preview results, associate them with Lore, and manage them in the app.
- **Versions and recovery**: save local versions, inspect changes, restore historical files, and review or undo Agent edits to the workspace.
- **Automation**: run scheduled review, continuation, and custom creative tasks.
- **Cross-platform experience**: Chinese and English UI, light and dark themes, Windows / macOS / Linux support, remote access, installation to a phone's home screen, and in-app updates from online releases or uploaded packages.

## Writing and Game

Writing and Game are peer top-level destinations in the workbench. Writing focuses on ideas, settings, outlines, chapters, and progress. Game focuses on player actions, story branches, character state, and storyline progression.

Lore, Creative Setups, Skills, and version management are shared between the two workflows. Chapter progress and game state remain separate so temporary state from one destination does not affect the other.

## Quick Start

### Install a Release

macOS / Linux users can run the installer:

```bash
curl -fsSL https://github.com/vevila2020/writing_platform_ai/releases/latest/download/install.sh | sh
```

This project has not published its first release yet. Once it does, the command above and [InkDock Releases](https://github.com/vevila2020/writing_platform_ai/releases) will provide the installer and platform archives. Run `denova` after installation; on Windows, run `denova.exe`.

For stable use, choose a Release. The `master` branch may contain unreleased changes.

### First Run

1. Follow the startup guide to add a language-model API key and model name.
2. Create or import a book, or open an existing project directory.
3. Start creating from Writing, or create an interactive storyline from Game. Configure an image model only when you need image generation.

### Run from Source

You need Go 1.26.9+, Node.js 22.13+, pnpm, ripgrep, and Bash. On Windows, run these commands from Git Bash or WSL.

```bash
git clone https://github.com/vevila2020/writing_platform_ai.git
cd writing_platform_ai
corepack enable
./scripts/bootstrap.sh
```

Default addresses:

- Frontend: `http://localhost:7788`
- Backend: `http://localhost:7789`

## Models and Configuration

The recommended setup is through Settings: add a provider connection, select or enter a model, and test the connection. One connection can be reused by multiple models. Language models support built-in providers and custom API-compatible endpoints. Image models support OpenAI, xAI/Grok, Volcengine Seedream, Google Gemini Image, ComfyUI Workflow, and custom endpoints.

For Google Gemini text models, select “Gemini native API” in the connection's protocol options to use native tool calls, image inputs and reasoning continuation. Existing OpenAI-compatible connections keep their protocol.

Optional decision models can be configured in Settings for Game stage media, Lore semantic classification, and automation semantic triggers, with language-model fallback for uncertain results or service failures.

## Remote Access and Phone Usage

Enable LAN access under **Settings → Access**, set a username and password, and restart. Other devices can sign in at the displayed address. The host can also create a one-use sign-in QR code and link that expire after 5 minutes. Scan with a phone on the same LAN, or copy the link to another device. The browser stays signed in for 30 days across refreshes and server restarts; sign out from Settings. Phone browsers can add InkDock to the home screen.

When running from source, QR codes and connection links use the backend port to serve the `web/dist` build. Run `pnpm --dir web build` and restart the backend before first use or after frontend changes. The local Vite entry point remains available for development with hot reload.

For public or domain-based deployments, put InkDock behind an HTTPS reverse proxy such as Caddy or Nginx so login credentials are not transmitted in cleartext.

## Usage statistics

Release builds explain usage statistics on first use. Collection is enabled by default and starts only after the notice is acknowledged. Turn it off anytime under **Settings → General**. PostHog receives fixed activity, feature usage, retention, version, platform, execution outcome, duration in milliseconds, error category, HTTP status, and theme/language changes, without creative text, prompts, AI replies, images, project names, file paths, keys or raw error text. A random local ID links return visits; the cloud receives the connection IP. Source and development builds do not send statistics. See the [metric and field definitions](./docs/analytics.md).

## Development

Read [CONTRIBUTING.md](./CONTRIBUTING.md) before contributing. Common commands:

```bash
./scripts/bootstrap.sh fe
./scripts/bootstrap.sh be
./scripts/build.sh
```

## Community

InkDock is maintained by Elias. Public feedback and community links will be added when available.

## License

This project is a modified fork of [Denova](https://github.com/alfredxw/denova) and is distributed under Apache-2.0. The upstream license and copyright notices remain in [LICENSE](./LICENSE) and the relevant source files. Elias rewrote this README for InkDock; this project is not endorsed by the upstream author.
