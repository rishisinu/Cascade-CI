package testdata

import (
	"context"
	"database/sql"
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
)

func runSuggestedCommand(ctx context.Context, client anthropic.Client, task string) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-opus-5-5",
		MaxTokens: 1024,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(task))},
	})
	if err != nil {
		return err
	}
	cmd := msg.Content[0].Text
	// ruleid: agentic-go-llm-output-to-dangerous-sink
	exec.CommandContext(ctx, "sh", "-c", cmd).Run()
	// ok: agentic-go-llm-output-to-dangerous-sink
	return exec.CommandContext(ctx, "git", "status").Run()
}

func runSuggestedQuery(ctx context.Context, client anthropic.Client, db *sql.DB, question string) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-opus-5-5",
		MaxTokens: 1024,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(question))},
	})
	if err != nil {
		return err
	}
	query := msg.Content[0].Text
	// ruleid: agentic-go-llm-output-to-dangerous-sink
	db.QueryContext(ctx, query)
	// ok: agentic-go-llm-output-to-dangerous-sink
	_, err = db.QueryContext(ctx, "SELECT name FROM products WHERE sku = ?", query)
	return err
}
