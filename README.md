# gitai

Uses Claude to analyze your uncommitted git changes and split them into logical, conventional commits.

## Requirements

- Go
- `claude` CLI installed and authenticated

## Install

```sh
go install github.com/finn/gitai@latest
```

Or build from source:

```sh
go build -o gitai .
```

## Usage

Inside any git repo with uncommitted changes, run:

```sh
gitai
```

It will:
1. Analyze your diff with Claude
2. Show a review screen with suggested commits grouped by logical change
3. Let you confirm, edit, or deselect commits before anything is applied
4. Stage and commit the selected groups

### Controls

| Key | Action |
|-----|--------|
| `enter` | Apply selected commits |
| `q` / `ctrl+c` | Quit without committing |
