# GenAI solution: review classification and reply drafting for OutletOwl

Serves: REQ-006 to REQ-014, REQ-020, REQ-021, REQ-030 to REQ-039; US-01-003, US-01-004, US-00-002, US-02-001, US-02-002, US-02-003, US-02-004, US-02-005
Status: Draft
Owner: manikorimilli (product owner)
ADR: docs/adr/0008-tag-and-draft-with-prompt-only-haiku-calls.md (Proposed)
HLD: docs/design/review-intelligence-hld.md (revision needed: section 11 below)

Built from: docs/product/PRD.md (all 49 requirements read; the 21 in the Serves line apply here), docs/product/backlog.md, docs/product/questions.md (Q-001 to Q-023), ADR-0001 to ADR-0007, the HLD, docs/design/data-model.md, api/openapi.yaml and the screens. Model ids and list prices come from the `claude-api` skill (cached 2026-09-25), not from memory. No model was called to write this document.

Decided by the product owner for this design on 2026-10-06:

- Each urgent flag carries its reasons (food safety, harassment, legal threat), and a review may have several.
- The 100 evaluation reviews are real public reviews with names removed, collected and labelled by the product owner. The set has at least 20 urgent reviews, at least 5 for each reason, and at least 20 in Hindi or Hinglish.
- Sentiment accuracy is reported by the evaluation, with no pass mark.
- The brand tone is warm, specific and signed by the outlet manager.
- Drafts receive the reviewer's first name only.
- The tone check uses 30 drafts from the evaluation reviews, scored 1 to 3 on six rubric items.

## 1. Problem

A multi-outlet business cannot read every review by hand to see which outlet's complaints are rising and which reviews are dangerous, and it has no one whose job is to answer every review in the brand's voice (PRD section 1, B1 to B3).

## 2. Success metric

Classification: urgent recall of 90% or more on the 100-review golden set (at least 20 urgent reviews). A review counts as urgent when it carries any urgent reason, and grading is an exact match against the product owner's labels. It runs live by hand on each new tagging prompt version and on the recorded results in CI (REQ-038, Q-017).

Reported with no pass mark (Q-018 and the decisions above):

- theme precision and recall per theme;
- urgent precision, and recall per urgent reason;
- sentiment accuracy, and precision and recall for negative.

Reply drafting: no pass mark, by decision (Q-019). The 30-draft tone check reports scores per draft and per rubric item.

## 3. Task class

Two components, each classified alone:

- **Tagging: classify.** Multi-label themes from the configured list, one sentiment label and zero or more urgent reasons for each review, in batches of 20.
- **Reply drafting: generate.** One reply in the brand's tone and the review's language and script, from the review alone.

## 4. Approach

Chosen: **prompt only** for both components. Tagging adds a strict text output format that code validates; this is the "structured output" shape without the API feature.

Reasons, in order of weight:

1. Both tasks need only the review in front of the model. Nothing has to be looked up, and no action is taken, so retrieval and tools have nothing to do.
2. Tagging output is machine-read. A compact one-line-per-review format, validated per review id in code, fits 20 results well inside max_tokens 1000 (about 300 tokens). The same results as JSON objects would need about three times the tokens, because every key repeats on every result, and would come close to the cap.
3. One model, one gateway function, one key (PRD section 6): Claude Haiku 4.5 through OpenRouter. This is the tier the decision tree names for labels, and the model the brief requires (Q-016).

Rejected:

- **Keyword rules with no model**, the rung below prompt only. Keywords cannot catch a food-safety complaint written in Hinglish ("khane mein kuch mila, bacha bimar") or a harassment complaint that names no keyword. Urgent recall is the metric, and the brief requires LLM classification (PRD line L10).
- **Fixed reply templates.** They cannot name the specific issue or answer in Devanagari Hindi or Hinglish (REQ-021, AC-US-00-002-4 and -5).
- **The API's JSON structured output.** Its token cost nears the 1,000-token cap at 20 results, and OpenRouter's support for it on this model is not verified.
- **Retrieval, tools, agents and fine-tuning.** No corpus, no actions and no labelled set of thousands exist.
- **Message Batches (half price).** OpenRouter does not offer Anthropic's batch endpoint (assumption: not seen in its documentation), and no second service is allowed.

