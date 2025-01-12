package conversation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coding-hui/wecoding-sdk-go/services/ai/llms"
)

const cacheExt = ".gob"

var errInvalidID = errors.New("invalid id")

type SimpleChatHistoryStore struct {
	dir      string
	messages map[string][]llms.ChatMessage
}

func NewSimpleChatHistoryStore(dir string) *SimpleChatHistoryStore {
	return &SimpleChatHistoryStore{
		dir:      dir,
		messages: make(map[string][]llms.ChatMessage),
	}
}

// AddAIMessage adds an AIMessage to the chat message history.
func (h *SimpleChatHistoryStore) AddAIMessage(ctx context.Context, convoID, message string) error {
	return h.AddMessage(ctx, convoID, llms.AIChatMessage{Content: message})
}

// AddUserMessage adds a user to the chat message history.
func (h *SimpleChatHistoryStore) AddUserMessage(ctx context.Context, convoID, message string) error {
	return h.AddMessage(ctx, convoID, llms.HumanChatMessage{Content: message})
}

func (h *SimpleChatHistoryStore) AddMessage(_ context.Context, convoID string, message llms.ChatMessage) error {
	if err := h.load(convoID); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	h.messages[convoID] = append(h.messages[convoID], message)
	return h.persistent(convoID, h.messages[convoID])
}

func (h *SimpleChatHistoryStore) SetMessages(_ context.Context, convoID string, messages []llms.ChatMessage) error {
	h.messages[convoID] = messages
	if err := h.invalidate(convoID); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return h.persistent(convoID, h.messages[convoID])
}

func (h *SimpleChatHistoryStore) Messages(_ context.Context, convoID string) ([]llms.ChatMessage, error) {
	if err := h.load(convoID); err != nil {
		return nil, err
	}
	return h.messages[convoID], nil
}

func (h *SimpleChatHistoryStore) load(convoID string) error {
	if convoID == "" {
		return fmt.Errorf("read: %w", errInvalidID)
	}
	file, err := os.Open(filepath.Join(h.dir, convoID+cacheExt))
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	defer file.Close() //nolint:errcheck

	var rawMessages []llms.ChatMessageModel
	if err := decode(file, &rawMessages); err != nil {
		return fmt.Errorf("read: %w", err)
	}

	var messages []llms.ChatMessage
	for _, v := range rawMessages {
		messages = append(messages, v.ToChatMessage())
	}
	h.messages[convoID] = messages

	return nil
}

func (h *SimpleChatHistoryStore) persistent(convoID string, messages []llms.ChatMessage) error {
	if convoID == "" {
		return fmt.Errorf("write: %w", errInvalidID)
	}

	file, err := os.Create(filepath.Join(h.dir, convoID+cacheExt))
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	defer file.Close() //nolint:errcheck

	var rawMessages []llms.ChatMessageModel
	for _, v := range messages {
		if v != nil {
			rawMessages = append(rawMessages, llms.ConvertChatMessageToModel(v))
		}
	}
	if err := encode(file, &rawMessages); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (h *SimpleChatHistoryStore) invalidate(convoID string) error {
	if convoID == "" {
		return fmt.Errorf("delete: %w", errInvalidID)
	}
	if err := os.Remove(filepath.Join(h.dir, convoID+cacheExt)); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}
