package bridge

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wy51ai/moltbotCNAPP/internal/clawdbot"
	"github.com/wy51ai/moltbotCNAPP/internal/feishu"
)

// Bridge connects Feishu and ClawdBot
type Bridge struct {
	feishuClient   *feishu.Client
	clawdbotClient *clawdbot.Client
	thinkingMs     int
	sessionKey     string
	seenMessages   *messageCache
}

// messageCache stores seen message IDs to prevent duplicate processing
type messageCache struct {
	cache map[string]time.Time
	mu    sync.RWMutex
	ttl   time.Duration
}

func newMessageCache(ttl time.Duration) *messageCache {
	mc := &messageCache{
		cache: make(map[string]time.Time),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go mc.cleanup()

	return mc
}

func (mc *messageCache) has(messageID string) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	_, exists := mc.cache[messageID]
	return exists
}

func (mc *messageCache) add(messageID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.cache[messageID] = time.Now()
}

func (mc *messageCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		mc.mu.Lock()
		now := time.Now()
		for id, timestamp := range mc.cache {
			if now.Sub(timestamp) > mc.ttl {
				delete(mc.cache, id)
			}
		}
		mc.mu.Unlock()
	}
}

// NewBridge creates a new bridge
func NewBridge(feishuClient *feishu.Client, clawdbotClient *clawdbot.Client, thinkingMs int, sessionKey string) *Bridge {
	return &Bridge{
		feishuClient:   feishuClient,
		clawdbotClient: clawdbotClient,
		thinkingMs:     thinkingMs,
		sessionKey:     sessionKey,
		seenMessages:   newMessageCache(10 * time.Minute),
	}
}

// SetFeishuClient sets the Feishu client after construction
func (b *Bridge) SetFeishuClient(client *feishu.Client) {
	b.feishuClient = client
}

// HandleMessage processes a message from Feishu
func (b *Bridge) HandleMessage(msg *feishu.Message) error {
	// Check for duplicates
	if msg.MessageID != "" && b.seenMessages.has(msg.MessageID) {
		log.Printf("[Bridge] Skipping duplicate message: %s", msg.MessageID)
		return nil
	}

	// Mark as seen
	if msg.MessageID != "" {
		b.seenMessages.add(msg.MessageID)
	}

	// Clean up message text
	text := msg.Content
	text = removeMentions(text)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	// For group chats, check if we should respond
	if msg.ChatType == "group" {
		if !shouldRespondInGroup(text, msg.Mentions) {
			log.Printf("[Bridge] Skipping group message (no trigger): %s", text)
			return nil
		}
	}

	log.Printf("[Bridge] Processing message from %s: %s", msg.ChatID, text)

	// Process asynchronously
	go b.processMessage(msg.ChatID, text)

	return nil
}