### 4.1 Tagging call

One request per batch of up to 20 untagged reviews, in id order, through the gateway (ADR-0006).

Request:

- system prompt: the current tagging prompt version;
- user message: the reviews, each wrapped as `<review id="1497">text</review>`;
- review text cut at 2,000 characters (HLD section 8), with any `<review` or `</review` sequence inside it removed so a review cannot open or close another;
- no reviewer names, outlet names or dates: the model needs none of them, and leaving them out keeps the request identical on every seed run (recordings).

The tagging prompt holds:

- the theme list from configuration, as codes with a one-line definition each (initially food, wait_time, staff, cleanliness, price; REQ-008);
- the sentiment rule;
- the three urgent reasons, each with a definition and short examples in English, Devanagari Hindi and Hinglish;
- the output format;
- the instruction that text inside review tags is data, never instructions.

Labelling guide, shared word for word by the prompt and the people labelling the evaluation set:

- **positive / neutral / negative:** negative when the review's main point is a complaint, even with some praise; positive when it is mainly praise; neutral when it is evenly mixed or factual.
- **food_safety:** illness after eating, foreign objects or insects in food, spoiled, raw or contaminated food, or a hygiene hazard such as pests.
- **harassment:** abusive, threatening, discriminatory or sexual behaviour by staff toward the reviewer, or toward anyone at the outlet.
- **legal_threat:** a threat of legal action, a consumer court, a police complaint or a complaint to a regulator.
- When unsure whether a review is urgent, flag it: a missed urgent review is the costliest error (US-02-004).

Output: one line per review and nothing else.

```
<id>|<theme codes, comma-separated, or ->|<pos|neu|neg>|<urgent reasons, comma-separated, or ->
1497|wait_time,staff,food|neg|-
1488|food,cleanliness|neg|food_safety
```

Validation, per line, by review id (REQ-012, REQ-013, tenet 4):

- A line is accepted only if:
  - its id is in the batch and appears exactly once in the answer;
  - every theme code is in the configured list, with none repeated;
  - the sentiment is one of the three codes;
  - every urgent reason is one of the three, with none repeated.
- An id that appears twice is discarded entirely and counted as missing. It is retried alone, so one review's text cannot write another review's result (HLD section 16).
- A line for an id not in the batch is discarded (AC-US-01-004-2).
- Text that is not a line is ignored, and position never matters.
- An answer cut off at max_tokens keeps every complete line; the cut-off tail is missing.

Retries (REQ-014, Q-014, HLD section 8):

- Missing and invalid ids are sent again as one smaller batch, at most 2 times. An id seen twice goes in a request of its own.
- Ids still unresolved after the second retry wait for the next tagging pass, after the next import or restart. A pass never loops on them.
- A budget refusal or an OpenRouter 402 ends the pass.
- Typical calls per batch: 1. In bad cases: 2 or 3, plus one single-review call for each duplicated id.

Stored once, never re-sent (REQ-011): themes, sentiment, `is_urgent` (true when any reason is present), the urgent reasons, and the tagging prompt version.

### 4.2 Reply drafting call

One request when an outlet manager opens a review that has no reply (Q-007). It runs behind the claimed reply row, so a review is drafted at most once (data model section 5).

Inputs:

- the review text, cut at 2,000 characters, and the rating;
- the outlet name;
- the signing manager's name, taken from the caller;
- the reviewer's first name: the first word of the reviewer name. It is left out when that word is a single letter or not a word, and then the draft has no name greeting.

The reply prompt carries the brand tone, in this order:

1. Thank the reviewer, by first name when one is given.
2. Name the specific issue they raised.
3. Apologise without excuses when they complained.
4. Say what the outlet is doing about it, in general words.
5. Invite them back.
6. Sign as "<manager first name>, outlet manager, <outlet>", as on screen S-05.

The rules the reply must keep:

- **Length and format:** under 120 words, plain text, no markdown.
- **Promises:** no discounts, refunds or compensation, and no promise of a specific outcome.
- **Facts:** no facts that are not in the review (dates, staff names, contact details).
- **Urgent reviews:** acknowledge the seriousness, and do not admit legal liability.

