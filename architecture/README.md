# Blog architecture

This folder contains PlantUML diagrams describing the structure and flow of the go-htmx-blog application.

## Diagrams

| File | Description |
|------|-------------|
| [components.puml](components.puml) | High-level packages and dependencies (handlers, services, content abstraction) |
| [content-abstraction.puml](content-abstraction.puml) | Content interfaces and implementations (Source, BlockRenderer, PageRenderer) |
| [sequence-single-post.puml](sequence-single-post.puml) | Request flow for rendering a single post (GET /notion/content/...) |

## Viewing / generating images

- **VS Code**: Install the "PlantUML" extension, then open a `.puml` file and use "Preview Current Diagram" (Alt+D).
- **CLI**: Install [PlantUML](https://plantuml.com/) (e.g. `brew install plantuml`), then run:
  ```bash
  plantuml architecture/*.puml
  ```
  This generates `.png` (or `.svg`) files next to each diagram.
- **Online**: Copy the contents of a `.puml` file into [plantuml.com/plantuml](https://www.plantuml.com/plantuml/uml/).
