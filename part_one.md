## 1. Polling Changes

The current implementation performs classification every **5 seconds**.

I would recommend **reducing the polling frequency**. For example, it could be adjusted based on the timezone — during night hours, polling can be paused or slowed since the likelihood of incoming calls is low.

However, upon further consideration, since classification doesn’t need to be immediate, a **batch processing** approach (e.g., every few hours) would be more efficient than relying on timezone-based scheduling. This avoids additional complexity in determining time periods and tracking active hours, assuming there’s no business-critical need for real-time classification.

Additionally, the **polling frequency** should be moved into `config.yaml` to maintain clean configuration separation.

Instead of continuous polling, an **event-driven architecture** could be implemented — classification should trigger whenever a new call is created. Given the current codebase structure (where there’s no central orchestrator), directly calling the classification after saving the voice call in `xxxx.go` would introduce tight coupling. 

To maintain **separation of concerns** and modularity, an **event-driven system** (e.g., using a Redis Pub/Sub mechanism) could be used. This could be introduced in a future PR as part of broader improvements.

---

## 2. Reclassification of All Calls in an Unclassified Thread

The current logic assumes that if a thread is marked as `voice_call_unclassified`, all its calls are unclassified and must be processed. It does **not check if individual calls are already classified**, which leads to reclassification of some calls and results in **duplicate or conflicting records**.

In practice, a thread may have multiple calls over time, with some already classified and others not. We should **only classify unclassified calls**.

To fix this, before classifying, we should:
- Use `CampaignThreadID` and `CalledAt` to identify a specific `VoiceCall`.
- Check if it already exists in the `Classifications` table.

This ensures:
- Each call is only classified once.
- Duplicate and conflicting classifications are avoided.
- Unnecessary reprocessing is minimized.

---

## 3. Error Handling When Classifying Calls

Currently, the code processes each call in a loop and **returns immediately upon failure**. As a result:
- Remaining calls in the thread are not processed.
- Some `Classification` records may already be inserted for earlier calls.
- Retries could lead to **duplicate records**.

This issue would be mitigated by **Solution 2** (checking for unclassified calls before classification), but additionally:
- The classification loop should handle errors gracefully and continue processing subsequent calls.
- Failures should be logged and retried later without aborting the entire thread’s classification.

---

## 4. Each Classification Record Update as a Single DB Transaction

The current implementation separates:
1. Insertion of `Classification` records.
2. Updating of the thread’s status.

If a failure occurs between these steps, it can lead to **inconsistent state** — some classifications may be recorded, but the thread remains unupdated.

To prevent this:
- Use a **single database transaction** per thread classification.
- Wrap all related DB operations (inserting `Classification` records and updating thread status) within one transaction.
- Roll back the transaction entirely upon any failure to ensure data consistency.

---
