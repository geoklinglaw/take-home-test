

## Proposed Flow: 
![alt text](<image.png>)

## Main Changes: 

**1) Tighten inputs before the model** 

Use regex prefilters to catch voicemail, wrong number, clear hangup, since these are pattern-based and reliable without an LLM

**2) Make the model prove itself**

Considering opposing views / evidence {intent, verdict (“supported/insufficient/contradicted”)  evidence_lists[], agreedDatetime} reduces hallucinations visible and signal risky classified intents

**3) Policy by intent risk tiers**

Since different intents have various business impact, we should classify them such that low-consequence intents are more likely to be auto-approved aggressively while high-consequence intents face tighter evidence requirements, keeping quality high and manual review low where it matters most.

**LOW tier**

voice_voice_mail, voice_wrong_number, clear voice_immediate_hangup 
* Auto-approval rule: Approve when verdict = "supported" and there is at least one valid, verbatim User: evidence span.
* Rationale: these actions are pattern-driven and easily reversible. Hence, the operational risk of a false auto-approval is minimal.

**MID tier**

voice_not_interested, voice_wants_email_follow_up, voice_wants_whatsapp_sms_follow_up, voice_no_action
* Auto-approval rule: Approve only when the user’s request is explicit in the User: evidence, verdict = "supported", and no contradiction/hedge flags are present.
* Rationale: these actions are generally reversible but carry higher user and workflow cost than low, require unambiguous user language and clean evidence.

**HIGH tier**

voice_interested, voice_wants_call_back
* Auto-approval rule: Approve only with explicit, verbatim User: evidence, verdict = "supported", no flags, and valid time.
* Rationale: these actions trigger substantive follow-ups and affect customer experience. Errors are also costly, so stricter conditions are necessary.


**4) Rule layer to catch the classic traps**
* Contradiction flags: “not interested” occurring with a predicted intent of “wants_*” without an explicit ask.
* Hedge flags: “maybe / later / I’ll reach out” with no explicit channel/time.
* Agent-only-offer: agent proposes a channel/time but user never accepts.

**5) Improve the classifier itself**
* Prompting: simpler wording + a couple of real examples 
* Model choice: evaluate a smaller/cheaper model vs a stronger one; pick per cost/quality.
* Fine-tuning: use examples particularly those with confusing pairs to stabilize the model’s behavior on those cases.



## Plan

Week 1
* Add basic regex filters
* Improve prompt wording and add more output fields
* Explore various LLM models
* Add verdict gate rules


Week 2
* Add contradiction/hedge patterns (negators, agent-only-offer, sensitive terms)
* Add a weak-evidence check (too few/too vague User: quotes): keep in review
* Fine-tune the first LLM call on a small, representative set to stabilize intent + evidence_spans + verdict
* Add AutoApprovePolicy


Week 3
* Update AutoApprovePolicy 
* Grow the fine-tune and use the FT model for the first call.


Week 4 (buffer)
* Refine contradiction/hedge lists
* Final fine-tune pass using Week 2–3 error cases



