# Prompts

The prompts the model gateway sends, one directory per prompt (Q-011,
US-02-002). The Go package in this directory parses them; the gateway sends
`Text` as the system message with the version's `max_tokens` (at most 1000)
and `temperature`.

```
prompts/<name>/v1.md     ---
                         max_tokens: 1000
                         temperature: 0      (optional; unset = provider default)
                         ---
                         The prompt text.
prompts/<name>/current   1
```

Rules:

- Never edit or delete a version once it has been used: stored tags and
  drafts, and every cost row, name it. Any change, a word, the tone, a
  setting, the urgent reasons or the theme list, is a new version.
- Numbers start at 1 with no gap. `current` names the version in use.
- A theme-list change in configuration ships with a new tagging version in
  the same merge request (GenAI design section 4.3).
- A new version makes the recordings for that prompt stale: record again,
  deliberately (`MODEL_GATEWAY_MODE=record`).

`tagging/` arrives with build phase 3 and `reply/` with phase 5; each adds the
`//go:embed` that feeds `Parse`.
