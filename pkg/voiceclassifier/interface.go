package voiceclassifier

import (
	"context"
	"time"

	"github.com/tryhavana/take-home-test/pkg/common"
	"github.com/tryhavana/take-home-test/pkg/svc"
)

type ClassifyParams struct {
	Transcript string
	Timezone   string
	CalledAt   time.Time
}

type ClassifyResponse struct {
	Intent         common.Intent  `json:"intent"`
	Verdict 	   string         `json:"verdict"`
	InterestedTime *time.Time     `json:"interestedTime,omitempty"`
	CallBackTime   *time.Time     `json:"callBackTime,omitempty"`
	Evidence 	   []string 	  `json:"evidence"`
	NeedsReview    bool    		  `json:"needsReview"`
	DecisionReason DecisionReason `json:"decisionReason,omitempty"`
	Flags 		   []Flag `json:"flags,omitempty"`
}

type ClassifierInterface interface {
	Classify(ctx context.Context, senv *svc.Env, params ClassifyParams) (*ClassifyResponse, error)
}

type firstCall struct {
	Intent         common.Intent  `json:"intent"`
	Verdict        string   	  `json:"verdict"`
	Evidence       []string 	  `json:"evidence"`
	AgreedDatetime string  		  `json:"agreedDatetime"`
}

