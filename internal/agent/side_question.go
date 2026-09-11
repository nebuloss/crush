package agent

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"charm.land/fantasy"

	"github.com/charmbracelet/crush/internal/csync"
)

// This file implements side questions (the `/btw` command): a one-shot,
// tool-less model call answered from the session's context while the main
// agent keeps working.
//
// It is kept in its own file so the edits it needs in agent.go and
// coordinator.go are limited to interface lines, which keeps this patch
// applying cleanly across upstream releases.

// sideQuestionReminder is prepended to the user's question. Most of it exists
// to stop a tool-less model from promising actions it cannot take ("Let me
// check that file…" followed by nothing), which is worse than an honest "I
// don't know from the context".
const sideQuestionReminder = `<system-reminder>This is a side question from the user. You must answer this question directly in a single response.

IMPORTANT CONTEXT:
- You are a separate, lightweight agent spawned to answer this one question
- The main agent is NOT interrupted - it continues working independently in the background
- You share the conversation context but are a completely separate instance
- Do NOT reference being interrupted or what you were "previously doing" - that framing is incorrect

CRITICAL CONSTRAINTS:
- You have NO tools available - you cannot read files, run commands, search, or take any actions
- This is a one-off response - there will be no follow-up turns
- You can ONLY provide information based on what you already know from the conversation context
- NEVER say things like "Let me try...", "I'll now...", "Let me check...", or promise to take any action
- If you don't know the answer, say so - do not offer to look it up or investigate

Simply answer the question with the information you have.</system-reminder>

%s`

// sideQuestionFallbackPrompt is used only when the session agent has no system
// prompt yet. Normally a side question reuses the MAIN agent's system prompt
// verbatim - see SideQuestion for why that is load-bearing rather than lazy.
const sideQuestionFallbackPrompt = `You are answering a side question about an in-progress coding session. Answer from the conversation context only, concisely and directly. You have no tools.`

// deniedToolMessage is what a tool returns if the model calls one anyway.
const deniedToolMessage = "Side questions cannot use tools. Answer directly from the conversation context."

// denyingTool wraps a real tool so its schema is still sent to the provider -
// which is what keeps the cached prompt prefix intact - while refusing to
// actually run. It embeds the original for Info and ProviderOptions, so the
// serialized definition is byte-identical to the main agent's.
//
// It must never call SetProviderOptions: the wrapped tools are the same
// instances the main agent uses, and mutating them here would corrupt the
// running conversation's cache markers.
type denyingTool struct {
	fantasy.AgentTool
}

func (denyingTool) Run(context.Context, fantasy.ToolCall) (fantasy.ToolResponse, error) {
	return fantasy.NewTextErrorResponse(deniedToolMessage), nil
}

// SideQuestionUsage reports what one side question cost. It exists so the
// cache behaviour can be measured: if CacheReadTokens stays zero on a large
// session, every side question is re-sending the whole conversation as fresh
// input and the feature is too expensive to ship.
type SideQuestionUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
}

// SideQuestionResult is one answered side question.
type SideQuestionResult struct {
	Answer string
	Usage  SideQuestionUsage
	Model  string
	// ToolCallAttempted is set when the model emitted a tool call despite
	// having none bound. Some providers do this regardless; surfacing it
	// beats rendering an empty answer.
	ToolCallAttempted bool
}

// sideExchange is one prior question/answer pair, kept so successive side
// questions form a thread.
type sideExchange struct {
	question string
	answer   string
}

// sideQuestionHistory holds per-session side-question threads. It is
// deliberately in-memory only: a side question is never written to the
// transcript, so there is nothing to restore after a restart.
type sideQuestionHistory struct {
	mu sync.Mutex
	by *csync.Map[string, []sideExchange]
}

func newSideQuestionHistory() *sideQuestionHistory {
	return &sideQuestionHistory{by: csync.NewMap[string, []sideExchange]()}
}

func (h *sideQuestionHistory) get(sessionID string) []sideExchange {
	v, _ := h.by.Get(sessionID)
	return v
}

func (h *sideQuestionHistory) append(sessionID string, ex sideExchange) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cur, _ := h.by.Get(sessionID)
	h.by.Set(sessionID, append(append([]sideExchange{}, cur...), ex))
}

func (h *sideQuestionHistory) clear(sessionID string) { h.by.Del(sessionID) }

// ErrEmptySideQuestion is returned when there is no question to ask.
var ErrEmptySideQuestion = errors.New("side question is empty")

