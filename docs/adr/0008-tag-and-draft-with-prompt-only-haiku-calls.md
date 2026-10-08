# ADR-0008: Tag reviews and draft replies with prompt-only calls to Claude Haiku 4.5

- Status: Accepted
- Date: 2026-10-06
- Task: none (design before the first task)
- Deciders: the product owner, 2026-10-08
- Area: llm approach
- Reversibility: cheap: the prompts are versioned files and the line format is parsed in one place; a new approach is a new prompt version and parser

## Context

- The brief requires LLM classification and reply generation through one gateway on OpenRouter, with Claude Haiku 4.5 and no fallback model (PRD section 6, Q-016).
- Tagging sends reviews in batches of 20 (REQ-006), matches results by review id (REQ-012), validates each result and retries only the missing ids (REQ-013, REQ-014), all under max_tokens 1000 (REQ-032).
- Every call counts toward a USD 8 application stop under a USD 10 key cap (REQ-031, Q-015).
- The reasoning, numbers and risks are in docs/genai/review-classification-and-replies-solution.md.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| Prompt only, compact line output validated in code (proposed) | Code owns the parser and the validator; the line format is a contract to keep stable | Labels and short drafts from the input alone |
| Keyword rules with no model | Misses urgent complaints written in Hinglish or with no keyword; the brief requires LLM classification | A fixed vocabulary in one language |
| JSON structured output | About three times the output tokens at 20 results, close to the 1,000 cap; OpenRouter support for it on this model is unverified | Fewer results per call, or no output cap |
| Retrieval, tools or an agent | Nothing to retrieve or act on; every answer comes from the review alone | Answers that need documents or live systems |
| Message Batches at half price | Not offered through OpenRouter (assumption); no second service is allowed | Direct Anthropic API use with tolerance for an hour's delay |

## Decision

Accepted 2026-10-08. Temperature 0 lowers variation but does not make answers repeatable; recordings do (phase 3 LLD).

- **Tagging:** one prompt-only call per batch of up to 20 reviews to Claude Haiku 4.5 through OpenRouter, with thinking off and temperature 0. The answer is one line per review: id, theme codes, sentiment, urgent reasons. Code validates each line by review id; missing or invalid ids are retried at most twice.
- **Drafting:** one prompt-only call per review a manager opens, carrying the brand tone and the language rule.
- **Prompts:** both are versioned files in the repository.
- **Caching:** stored results and recorded responses; no prompt caching, because the prompts are below Haiku 4.5's 4,096-token cache minimum.

## Consequences

- The tagging line format and its validator are a contract: changing them is a new tagging prompt version and new recordings.
- Urgent reasons are stored with each tag result, which needs a data model and OpenAPI revision before build phase 3.
- The evaluation needs 100 real reviews labelled by the product owner before the first live tagging prompt run.
- Revisit if the urgent-recall gate fails on the golden set after prompt changes, if batches of 20 are cut off at the cap more than rarely, or if a second model is ever allowed.

## Commits us to

OpenRouter chat completions, the model identifier for Claude Haiku 4.5 on OpenRouter (verified before the first live call), a line-format parser in the Go server