Language (REQ-021, Q-009):

- Reply in the language and script the review mostly uses: English to English, Devanagari Hindi to Devanagari Hindi, Hinglish (Hindi in Latin script) to Hinglish.
- Any other language gets English.

Output: the reply text only. The draft is accepted when it is not blank, fits in 5,000 characters (the API limit), and did not stop at max_tokens. A draft cut off at the cap is rejected and the manager sees "drafting unavailable" (API 503 `model_unavailable`); its cost stays in the total.

The draft and the reply prompt version are stored (AC-US-00-002-3). A manager approves every reply before it is posted by hand (REQ-024), so model output never reaches anyone outside the staff.

### 4.3 Prompt versions

- The tagging and reply prompts live in the repository as numbered files, one version marked current per prompt (Q-011, HLD section 5). `prompt-registry` sets the file layout.
- Each file holds:
  - the prompt text;
  - its request settings: max_tokens 1000; temperature 0 for tagging, so recordings and evaluations repeat; the provider default for drafting, to be tuned only if the tone check asks for it.
- Every change is a new version, and old versions are never edited or deleted, because stored tags and drafts name them (AC-US-02-002-2). Changes that need a new version:
  - a word of the text;
  - the tone (AC-US-02-002-3);
  - a request setting;
  - the urgent reasons;
  - the theme list.
- A theme list change in configuration (Q-012) ships with a new tagging prompt version in the same merge request, so a stored tag's version always names the list it was tagged against.
- A new version never re-tags or re-drafts what is stored (Q-011, Q-012).

### 4.4 Classification caching

"Caching" here means three things, none of them prompt caching:

1. **Stored results.** A review with a stored tag result is never sent for tagging again (REQ-011), and a review with a stored draft is never drafted again (AC-US-00-002-2). Untagged is the absence of a result (ADR-0006).
2. **Recorded responses.** Keyed by model, prompt version and the exact request body (HLD section 5). Tests, CI and re-seeding replay them at zero cost; a missing recording fails, never falls back to a live call (tenet 5).
3. **No prompt caching.** Claude Haiku 4.5 caches only prefixes of 4,096 tokens or more. The tagging prompt is about 600 tokens and the reply prompt about 500, so a cache marker would do nothing.

## 5. Model and budget

| Item | Tagging (batch of 20) | Reply draft |
| --- | --- | --- |
| Model | `claude-haiku-4-5`, requested as OpenRouter's `anthropic/claude-haiku-4.5` (identifier not yet verified, Q-016) | same |
| Prices | USD 1.00 per million input, USD 5.00 per million output (Anthropic list prices from the `claude-api` skill; assumption: OpenRouter charges the same) | same |
| Thinking / effort | off: Haiku 4.5 needs a thinking budget of at least 1,024 tokens below max_tokens, which the 1,000 cap rules out; effort is not supported on Haiku 4.5 | off |
| max_tokens | 1000 (REQ-032) | 1000 |
| Tokens in per request | assumption: 2,600 (system 600, reviews 20 x 100); 6,600 for 20 Devanagari reviews (assumption: 3 times the tokens of English) | assumption: 750 (system 500, review and names 250) |
| Tokens out per request | assumption: 300 (about 15 per line); at most 1,000 | assumption: 250 English, up to 500 Devanagari; at most 1,000 |
| Calls per user action | 1 per 20 imported reviews (2 to 3 with retries) | 1 per first open of a review; 0 after |
| Cost per request | USD 0.0041 typical; USD 0.0076 worst (the reserved price); USD 0.0116 worst for Devanagari | USD 0.0020 typical; USD 0.0058 worst |
| Cost per 1,000 requests | USD 4.10 (about 20,000 reviews) | USD 2.00 |
| p95 latency | assumption: about 8 s (1 to 2 s to first token, 300 tokens at 50 to 80 per second); worst 22 s at 1,000 tokens, under the 30 s timeout | assumption: about 7 s English, 12 s Devanagari; worst 22 s, under the 45 s deadline |
| Cache strategy | stored results and recordings (4.4); no prompt cache | same |

