# mutcheck

`mutcheck` is a small, custom mutation-testing engine for trusted local Go projects. It makes one AST-level change at a time, runs `go test`, and reports how effectively the existing tests detect those changes.

This is a senior-project MVP: it deliberately favors reliable, explainable behavior over broad language coverage or parallel execution.

## Supported mutations

- `>` / `>=`, `<` / `<=`, and `==` / `!=`
- `+` / `-`
- `&&` / `||`
- `true` / `false`

## Install and run

From this repository during development:

```powershell
go run ./cmd/mutcheck run --source internal\pricing\discount.go --packages ./internal/pricing --report mutation-report.json
```

Once released, install a versioned copy and use it from any Go module:

```powershell
go install github.com/aiden/mutcheck/cmd/mutcheck@v1
mutcheck run --project C:\code\orders-service --source internal\pricing\discount.go --packages ./internal/pricing --timeout 20s --report mutation-report.json
```

`--source` must be a `.go` file inside the selected project. `--packages` is passed as one Go package pattern; it defaults to `./...`.

## Results

Each candidate has one isolated temporary project copy. The original project is not edited. The tool checks that the unmodified baseline test suite passes before testing mutants.

- **Killed**: a test failed after the mutation.
- **Survived**: all selected tests passed after the mutation.
- **Timeout**: the test process exceeded its configured timeout.
- **Invalid**: the mutated project could not compile.
- **Error**: the test command failed before reporting a test failure.

The score is `killed / (killed + survived) * 100`. Timeouts, invalid mutants, and errors are reported but excluded from the score.

The JSON report stores only metadata and outcome categories; it intentionally does not store raw test output.

## Safety

`mutcheck` runs test code, which can execute arbitrary commands. Use it only on trusted repositories. Temporary workspaces keep source changes away from the original tree, but they are not a security sandbox.

The workspace copier excludes `.git`, `.env`, `.env.*`, `.pem`, and `.key` files. Do not use the initial GitHub Actions integration for tests that need credentials. The process never uses a shell, and every `go test` run has a timeout.

## GitHub Actions

After publishing this repository at `github.com/aiden/mutcheck` and tagging a `v1` release, another repository can call the reusable workflow:

```yaml
name: Mutation testing

on: workflow_dispatch

permissions:
  contents: read

jobs:
  mutation-test:
    uses: aiden/mutcheck/.github/workflows/mutation-test.yml@v1
    with:
      source: internal/pricing/discount.go
      packages: ./internal/pricing
      timeout: 30s
```

The workflow checks out the caller repository, installs the versioned CLI, and uploads a sanitized JSON report. It uses read-only repository permissions and accepts no secrets.
