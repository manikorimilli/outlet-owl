# ADR-0004: Use GitHub Actions for CI, with no live model calls

- Status: Accepted
- Date: 2026-10-06
- Task: none (stack selection before the first task)
- Deciders: the product owner, in session
- Area: ci and delivery
- Reversibility: cheap: the checks are make targets; only the workflow file is GitHub-specific

## Context

- The repository is hosted on GitHub (`origin` is github.com/manikorimilli/outlet-owl).
- Tests replay recorded model responses and CI makes no live model calls (REQ-034, REQ-035), so CI never holds the OpenRouter key.
- The server is Go with sqlc and goose (ADR-0001, ADR-0003) and the UI is React with Vite (ADR-0002): CI needs both toolchains and a PostgreSQL instance for integration and migration tests.
- The PRD's "nothing else" constraint (section 6) limits the runtime, not CI.
- The product owner pushes, merges and deploys; CI only checks.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| GitHub Actions (chosen) | YAML workflows; minutes count against the GitHub plan | A repository already on GitHub |
| GitLab CI | The repository would have to move or be mirrored | A repository on GitLab |
| Local only (`make check`) | REQ-035 names a CI pipeline | A throwaway prototype |
| Buildkite or Jenkins | A server or agent to run and maintain | Large estates with special build hardware |

## Decision

We will run CI on GitHub Actions, with PostgreSQL as a service container and no OpenRouter key, because it is the CI of the git host already in use and needs no new account. The product owner chose the recommended option.

## Consequences

- CI runs Go and Node jobs, plus PostgreSQL as a service container for migration and integration tests.
- No OpenRouter key is stored in GitHub secrets; a test that finds no recording fails instead of calling the model (US-02-003).
- Checks are exposed as make targets so they run the same way locally and in CI.
- Revisit if the repository moves away from GitHub, or if CI minutes on the plan run out.

## Commits us to

GitHub Actions, PostgreSQL service container