func (b *Bridge) processMessage(chatID, text string) {
	var placeholderID string
	var responseMessageID string
	var done bool
	var thinkingDots int
	var mu sync.Mutex

	// Dynamic thinking animation ticker
	var thinkingTicker *time.Ticker
	var thinkingStop chan bool

	// Show "thinking..." if response takes too long
	var timer *time.Timer
	if b.thinkingMs > 0 {
		timer = time.AfterFunc(time.Duration(b.thinkingMs)*time.Millisecond, func() {
			mu.Lock()
			defer mu.Unlock()

			if done {
				return
			}

			// Send initial thinking message
			msgID, err := b.feishuClient.SendMessage(chatID, "正在思考.")
			if err != nil {
				log.Printf("[Bridge] Failed to send thinking message: %v", err)
				return
			}
			placeholderID = msgID
			thinkingDots = 1

			// Start thinking animation
			thinkingStop = make(chan bool)
			thinkingTicker = time.NewTicker(500 * time.Millisecond)
			go func() {
				for {
					select {
					case <-thinkingTicker.C:
						mu.Lock()
						if done || placeholderID == "" {
							mu.Unlock()
							return
						}
						
						// Cycle through 1, 2, 3 dots
						thinkingDots = (thinkingDots % 3) + 1
						dots := strings.Repeat(".", thinkingDots)
						thinkingText := "正在思考" + dots
						
						if err := b.feishuClient.UpdateMessage(placeholderID, thinkingText); err != nil {
							log.Printf("[Bridge] Failed to update thinking animation: %v", err)
						}
						mu.Unlock()
					case <-thinkingStop:
						return
					}
				}
			}()
		})
	}

	// Stream buffer for accumulating response
	var streamBuffer strings.Builder
	var lastUpdateTime time.Time
	updateInterval := 300 * time.Millisecond // Update every 300ms

	// Progress callback for streaming
	onProgress := func(stream, data string) {
		if stream != "assistant" {
			return
		}

		mu.Lock()
		defer mu.Unlock()

		if done {
			return
		}

		// Parse stream data
		var streamData struct {
			Text  string `json:"text,omitempty"`
			Delta string `json:"delta,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &streamData); err != nil {
			log.Printf("[Bridge] Failed to parse stream data: %v", err)
			return
		}

		// Accumulate text or delta
		if streamData.Text != "" {
			streamBuffer.Reset()
			streamBuffer.WriteString(streamData.Text)
		} else if streamData.Delta != "" {
			streamBuffer.WriteString(streamData.Delta)
		} else {
			return // No text or delta, skip
		}

		currentText := streamBuffer.String()
		if currentText == "" {
			return
		}

		// First chunk - delete thinking message and create response message
		if responseMessageID == "" {
			// Stop thinking animation
			if thinkingTicker != nil {
				thinkingTicker.Stop()
				if thinkingStop != nil {
					close(thinkingStop)
				}
			}

			// Delete thinking placeholder
			if placeholderID != "" {
				if err := b.feishuClient.DeleteMessage(placeholderID); err != nil {
					log.Printf("[Bridge] Failed to delete thinking placeholder: %v", err)
				}
				placeholderID = ""
			}

			// Create new response message with first chunk
			msgID, err := b.feishuClient.SendMessage(chatID, currentText)
			if err != nil {
				log.Printf("[Bridge] Failed to create response message: %v", err)
				return
			}
			responseMessageID = msgID
			lastUpdateTime = time.Now()
			return
		}

		// Throttle updates to avoid rate limiting
		if time.Since(lastUpdateTime) < updateInterval {
			return
		}

		// Update existing message with accumulated content
		if err := b.feishuClient.UpdateMessage(responseMessageID, currentText); err != nil {
			log.Printf("[Bridge] Failed to update streaming message: %v", err)
		} else {
			lastUpdateTime = time.Now()
		}
	}

	// Ask ClawdBot with streaming
	sessionKey := b.sessionKey
	if sessionKey == "" {
		sessionKey = fmt.Sprintf("feishu:%s", chatID)
	}
	log.Printf("[Bridge] sessionKey: %s", sessionKey)
	
	reply, err := b.clawdbotClient.AskClawdbot(text, sessionKey, onProgress)
	log.Printf("[Bridge] reply: %s", reply)
	
	// Mark as done
	mu.Lock()
	done = true
	
	// Stop thinking animation
	if thinkingTicker != nil {
		thinkingTicker.Stop()
		if thinkingStop != nil {
			close(thinkingStop)
		}
	}
	mu.Unlock()

	if timer != nil {
		timer.Stop()
	}

	if err != nil {
		reply = fmt.Sprintf("（系统出错）%v", err)
		log.Printf("[Bridge] Error from ClawdBot: %v", err)
	}

	// Clean up reply
	reply = strings.TrimSpace(reply)
	log.Printf("[Bridge] ClawdBot raw reply: %q", reply)

	// Check for NO_REPLY
	if reply == "" || reply == "NO_REPLY" {
		log.Printf("[Bridge] Received NO_REPLY, not sending message")

		mu.Lock()
		// Delete thinking placeholder if it exists
		if placeholderID != "" {
			if err := b.feishuClient.DeleteMessage(placeholderID); err != nil {
				log.Printf("[Bridge] Failed to delete placeholder: %v", err)
			}
		}
		// Delete response message if it exists
		if responseMessageID != "" {
			if err := b.feishuClient.DeleteMessage(responseMessageID); err != nil {
				log.Printf("[Bridge] Failed to delete response message: %v", err)
			}
		}
		mu.Unlock()
		return
	}

	mu.Lock()
	currentPlaceholder := placeholderID
	currentResponse := responseMessageID
	mu.Unlock()

	// If we have a response message (from streaming), do final update
	if currentResponse != "" {
		if err := b.feishuClient.UpdateMessage(currentResponse, reply); err != nil {
			log.Printf("[Bridge] Failed to final update message: %v", err)
		} else {
			log.Printf("[Bridge] Final updated message in %s", chatID)
		}
	} else if currentPlaceholder != "" {
		// No streaming happened, delete placeholder and send new message
		if err := b.feishuClient.DeleteMessage(currentPlaceholder); err != nil {
			log.Printf("[Bridge] Failed to delete placeholder: %v", err)
		}
		
		if _, err := b.feishuClient.SendMessage(chatID, reply); err != nil {
			log.Printf("[Bridge] Failed to send message: %v", err)
		} else {
			log.Printf("[Bridge] Sent new message to %s", chatID)
		}
	} else {
		// No placeholder, send new message
		if _, err := b.feishuClient.SendMessage(chatID, reply); err != nil {
			log.Printf("[Bridge] Failed to send message: %v", err)
		} else {
			log.Printf("[Bridge] Sent message to %s", chatID)
		}
	}
}

// shouldRespondInGroup determines if the bot should respond in a group chat
func shouldRespondInGroup(text string, mentions []feishu.Mention) bool {
	// Always respond if mentioned
	if len(mentions) > 0 {
		return true
	}

	lowerText := strings.ToLower(text)

	// Question marks
	if strings.HasSuffix(text, "?") || strings.HasSuffix(text, "？") {
		return true
	}

	// English question words
	questionWords := []string{"why", "how", "what", "when", "where", "who", "help"}
	for _, word := range questionWords {
		if regexp.MustCompile(`\b` + word + `\b`).MatchString(lowerText) {
			return true
		}
	}

	// Chinese action verbs
	actionVerbs := []string{
		"帮", "麻烦", "请", "能否", "可以", "解释", "看看",
		"排查", "分析", "总结", "写", "改", "修", "查", "对比", "翻译",
	}
	for _, verb := range actionVerbs {
		if strings.Contains(text, verb) {
			return true
		}
	}

	// Bot names/triggers
	botTriggers := []string{"alen", "clawdbot", "bot", "助手", "智能体"}
	for _, trigger := range botTriggers {
		pattern := fmt.Sprintf(`^%s[\s,:，：]`, trigger)
		if matched, _ := regexp.MatchString(pattern, lowerText); matched {
			return true
		}
	}

	return false
}

// removeMentions removes @mention patterns from text
func removeMentions(text string) string {
	re := regexp.MustCompile(`@_user_\d+\s*`)
	return re.ReplaceAllString(text, "")
}
