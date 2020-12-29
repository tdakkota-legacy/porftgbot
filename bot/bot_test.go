package bot

import (
	"context"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/tdakkota/porftgbot/runner"
)

type mockRunner struct {
	r runner.Result
}

func (m mockRunner) Query(ctx context.Context, q runner.Query) (runner.Result, error) {
	return m.r, nil
}

type InvokerFunc func(ctx context.Context, input bin.Encoder, output bin.Decoder) error

func (i InvokerFunc) InvokeRaw(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
	return i(ctx, input, output)
}

func TestBot(t *testing.T) {
	a := require.New(t)
	logger := zaptest.NewLogger(t)
	queryID := int64(10)
	query := "wtf"
	answer := "abc"

	raw := tg.NewClient(InvokerFunc(func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		b := bin.Buffer{}
		if err := input.Encode(&b); err != nil {
			return err
		}
		req, ok := input.(*tg.MessagesSetInlineBotResultsRequest)
		a.Truef(ok, "unexpected type %T", input)
		a.Equal(queryID, req.QueryID)

		a.NotEmpty(req.Results)
		result, ok := req.Results[0].(*tg.InputBotInlineResult)
		a.Truef(ok, "unexpected type %T", req.Results[0])
		a.Equal(query+answer, result.Description)

		msg, ok := result.SendMessage.(*tg.InputBotInlineMessageText)
		a.Truef(ok, "unexpected type %T", result.SendMessage)
		a.Equal(query+answer, msg.Message)
		return nil
	}))
	r := mockRunner{
		r: runner.Result{Replies: []string{answer}},
	}
	bot := NewBot(r, raw, logger.Named("bot"))

	ctx := tg.UpdateContext{
		Context: context.Background(),
		Users:   map[int]*tg.User{},
		Chats:   map[int]*tg.Chat{},
	}

	err := bot.Handler()(ctx, &tg.UpdateBotInlineQuery{
		QueryID: queryID,
		Query:   query,
	})
	a.NoError(err)
}
