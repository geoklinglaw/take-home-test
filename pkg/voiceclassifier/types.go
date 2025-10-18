package voiceclassifier

type DecisionReason string

const (
	// Verdict gates
	DecisionVerdictInsufficient DecisionReason = "verdict_insufficient"
	DecisionVerdictContradicted DecisionReason = "verdict_contradicted"

	// Evidence gates
	DecisionNoEvidence              DecisionReason = "no_evidence"
	DecisionBadEvidenceNotUser      DecisionReason = "bad_evidence_not_user"
	DecisionEvidenceNotInTranscript DecisionReason = "evidence_not_in_transcript"

	// Prefilters
	DecisionPrefilterVoicemail   DecisionReason = "prefilter_voicemail"
	DecisionPrefilterWrongNumber DecisionReason = "prefilter_wrong_number"
	DecisionPrefilterHangup      DecisionReason = "prefilter_immediate_hangup"

	// Policy/guardrails
	DecisionContradictionFlag DecisionReason = "contradiction_flag"
	DecisionHedgeFlag         DecisionReason = "hedge_flag"
	DecisionAgentOnlyOffer    DecisionReason = "agent_only_offer"
	DecisionSensitiveTerm     DecisionReason = "sensitive_term"
)


type Flag string

const (
	FlagHedge          Flag = "hedge"
	FlagContradiction  Flag = "contradiction"
	FlagAgentOnlyOffer Flag = "agent_only_offer"
	FlagSensitive      Flag = "sensitive_term"
	FlagWeakEvidence   Flag = "weak_evidence"
)
