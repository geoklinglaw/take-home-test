package voiceclassifier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/tryhavana/take-home-test/pkg/common"
	"github.com/tryhavana/take-home-test/pkg/svc"
)

const classifySystemPrompt = `
You are an responsible for classifying incoming call transcripts into relevant intent types so that the humans that are assigned to follow up know what is the best course of action.
You will be given a call transcript.
Focus exclusively on the user's reply, i.e. sentences that starts with 'User: '.
Read the transcripts carefully and classify the call transcript into the following categories:
- 'voice_interested': The User EXPLICITLY indicated he/she is interested in the course or the university.
- 'voice_not_interested': The user EXPLICITLY say that he/she is not interested in the course or the university.
- 'voice_immediate_hangup': The user hangs up without expressing their full intent. This includes cutting off halfway through when the caller is talking, or no substantial discussion after exchanging greetings.
- 'voice_wrong_number': The user who answered the phone indicated that we are calling the wrong number.
- 'voice_no_action': The student has already signed up to the course or is in contact with the advisor. Only use this if the student does not need any additional help.
- 'voice_wants_call_back': The user EXPLICITLY request that he/she wants a call back.
- 'voice_wants_email_follow_up': The user wished to follow up through email.
- 'voice_wants_whatsapp_sms_follow_up': The user wished to follow up through instant messaging, such as via Whatsapp or SMS.
- 'voice_voice_mail': The call goes into an automated reply or a voice mail. Reply does not come from an actual user.
- 'voice_unknown': Anything that does not fit into the above categories.

Verdict rules (about your own label choice):
- "supported"     : You can quote clear User: evidence that directly satisfies the label definition.
- "insufficient"  : You cannot find clear, explicit User: evidence for any label above.
- "contradicted"  : The strongest User: quotes point to a different label than the one you would pick.

Evidence rules:
- Return 1–3 "User:" quotes that justify the intent.
- Quotes MUST start with "User:" and be verbatim substrings from the transcript.
- Do NOT include agent lines.
- Keep quotes short but complete.

Time rules (agreedDatetime):
- If the USER explicitly agrees to a specific callback/meeting time/date, output it as "YYYY-MM-DDTHH:MM:SS" (no timezone).
- Use the provided "ConversationLocalTime" as the reference for resolving relative dates like "tomorrow" or day-of-week.
- If there is NO explicit user-agreed time, output "null". Do NOT guess.

Return exact JSON:
{
  "intent": "voice_*",
  "verdict": "supported" | "insufficient" | "contradicted",
  "evidence": ["User: ...", "..."],
  "agreedDatetime": "YYYY-MM-DDTHH:MM:SS" | "null"
}
`

const classifyUserPrompt = `
Transcript:
--- BEGIN TRANSCRIPT ---
%s
--- END TRANSCRIPT ---

Remember to focus on the User's replies to derive their intent, and not the Agent's!
Reply in the following JSON format {'intent': <category>}
`

const extractInterestedPrompt = `
NEXT OBJECTIVE:
Extract the date and time which the user has agreed to meet the enrollment advisor. Output your answer in ISO time format (without time zone) only (e.g. 2006-01-02T15:04:00).
Extract the time as is, do not do any timezone conversion. If no date and time is given, output 'null' value.

CONTEXT:
The time and date during the conversation is %s.

Reply in the following JSON format {'agreedDatetime': <iso_datetime>}
`

const extractWantsCallBackPrompt = `
NEXT OBJECTIVE:
Extract the date and time which the user has agreed for the callback.

RULES YOU MUST FOLLOW:
- Output your answer in ISO time format (without time zone) only (e.g. 2006-01-02T15:04:00).
- Extract the time as is, DO NOT perform any timezone conversion back to UTC. i.e. If the user mention 9PM Singapore Time, then just take the time as 9PM.
- If user mention a time without AM or PM, assume it is the time when most people are active. i.e. 5:30 means 5:30PM, 12:30 means 12:30PM.
- REMEMBER to convert the time appropriately to 24-hour format. i.e. 6PM means 18:00, not 6:00.

INSTRUCTIONS:
Extract the agreed date time for a call back.
If the user is ambiguous, use the best of your ability to determine the most appropriate agreed date and time that is in line with the intent of this objective.
If you were to decide an agreed date time, remember to keep within human active hours.
If there is little hint on when is the time to call back, just set it to the next available 6PM (18:00H).
Do not leave empty.

CONTEXT:
The time and date during the conversation is %s.

Reply the case number corresponding to the situation and the agreed date time in the following JSON format: {'agreedDatetime': <iso_datetime>}
`

