# ADR-0009: Allow one configured model in place of Claude Haiku 4.5

- Status: Accepted
- Date: 2026-10-08
- Task: none
- Deciders: the product owner, 2026-10-08
- Area: llm approach
- Reversibility: cheap: unset MODEL_ID and every call goes back to Claude Haiku 4.5

## Context

- ADR-0008 fixes Claude Haiku 4.5 as the one model, with no fallback (Q-016).
- Before paying for the first live run, the product owner wants to try the free OpenRouter model `apodex/apodex-1.1-mini:free`. OpenRouter lists it at USD 0 per token, with `temperature`, `max_tokens` and `reasoning` among its parameters; it is a reasoning model.

## Decision

`MODEL_ID` names the OpenRouter model for every call of a run; empty keeps Claude Haiku 4.5. There is still one model per run and no fallback. The model id is part of the request body, so recordings made for one model never answer for another, and each budget row records the model it called.

## Consequences

- The prompts, the line format, the parser, the retries and the budget stop are unchanged.
- Quality on another model is not the quality ADR-0008 was chosen for: `make eval` must pass on the model actually used before its results are trusted.
- Free models have daily request limits on OpenRouter and may answer slower; a run can stop part way, and the worker tags the rest on a later pass.