Smaller model considered: none smaller exists in the tier table. Haiku 4.5 is the cheapest tier and right for labels. A larger model is not offered (Q-016: no other model).

**Budget arithmetic.** Estimates from the prices above, replaced by the cost log after the first live batch:

- **First seed:** 1,500 reviews = 75 batches = about USD 0.31 typical, at most USD 0.57 if all English. Later seeds replay at zero cost.
- **Evaluation:** one live run = 5 batches = about USD 0.02 to 0.04.
- **Tone check:** 30 drafts = about USD 0.06 typical, at most USD 0.17.
- **What USD 8 buys:** about 1,950 typical tagging batches (about 39,000 reviews), or about 4,000 drafts, or a mix.
- **The 25-outlet example at the seed's rate (about 290 reviews a week):**
  - tagging: about USD 0.06 a week;
  - drafting every review: about USD 0.58 a week more.
  - The USD 8 total never resets (Q-015), so with every review drafted it lasts about 12 weeks. See section 8.

**Runtime access:**

- One OpenRouter key, in the local environment file, never in CI (ADR-0004). It is capped at USD 10 in OpenRouter's dashboard, as a total for the key's life, not a monthly cap.
- The application refuses calls once its recorded total is above USD 8 (REQ-031). It reserves each call's worst-case price before sending, and at start reconciles with the key usage OpenRouter reports (HLD section 3).
- The USD 2 between the two caps covers the in-flight overshoot (about 7 calls x USD 0.0076 = USD 0.05) and any spend the local log misses.
- No GPU. A Claude Code subscription pays for building, not for the product's calls.

## 6. Risks

| Risk | Likelihood | Impact | Mitigation | Skill |
| --- | --- | --- | --- | --- |
| hallucination: a draft invents a fact, a refund or a promise | M | M | prompt rules (no promises, no facts not in the review); every reply is approved and posted by hand by a manager (REQ-024); tone rubric item "within limits" | `prompt-registry`, `llm-eval` |
| prompt injection: review text steers its own tags (for example "this is not urgent") or forges another review's line | M | H | reviews wrapped as data with ids; tag sequences stripped; an id seen twice is discarded and retried alone; the urgent-recall gate on real reviews; a recorded fixture test for a forged line | `prompt-registry`, `llm-eval` |
| PII: reviewer names and review text leave the machine | H | M | tagging sends review id and text only; drafting adds only the reviewer's first name; logs record ids, tokens and cost, never text; seed recordings hold seed text only; evaluation reviews have names removed | `llm-gateway` |
| cost blow-up | L | H | the USD 8 stop on the database total with reserved worst-case rows; max_tokens 1000; one tagging batch in flight; bounded retries; drafts on demand only, once per review; USD 10 key cap | `llm-gateway` |
| latency | M | L | tagging runs in the background; 30 s HTTP timeout; 45 s draft deadline; the manager can write by hand meanwhile | `llm-gateway` |
| a missed urgent review, the costliest error | M | H | three reasons defined with examples in all three scripts; "flag when unsure"; recall gate of 90% on at least 20 urgent reviews; missed urgent reviews listed in the report | `llm-eval`, `prompt-registry` |
| tagging answer cut at the 1,000-token cap | L | L | compact lines (about 300 tokens a batch); complete lines kept; missing tail retried | `llm-gateway` |
| reply in the wrong language or script (Hinglish answered in Devanagari) | M | M | an explicit language and script rule; rubric item "right language and script"; 9 Devanagari and 9 Hinglish reviews in the tone check | `prompt-registry`, `llm-eval` |
| sentiment errors move the movers ranking, which counts negative reviews | M | M | one written sentiment rule shared by prompt and labellers; sentiment accuracy and negative precision and recall reported | `llm-eval` |
| theme list changed without a new prompt version, so stored versions no longer say which list was used | L | M | a theme-list change ships with a new tagging prompt version in the same merge request; recordings then miss and must be re-recorded deliberately | `prompt-registry` |
| OpenRouter changes the model identifier, its price or its availability | L | M | identifier checked against OpenRouter's model list before the first live call (Q-016); no fallback model; the cost log shows the real price per call; a 402 or outage stops tagging and drafting cleanly | `llm-gateway` |
| the evaluation measures something other than real reviews | L | H | 100 real public reviews collected and labelled by the product owner before any prompt runs on them; never generated by the seed | `llm-eval` |