type Classifier struct{}

var _ ClassifierInterface = (*Classifier)(nil)

func (c *Classifier) Classify(ctx context.Context, senv *svc.Env, params ClassifyParams) (*ClassifyResponse, error) {
	if pre, hit := runPrefilters(params.Transcript); hit {
		return pre, nil
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: classifySystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf(classifyUserPrompt, params.Transcript),
		},
	}
	resp, err := senv.OpenAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4.1",
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.0,
		Messages:    messages,
	})
	if err != nil {
		return nil, errors.Wrap(err, "openai create chat completion")
	}
	var result1 firstCall
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result1); err != nil {
		return nil, errors.Wrap(err, "fail to parse response")
	}

	ret := &ClassifyResponse{
		Intent:   common.Intent(fc.Intent),
		Verdict:  fc.Verdict,
		Evidence: append([]string(nil), fc.Evidence...),
	}

	if ret.Verdict == "" {
		ret.Verdict = "insufficient"
	}
	if !applyVerdictEvidenceGates(params.Transcript, ret) {
		return ret, nil
	}

	if ret.Intent == common.IntentVoiceInterested || ret.Intent == common.IntentVoiceWantsCallBack {
		location, err := time.LoadLocation(params.Timezone)
		if err != nil {
			return nil, errors.Wrap(err, "error loading location timezone")
		}
		calledAtConverted := params.CalledAt.In(location).Format("2006-01-02 15:04:05, Monday")

		var prompt string
		if ret.Intent == "voice_interested" {
			prompt = fmt.Sprintf(extractInterestedPrompt, calledAtConverted)
		} else {
			prompt = fmt.Sprintf(extractWantsCallBackPrompt, calledAtConverted)
		}
		messages = append(
			messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: resp.Choices[0].Message.Content,
			},
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		)
		resp, err := senv.OpenAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: "gpt-4.1",
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			Temperature: 0.0,
			Messages:    messages,
		})
		if err != nil {
			return nil, errors.Wrap(err, "openai create chat completion")
		}
		var result2 map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result2); err != nil {
			return nil, errors.Wrap(err, "fail to parse response")
		}

		if agreedDatetime, ok := result2["agreedDatetime"].(string); ok {
			if agreedDatetime != "" && agreedDatetime != "null" {
				tUTC, err := time.Parse("2006-01-02T15:04:05", agreedDatetime)
				if err != nil {
					return nil, errors.Wrap(err, "error parsing time")
				}
				t := time.Date(tUTC.Year(), tUTC.Month(), tUTC.Day(), tUTC.Hour(), tUTC.Minute(), tUTC.Second(), tUTC.Nanosecond(), location)
				if ret.Intent == common.IntentVoiceInterested {
					ret.InterestedTime = &t
				} else if ret.Intent == common.IntentVoiceWantsCallBack {
					ret.CallBackTime = &t
				}
			}
		}
	}
	return ret, nil
}


func applyVerdictEvidenceGates(transcript string, ret *ClassifyResponse) bool {
	// verdict must be supported
	switch ret.Verdict {
	case "supported":
	case "insufficient":
		ret.NeedsReview = true
		ret.DecisionReason = DecisionVerdictInsufficient
		return false
	case "contradicted":
		ret.NeedsReview = true
		ret.DecisionReason = DecisionVerdictContradicted
		return false
	default:
		ret.NeedsReview = true
		ret.DecisionReason = DecisionVerdictInsufficient
		return false
	}

	// evidence must be present and has "User:"" quotes
	if len(ret.Evidence) == 0 {
		ret.NeedsReview = true
		ret.DecisionReason = DecisionNoEvidence
		return false
	}
	for _, q := range ret.Evidence {
		if !strings.HasPrefix(q, "User:") {
			ret.NeedsReview = true
			ret.DecisionReason = DecisionBadEvidenceNotUser
			return false
		}
		if !strings.Contains(transcript, q) {
			ret.NeedsReview = true
			ret.DecisionReason = DecisionEvidenceNotInTranscript
			return false
		}
	}

	return true
}