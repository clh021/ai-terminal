package conversation

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coding-hui/wecoding-sdk-go/services/ai/llms"
)

func TestStore(t *testing.T) {
	convoID := NewConversationID()

	t.Run("read non-existent", func(t *testing.T) {
		store := NewSimpleChatHistoryStore(t.TempDir())
		err := store.load("super-fake")
		require.ErrorIs(t, err, os.ErrNotExist)
		defer func() {
			_ = store.invalidate(convoID)
		}()
	})

	t.Run("set messages", func(t *testing.T) {
		ctx := context.Background()
		store := NewSimpleChatHistoryStore(t.TempDir())
		_ = store.invalidate(convoID)
		require.NoError(t, store.AddUserMessage(ctx, convoID, "hello"))
		require.NoError(t, store.AddAIMessage(ctx, convoID, "hi"))
		require.NoError(t, store.AddAIMessage(ctx, convoID, "bye"))
		messages, err := store.Messages(ctx, convoID)
		require.NoError(t, err)
		require.Equal(t, 3, len(messages))
		defer func() {
			_ = store.invalidate(convoID)
		}()
	})

	t.Run("write", func(t *testing.T) {
		store := NewSimpleChatHistoryStore(t.TempDir())
		messages := []llms.ChatMessage{
			llms.AIChatMessage{Content: "foo"},
			llms.HumanChatMessage{Content: "bar"},
			llms.AIChatMessage{Content: "zoo"},
		}
		require.NoError(t, store.persistent(convoID, messages))
		require.NoError(t, store.load(convoID))
		require.ElementsMatch(t, messages, store.messages[convoID])
		defer func() {
			_ = store.invalidate(convoID)
		}()
	})

	t.Run("delete", func(t *testing.T) {
		store := NewSimpleChatHistoryStore(t.TempDir())
		require.NoError(t, store.persistent(convoID, []llms.ChatMessage{}))
		require.NoError(t, store.invalidate(convoID))
		require.ErrorIs(t, store.load(convoID), os.ErrNotExist)
		defer func() {
			_ = store.invalidate(convoID)
		}()
	})

}
