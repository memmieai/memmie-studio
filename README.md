# ReYNa Studio (RYN)

**ReYNa Studio** - A universal productivity platform that transforms how users interact with AI through dynamic, content-aware interfaces.

> **RYN** (ReYNa) is your AI-powered creative workspace where ideas evolve into polished content.

## 🎯 Vision

Transform personal productivity by treating all content as interconnected "blobs" in your personal knowledge graph, with AI providers that adapt their interfaces to your workflow - whether you're writing books, composing music, researching, or coding.

## 🚀 Quick Start

```bash
# Clone and setup
git clone https://github.com/memmieai/memmie-studio.git
cd memmie-studio

# Start all services
cd ../memmie-infra && ./dev-hot-reload.sh

# Access the app
open http://localhost:8010
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Clients (Web, Mobile, AR, CLI)              │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│           Studio Service (8010) - API Gateway            │
│     • Serves React app  • WebSocket  • Routes APIs       │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│State Service │   │Provider Svc  │   │Workflow Svc  │
│    (8006)    │   │   (8007)     │   │   (8005)     │
│              │   │              │   │              │
│ User Blobs   │   │  Providers   │   │  Pipelines   │
│ DAG Storage  │   │  Templates   │   │  Execution   │
└──────────────┘   └──────────────┘   └──────────────┘
```

## 🔑 Key Concepts

### Blobs - Universal Content Units
Every piece of content is a "blob" in your personal knowledge graph:
- Text, code, audio, images - all stored as blobs
- Form a DAG (Directed Acyclic Graph) showing relationships
- Version controlled through deltas
- Per-user isolation for privacy

### Providers - AI Transformers
Providers process and transform your blobs:
- **Text Expander**: Turn bullet points into prose
- **Music Generator**: Convert descriptions to Strudel code
- **Research Assistant**: Organize and link knowledge
- **Code Documenter**: Auto-generate documentation

### Dynamic UI - Adaptive Interfaces
The UI adapts to your content and workflow:
- Providers define their optimal layout
- Same content renders differently based on context
- Works across web, mobile, and AR platforms
- Real-time updates via WebSocket

## 🎨 Use Cases

### 📚 Book Writing
```yaml
Input: "Chapter 1: Hero meets mentor"
Provider: book-writer
Output: Expanded chapter with dialogue, descriptions
UI: Split view - draft left, expanded right
```

### 🎵 Music Creation
```yaml
Input: "Upbeat electronic with bass drops"
Provider: music-generator  
Output: Strudel code generating the music
UI: Code editor + live audio playback
```

### 🔬 Research Assistant
```yaml
Input: Research papers, notes, highlights
Provider: research-assistant
Output: Knowledge graph with connections
UI: Graph visualization + document viewer
```

### 🎙️ Voice Capture ("Ramble")
```yaml
Input: Speech via microphone
Process: Audio → Whisper → Text → Blob
Context: Target specific projects/providers
Platform: All devices with mic access
```

## 🛠️ Development

### Service Ports
- `8010` - Studio Service (API Gateway + React)
- `8006` - State Service (Blob Storage)
- `8007` - Provider Service (Registry)
- `8005` - Workflow Service (Execution)

### Tech Stack
- **Backend**: Go, MongoDB, PostgreSQL, NATS, Redis
- **Frontend**: React 18, TypeScript, Tailwind, WebSocket
- **Mobile**: React Native
- **AR**: SwiftUI + RealityKit (Vision Pro)
- **AI**: GPT-4, Whisper, Custom models

### Project Structure
```
memmie-studio/
├── cmd/server/          # Main server entry
├── internal/
│   ├── api/            # HTTP handlers
│   ├── blob/           # Blob management
│   ├── provider/       # Provider logic
│   ├── websocket/      # Real-time updates
│   └── workflows/      # YAML workflows
├── web/                # React frontend
├── mobile/             # React Native app
└── plans/              # Architecture docs
```

## 📖 Documentation

- [Master Plan](plans/00-MASTER-PLAN.md) - Complete architecture overview
- [State Service](plans/01-state-service-design.md) - Blob storage design
- [Provider Service](plans/02-provider-service-design.md) - Provider system
- [Dynamic UI](plans/04-dynamic-ui-system.md) - Adaptive interface system
- [API Reference](plans/03-studio-api-design.md) - Complete API docs

## 🚦 Roadmap

### Phase 1: Core (Weeks 1-2)
- [x] Architecture design
- [ ] State Service - blob storage
- [ ] Provider Service - registry
- [ ] Studio API Gateway

### Phase 2: Features (Weeks 3-4)
- [ ] Text expansion provider
- [ ] Speech input ("Ramble")
- [ ] React frontend
- [ ] WebSocket updates

### Phase 3: Advanced (Weeks 5-6)
- [ ] Music generator
- [ ] Research assistant
- [ ] Mobile app
- [ ] Performance optimization

### Phase 4: Platform (Weeks 7-8)
- [ ] Vision Pro AR app
- [ ] Developer SDK
- [ ] Provider marketplace
- [ ] Public API

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT - See [LICENSE](LICENSE) for details.

## 🔗 Links

- [Documentation](https://docs.memmie.ai/studio)
- [API Reference](https://api.memmie.ai/studio)
- [Provider Marketplace](https://providers.memmie.ai)
- [Discord Community](https://discord.gg/memmie)

---

Built with ❤️ by the Memmie team. Making AI work the way you think.