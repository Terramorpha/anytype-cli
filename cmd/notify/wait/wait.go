package wait

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

const previewsSubId = "anytype-cli-notify"

func NewWaitCmd() *cobra.Command {
	var statePath string
	var once bool

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Block until a new message/notification, then drain all unseen ones",
		Long: "Blocks on the server event stream (no polling). On each wake it drains\n" +
			"every unseen message and notification (level-triggered), printing one per\n" +
			"line, and records their ids in a viewed-set so none are dropped or repeated.\n\n" +
			"With --once, exits after the first non-empty drain (suitable for scripting a\n" +
			"wake/handle loop); otherwise streams notifications until interrupted.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if statePath == "" {
				home, _ := os.UserHomeDir()
				statePath = filepath.Join(home, ".anytype", "notify-seen")
			}
			seen, err := loadSeen(statePath)
			if err != nil {
				return output.Error("failed to read state: %w", err)
			}

			token, _, err := core.GetStoredSessionToken()
			if err != nil {
				return output.Error("not authenticated: %w", err)
			}

			// Start the event stream (the level-triggered wake) and register the
			// global message-previews subscription so all chats push onto it.
			er, err := core.ListenForEvents(token)
			if err != nil {
				return output.Error("failed to start event stream: %w", err)
			}
			if err := core.SubscribeMessagePreviews(previewsSubId); err != nil {
				return output.Error("%w", err)
			}
			// Push wake for page mentions: subscribe to my member object so a
			// backlinks change emits an ObjectDetailsAmend on the same stream.
			if err := core.SubscribeMemberObjects(previewsSubId + "-member"); err != nil {
				output.Warning("page-mention subscription unavailable: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() { <-sig; cancel() }()

			isWake := func(m *pb.EventMessage) bool {
				return m.GetChatAdd() != nil ||
					m.GetNotificationSend() != nil ||
					m.GetObjectDetailsAmend() != nil ||
					m.GetObjectDetailsSet() != nil
			}

			// Drain authoritative state until a pass yields nothing new, recording
			// ids only after they are emitted. Returns how many were surfaced.
			drain := func() (int, error) {
				total := 0
				for {
					items, err := core.DrainNotifications(seen)
					if err != nil {
						return total, err
					}
					if len(items) == 0 {
						return total, nil
					}
					for _, it := range items {
						printItem(it)
						seen[it.Id] = true
					}
					if err := appendSeen(statePath, items); err != nil {
						return total, err
					}
					total += len(items)
				}
			}

			for {
				n, err := drain()
				if err != nil {
					output.Warning("drain error: %v", err)
				}
				if once && n > 0 {
					return nil
				}
				if _, err := er.WaitForEvent(ctx, isWake); err != nil {
					return nil // context cancelled / stream closed
				}
			}
		},
	}

	cmd.Flags().StringVar(&statePath, "state", "", "viewed-set file (default ~/.anytype/notify-seen)")
	cmd.Flags().BoolVar(&once, "once", false, "exit after the first non-empty drain")
	return cmd
}

func printItem(it core.NotifItem) {
	if it.Kind == "notification" {
		output.Print("[notification id=%s] %s", it.Id, it.Text)
		return
	}
	if it.Kind == "page-mention" {
		output.Print("[page-mention space=%s object=%s (%s)] you were mentioned",
			it.SpaceId, it.Id, it.ChatName)
		return
	}
	tag := fmt.Sprintf("space=%s chat=%s (%s)", it.SpaceId, it.ChatId, it.ChatName)
	if it.HasMention {
		output.Print("[@mention %s] %s: %s", tag, it.CreatorName, it.Text)
	} else {
		output.Print("[%s] %s: %s", tag, it.CreatorName, it.Text)
	}
}

func loadSeen(path string) (map[string]bool, error) {
	seen := map[string]bool{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return seen, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			seen[line] = true
		}
	}
	return seen, sc.Err()
}

func appendSeen(path string, items []core.NotifItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, it := range items {
		if _, err := fmt.Fprintln(f, it.Id); err != nil {
			return err
		}
	}
	return nil
}
