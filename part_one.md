## 1. Polling Changes

The current implementation is to do classification every 5s.

I would reduce the polling frequency. Since the classification isn’t needed immediately (most of the time? Unless the scheduled call is a while later <1h), we can batch process it every x hour rather than such at such high frequency of 5s. 

Also, I would put the polling frequency in `config.yaml` to keep things separated.

Instead of polling, I would make it event driven where I only do classification every time there’s a new call, but this could be done in a separate PR or as part of future improvements. 

---

## 2. Reclassification of All Calls in an Unclassified Thread

The current logic assumes that if a thread is marked as `voice_call_unclassified`, then all its calls are unclassified and needs to be processed. It does not check if a call already has a classification, so it reclassifies all calls within a thread, of which some were already classified, resulting in duplicate or conflicting records. 

In real-world scenarios, a thread may have multiple calls over time, with some already classified and others not. We should only classify calls that haven’t been classified. 

We can add an additional check on whether a call is classified by using `CampaignThreadID` and `CalledAt` to identify a specific `VoiceCall` and check it against `Classifications` table, before classifying it. This would prevent duplicated classifications and possible conflicting classifications, ensuring each call is only classified once.

----

## 3. Error Handling When Classifying Calls

The current code processes each call in a loop based on a thread and returns immediately if the call fails. As a result, the other calls in the thread are not processed and `Classification` records would have been inserted for previously classified calls. This would cause duplicated records when this thread undergoes classification in further retries.

This would not be an issue if we adopt problem 2’s solution by adding a check for unclassified calls.

---

## 4. Each Classification Record Update as a Single DB Transaction

The current code separates inserting `Classification` records and updating thread’s status into two DB calls. if something fails after classifying some calls but before updating the thread status, it might cause a discrepancy the classification and thread status.

Instead, I would use a single database transaction for all DB changes related to a thread where the insertion of `Classification` records and updating the status are within one transaction such that the transaction is rolled back if it fails and the system remains consistent.

---