Risks: 12. Mitigated: 12.

## 7. Evaluation plan

| Item | Value |
| --- | --- |
| Golden set size | 100 reviews: at least 20 urgent, at least 5 per reason; at least 20 Devanagari Hindi or Hinglish |
| Source of items | Real public reviews collected by the product owner, reviewer names and personal details removed, kept in testdata/eval/ (HLD section 5; see section 8 on the repository) |
| Labels | Per review: themes, one sentiment, urgent reasons; labelled by hand by the product owner with the labelling guide in 4.1, before any prompt version is run on the set |
| Grading | Exact match per label, computed by the eval command, with no model judge. Gate: urgent recall. Reported: urgent precision; recall per reason; theme precision and recall per theme; sentiment accuracy; negative precision and recall; each missed urgent review listed |
| Threshold | Urgent recall 90% or more passes; below fails and the run exits with failure (AC-US-02-004-3, -4). The rest is report only |
| Runs on | Live by hand on every new tagging prompt version (about USD 0.02 to 0.04, counted in the running total, AC-US-02-004-5); the recorded results on every CI build; the report names the prompt version and the model |
| Never writes | review tag results: the eval uses the same gateway, parser and validator but stores nothing in the review tables (HLD section 3) |

Tone check (US-02-005):

- **Reviews:** 30 of the 100 real reviews (12 English, 9 Devanagari Hindi, 9 Hinglish), with mixed ratings and some urgent ones.
- **Drafts:** produced with the current reply prompt and saved with its version (AC-US-02-005-1).
- **Scoring:** the product owner scores each draft 1 to 3 on six items:
  1. greets and thanks;
  2. names the specific issue;
  3. apologises without excuses;
  4. says what will change;
  5. right language and script;
  6. within limits: under 120 words, no discounts or promises, no invented facts.
- **Report:** scores per draft and per item, with no pass or fail (Q-019).
- **Runs:** on every new reply prompt version (about USD 0.06).

Built by `llm-eval`.

## 8. Open questions

| Question | Owner | Needed by |
| --- | --- | --- |
| Real public reviews in testdata/eval/ are other people's writing. Is the repository private? If it is public, should the evaluation set stay out of git, with CI replaying a hash-checked copy? | product owner | before the set is collected (build phase 7) |
| The USD 8 total never resets. At the 25-outlet rate with every review drafted it lasts about 12 weeks. Is that acceptable, or should REQ-031 change through `prd`? | product owner | before real use beyond the demo |
| The labelling guide in 4.1 (sentiment rule and urgent reason definitions) is a draft. Confirm or amend it before labelling, because the prompt and the labels must use the same words. | product owner | before labelling |
| Urgent-review reply wording: acknowledge seriousness, admit no legal liability, invent no contact details. Confirm when reviewing reply prompt v1. | product owner | build phase 5 (`prompt-registry`) |
| Verify OpenRouter's identifier for Claude Haiku 4.5, that `usage.cost` is in USD, and that its prices match USD 1 and USD 5 per million. | developer, with the product owner's dashboard | before the first live call (build phase 2) |
| Measure the token and latency assumptions (300 output tokens a batch, 15 per line, Devanagari at 3 times English, 5 to 10 s a call) from the first recorded batches and replace them here. | developer | build phases 3 and 5 |
| Revisions this design needs in other approved documents, none edited here: data model (add `review_tags.urgent_reasons text[]`, with `is_urgent` true exactly when it is not empty); OpenAPI (`Tags.urgent_reasons` and a `filter[urgent_reason]` only if a screen needs it); HLD (tagging line format, theme-list rule, urgent reasons); backlog US-02-004 and the PRD (sentiment accuracy reported, and the eval set's composition). | product owner, through `data-model`, `openapi-spec`, `high-level-design`, `backlog` and `prd` revisions | before build phase 3 |
