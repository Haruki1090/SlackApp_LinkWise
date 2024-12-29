package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// SlackMessageResponse Slack APIの conversations.replies メソッドのレスポンスを格納する構造体
type SlackMessageResponse struct {
	OK               bool           `json:"ok"`
	Messages         []SlackMessage `json:"messages"`
	HasMore          bool           `json:"has_more"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
	Error string `json:"error"`
}

// SlackMessage 個々のメッセージの情報を格納する構造体
type SlackMessage struct {
	Text string `json:"text"`
	User string `json:"user"`
	Ts   string `json:"ts"`
}

// RequestPayload フロントエンドから受け取るリクエストの構造体
type RequestPayload struct {
	URL string `json:"url"`
}

// ResponsePayload フロントエンドに返すレスポンスの構造体
type ResponsePayload struct {
	Messages []ResponseData `json:"messages"`
	Error    string         `json:"error,omitempty"`
}

// ResponseData 個々のメッセージをフロントエンドに返すための構造体
type ResponseData struct {
	Timestamp string `json:"timestamp"`
	UserName  string `json:"user_name"`
	Text      string `json:"text"`
}

var slackBotToken string

func main() {
	// .env ファイルから環境変数を読み込む
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	if slackBotToken == "" {
		log.Fatal("Error: SLACK_BOT_TOKEN environment variable is not set")
	}

	// Render の環境変数 PORT を取得
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// HTTPハンドラーの設定
	mux := http.NewServeMux()
	mux.HandleFunc("/api/fetch-message", handleFetchMessage)

	// CORS設定
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // 必要に応じてフロントエンドのURLを指定
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	fmt.Printf("Go backend running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

// handleFetchMessage フロントエンドからのリクエストを処理するハンドラー
func handleFetchMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// リクエストボディをデコード
	var reqPayload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&reqPayload)
	if err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	slackURL := reqPayload.URL
	if slackURL == "" {
		http.Error(w, "Slack URL is required", http.StatusBadRequest)
		return
	}

	// Slack URLの形式を検証し、チャンネルIDとタイムスタンプを抽出
	channelID, timestamp := extractSlackLinkInfo(slackURL)
	if channelID == "" || timestamp == "" {
		http.Error(w, "Invalid Slack message URL format", http.StatusBadRequest)
		return
	}

	// スレッド内のメッセージを取得
	messages, err := getThreadMessages(channelID, timestamp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting messages: %v", err), http.StatusInternalServerError)
		return
	}

	// 取得したメッセージをタイムスタンプ順にソート
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Ts < messages[j].Ts
	})

	// レスポンス用にメッセージを整形
	var responseMessages []ResponseData
	for _, msg := range messages {
		userName, err := getUserName(msg.User)
		if err != nil {
			userName = "Unknown"
		}

		formattedTimestamp, err := formatTimestamp(msg.Ts)
		if err != nil {
			formattedTimestamp = msg.Ts
		}

		responseMessages = append(responseMessages, ResponseData{
			Timestamp: formattedTimestamp,
			UserName:  userName,
			Text:      msg.Text,
		})
	}

	// JSONレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponsePayload{
		Messages: responseMessages,
	})
}

// getUserName Slack API を使用してユーザー名を取得する関数
func getUserName(userID string) (string, error) {
	apiURL := "https://slack.com/api/users.info"
	client := &http.Client{}
	data := url.Values{}
	data.Set("user", userID)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+slackBotToken)
	req.URL.RawQuery = data.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call slack api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("slack api request failed with status: %s", resp.Status)
	}

	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		User  struct {
			Profile struct {
				RealName string `json:"real_name"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode slack api response: %w", err)
	}

	if !response.OK {
		return "", fmt.Errorf("slack api returned an error: %s", response.Error)
	}

	return response.User.Profile.RealName, nil
}

// formatTimestamp タイムスタンプを日時にフォーマットする関数
func formatTimestamp(ts string) (string, error) {
	parts := regexp.MustCompile(`\.`).Split(ts, 2)
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid timestamp format")
	}

	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %w", err)
	}

	t := time.Unix(seconds, 0)
	return t.Format("2006-01-02 15:04:05"), nil
}

// extractSlackLinkInfo Slack のメッセージ URL からチャンネル ID とタイムスタンプを抽出する関数
func extractSlackLinkInfo(link string) (string, string) {
	re := regexp.MustCompile(`https:\/\/([a-zA-Z0-9-]+)\.slack\.com\/archives\/([CG][A-Za-z0-9]+)\/p([0-9]{10})([0-9]{6})`)
	match := re.FindStringSubmatch(link)
	if len(match) == 5 {
		channelID := match[2]
		timestamp := fmt.Sprintf("%s.%s", match[3], match[4])
		return channelID, timestamp
	}
	return "", ""
}

// getThreadMessages 指定されたチャンネルと親メッセージのタイムスタンプから、スレッド内の全てのメッセージを取得する関数
func getThreadMessages(channelID, parentTimestamp string) ([]SlackMessage, error) {
	ctx := context.Background()
	client := &http.Client{}
	apiURL := "https://slack.com/api/conversations.replies"

	var allMessages []SlackMessage
	cursor := ""

	for {
		data := url.Values{}
		data.Set("channel", channelID)
		data.Set("ts", parentTimestamp)
		data.Set("inclusive", "true")
		data.Set("limit", "100")

		if cursor != "" {
			data.Set("cursor", cursor)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+slackBotToken)
		req.URL.RawQuery = data.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call slack api: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("slack api request failed with status: %s", resp.Status)
		}

		var slackResponse SlackMessageResponse
		if err := json.NewDecoder(resp.Body).Decode(&slackResponse); err != nil {
			return nil, fmt.Errorf("failed to decode slack api response: %w", err)
		}

		if !slackResponse.OK {
			return nil, fmt.Errorf("slack api returned an error: %v", slackResponse.Error)
		}

		allMessages = append(allMessages, slackResponse.Messages...)
		cursor = slackResponse.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}

	return allMessages, nil
}