// SideQuestion answers one question from the session's context without
// touching the session: no tools, a single turn, and nothing written back to
// the transcript.
//
// It deliberately bypasses the per-session dispatch lock - answering while the
// agent works is the entire point - which is safe precisely because it mutates
// no session state. In particular it must NOT register in a.activeRequests the
// way Summarize does: that entry belongs to the running turn, and overwriting
// it would break cancelling the turn the user actually cares about.
func (a *sessionAgent) SideQuestion(ctx context.Context, sessionID, question string) (SideQuestionResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return SideQuestionResult{}, ErrEmptySideQuestion
	}

	currentSession, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return SideQuestionResult{}, fmt.Errorf("failed to get session: %w", err)
	}
	msgs, err := a.getSessionMessages(ctx, currentSession)
	if err != nil {
		return SideQuestionResult{}, err
	}

	// A side question must reuse the MAIN conversation's model, system prompt
	// and tool definitions. Anthropic caches by strict prefix, so changing any
	// of them breaks the match at position 0 and the whole history is re-sent
	// as fresh input. Measured against the gateway on a ~11.4k-token history:
	// reusing them read 11406 cached tokens; a bespoke prompt with no tools
	// read 0 and created a second, redundant cache entry.
	//
	// This is why the side question's framing lives in the user message (see
	// sideQuestionReminder) rather than in the system prompt, and why the
	// tools are present-but-denying rather than absent.
	model := a.largeModel.Get()
	systemPrompt := cmp.Or(a.systemPrompt.Get(), sideQuestionFallbackPrompt)
	systemPromptPrefix := a.systemPromptPrefix.Get()

	// Reuse the main path's converter rather than writing a second one that
	// could disagree with it about orphaned tool calls or image handling.
	history, _ := a.preparePrompt(msgs, model.CatwalkCfg.SupportsImages)

	// Thread prior side questions on the end, after the real conversation, so
	// the shared prefix the provider caches stays byte-identical to the main
	// conversation's.
	for _, ex := range a.sideQuestions.get(sessionID) {
		history = append(history,
			fantasy.NewUserMessage(ex.question),
			fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: ex.answer}},
			},
		)
	}

	prompt := fmt.Sprintf(sideQuestionReminder, question)

	// Same tool schemas as the main agent, each refusing to run. Dropping the
	// tools entirely would be simpler but would break the cache prefix.
	mainTools := a.tools.Copy()
	denied := make([]fantasy.AgentTool, 0, len(mainTools))
	for _, t := range mainTools {
		denied = append(denied, denyingTool{AgentTool: t})
	}

	tok := model.CatwalkCfg.DefaultMaxTokens
	if tok <= 0 {
		tok = 2048
	}
	opts := []fantasy.AgentOption{
		fantasy.WithSystemPrompt(systemPrompt),
		fantasy.WithMaxOutputTokens(tok),
		fantasy.WithUserAgent(userAgent),
		// One step: there is no follow-up turn to recover in, and a model
		// that keeps retrying denied tools must be stopped, not waited on.
		fantasy.WithStopConditions(fantasy.StepCountIs(1)),
	}
	if len(denied) > 0 {
		opts = append(opts, fantasy.WithTools(denied...))
	}

	streamCall := fantasy.AgentStreamCall{
		Prompt:   prompt,
		Messages: history,
		Headers:  sessionHeaders(sessionID),
		PrepareStep: func(callCtx context.Context, stepOpts fantasy.PrepareStepFunctionOptions) (_ context.Context, prepared fantasy.PrepareStepResult, err error) {
			prepared.Messages = stepOpts.Messages

			// Mirror the main run path's cache breakpoints exactly: last
			// system message, then the last two messages. Placing them
			// anywhere else would produce a different prefix and miss the
			// conversation's cache.
			lastSystemRoleInx := 0
			systemMessageUpdated := false
			for i, msg := range prepared.Messages {
				if msg.Role == fantasy.MessageRoleSystem {
					lastSystemRoleInx = i
				} else if !systemMessageUpdated {
					prepared.Messages[lastSystemRoleInx].ProviderOptions = a.getCacheControlOptions()
					systemMessageUpdated = true
				}
				if i > len(prepared.Messages)-3 {
					prepared.Messages[i].ProviderOptions = a.getCacheControlOptions()
				}
			}

			if systemPromptPrefix != "" {
				prepared.Messages = append([]fantasy.Message{
					fantasy.NewSystemMessage(systemPromptPrefix),
				}, prepared.Messages...)
			}
			return callCtx, prepared, nil
		},
	}

	// Deliberately no small-model fallback: the small model cannot read the
	// large model's cache (caches are per-model), so falling back would turn a
	// cache hit into a full re-send of the conversation.
	resp, err := fantasy.NewAgent(model.Model, opts...).Stream(ctx, streamCall)
	if err != nil {
		if ctx.Err() != nil {
			// The user dismissed the panel or quit; not a model failure.
			return SideQuestionResult{}, ctx.Err()
		}
		slog.Error("Side question failed", "err", err)
		return SideQuestionResult{}, err
	}
	used := model

	answer := strings.TrimSpace(resp.Response.Content.Text())
	answer = thinkTagRegex.ReplaceAllString(answer, "")
	answer = orphanThinkTagRegex.ReplaceAllString(answer, "")
	answer = strings.TrimSpace(answer)

	// Some providers emit a tool call even with no tools bound. Say so rather
	// than rendering an empty answer.
	toolCallAttempted := false
	if answer == "" {
		if calls := resp.Response.Content.ToolCalls(); len(calls) > 0 {
			toolCallAttempted = true
			answer = fmt.Sprintf(
				"(The model tried to call %s instead of answering directly. Try rephrasing, or ask in the main conversation.)",
				calls[0].ToolName,
			)
		}
	}
	if answer == "" {
		answer = "(No answer returned.)"
	}

	a.sideQuestions.append(sessionID, sideExchange{question: question, answer: answer})

	return SideQuestionResult{
		Answer: answer,
		Usage: SideQuestionUsage{
			InputTokens:         resp.TotalUsage.InputTokens,
			OutputTokens:        resp.TotalUsage.OutputTokens,
			CacheReadTokens:     resp.TotalUsage.CacheReadTokens,
			CacheCreationTokens: resp.TotalUsage.CacheCreationTokens,
		},
		Model:             used.ModelCfg.Model,
		ToolCallAttempted: toolCallAttempted,
	}, nil
}

// ClearSideQuestions drops the side-question thread for a session.
func (a *sessionAgent) ClearSideQuestions(sessionID string) {
	a.sideQuestions.clear(sessionID)
}
