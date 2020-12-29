package bot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand"
	"unicode/utf8"

	"github.com/gotd/td/tg"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/tdakkota/porftgbot/runner"
)

type Bot struct {
	runner runner.NetRunner
	tg     *tg.Client
	logger *zap.Logger

	// immutable
	minLength int
	maxLength int
}

func NewBot(runner runner.NetRunner, raw *tg.Client, logger *zap.Logger) *Bot {
	return &Bot{
		runner:    runner,
		tg:        raw,
		logger:    logger,
		minLength: 10,
		maxLength: 100,
	}
}

var ErrResultIsEmpty = errors.New("result is empty")

func (b *Bot) randLength() int {
	return mathrand.Intn(b.maxLength-b.minLength) + b.minLength
}

func (b *Bot) query(ctx context.Context, text string, length int) (s string, err error) {
	if length == 0 {
		length = b.randLength()
	}

	defer func() {
		b.logger.With(
			zap.String("text", text),
			zap.Int("length", length),
			zap.String("answer", s),
			zap.Error(err),
		).Debug("queried server")
	}()

	r, err := b.runner.Query(ctx, runner.Query{
		Prompt: text,
		Length: length,
	})
	if err != nil {
		return "", fmt.Errorf("failed to query neural network: %w", err)
	}
	if len(r.Replies) < 1 {
		return "", fmt.Errorf("failed to query neural network: %w", ErrResultIsEmpty)
	}

	return r.Replies[0], nil
}

func (b *Bot) sendAnswer(ctx tg.UpdateContext, u *tg.UpdateBotInlineQuery) error {
	answer, err := b.query(ctx, u.Query, 0)
	if err != nil {
		return xerrors.Errorf("query net: %w", err)
	}
	length := utf8.RuneCountInString(answer)
	answer = u.Query + answer

	message := &tg.InputBotInlineMessageText{
		NoWebpage: true,
		Message:   answer,
	}
	message.SetEntities([]tg.MessageEntityClass{
		&tg.MessageEntityBold{
			Offset: utf8.RuneCountInString(u.Query),
			Length: length,
		},
	})

	id := sha256.Sum256([]byte(u.Query))
	result := &tg.InputBotInlineResult{
		ID:          hex.EncodeToString(id[:]),
		Type:        "article",
		SendMessage: message,
	}
	result.SetTitle("Результат:")
	result.SetDescription(answer)

	req := &tg.MessagesSetInlineBotResultsRequest{
		QueryID: u.QueryID,
		Results: []tg.InputBotInlineResultClass{
			result,
		},
	}

	_, err = b.tg.MessagesSetInlineBotResults(ctx, req)

	return err
}

func (b *Bot) Handler() func(ctx tg.UpdateContext, u *tg.UpdateBotInlineQuery) error {
	return func(ctx tg.UpdateContext, u *tg.UpdateBotInlineQuery) (err error) {
		b.logger.With(
			zap.Int("user_id", u.UserID),
			zap.String("query", u.Query),
		).Info("Inline query")

		if u.Query == "" {
			return nil
		}

		return b.sendAnswer(ctx, u)
	}
}
