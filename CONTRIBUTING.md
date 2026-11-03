# Contributing to DeckDrop

## Commits

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(optional scope): <description>
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `build`.

Examples:

- `feat: add chunked upload endpoint`
- `fix(backend): reject negative offsets`
- `docs: update build instructions`

Keep commits small and focused on a single change.

## Development

- Use the latest stable Go release.
- Format code with `gofmt` (`make fmt`); CI rejects unformatted code.
- Write a failing test first where it makes sense, then the code to pass it.
- Before opening a pull request run `make build test lint`.
