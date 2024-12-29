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
)

// SlackMessageResponse Slack APIの conversations.history メソッドのレスポンスを格納する構造体
type SlackMessageResponse struct {
	OK               bool           `json:"ok"`       // API リクエストが成功したかどうか
	Messages         []SlackMessage `json:"messages"` // メッセージのリスト
	HasMore          bool           `json:"has_more"` // まだ取得できるメッセージがあるかどうか
	PinTo            interface{}    `json:"pin_to"`   // 使用しないが、レスポンスに含まれる可能性があるので定義
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"` // 次のページのカーソル
	} `json:"response_metadata"` // ページネーション情報
}

// SlackMessage 個々のメッセージの情報を格納する構造体
type SlackMessage struct {
	ClientMsgID  string        `json:"client_msg_id"`  // クライアントが生成したメッセージ ID
	Type         string        `json:"type"`           // メッセージの種類
	Subtype      string        `json:"subtype"`        // メッセージのサブタイプ
	Text         string        `json:"text"`           // メッセージの本文
	User         string        `json:"user"`           // メッセージを投稿したユーザーの ID
	Ts           string        `json:"ts"`             // メッセージのタイムスタンプ
	ThreadTs     *string       `json:"thread_ts"`      // スレッドの親メッセージのタイムスタンプ。スレッドにない場合は nil
	ParentUserID *string       `json:"parent_user_id"` // 親メッセージを投稿したユーザーの ID。親メッセージがない場合は nil
	Team         string        `json:"team"`           // メッセージを投稿したチームの ID
	Blocks       []interface{} `json:"blocks"`         // メッセージのブロック
}

var slackBotToken string

func main() {
	// .env ファイルから環境変数を読み込む
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// コマンドライン引数から Slack メッセージ URL を取得
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <slack_message_url>")
		return
	}

	// Slack メッセージ URL を取得
	messageURL := os.Args[1]

	// Slack メッセージ URL の形式を検証する正規表現
	re := regexp.MustCompile(`https:\/\/([a-zA-Z0-9-]+)\.slack\.com\/archives\/(C[A-Za-z0-9]+)\/p([0-9]{10})([0-9]{6})`)
	if !re.MatchString(messageURL) {
		fmt.Println("Error: Invalid Slack message URL format") // 正規表現にマッチしない場合はエラー
		return
	}

	slackBotToken = os.Getenv("SLACK_BOT_TOKEN") // 環境変数から Slack Bot のトークンを取得
	if slackBotToken == "" {
		fmt.Println("Error: SLACK_BOT_TOKEN environment variable is not set") // 環境変数が設定されていない場合はエラー
		return
	}

	channelID, timestamp := extractSlackLinkInfo(messageURL) // チャンネル ID とタイムスタンプを取得
	if channelID == "" || timestamp == "" {
		fmt.Println("Error: Failed to extract channel ID and timestamp from URL") // チャンネル ID とタイムスタンプが取得できない場合はエラー
		return
	}

	messages, err := getThreadMessages(channelID, timestamp) // スレッド内の全てのメッセージを取得
	if err != nil {
		fmt.Println("Error getting messages:", err) // メッセージの取得に失敗した場合はエラー
		return
	}

	// 取得したメッセージをタイムスタンプ順にソート
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Ts < messages[j].Ts
	})

	// 取得したメッセージを表示する処理
	for _, msg := range messages {
		// ユーザー名を取得
		userName, err := getUserName(msg.User)
		if err != nil {
			fmt.Printf("Error getting username for user %s: %v\n", msg.User, err)
			continue
		}

		// タイムスタンプを日時に変換
		timestamp, err := formatTimestamp(msg.Ts)
		if err != nil {
			fmt.Printf("Error formatting timestamp %s: %v\n", msg.Ts, err)
			continue
		}

		// メッセージの表示
		fmt.Printf("[%s] %s: %s\n", timestamp, userName, msg.Text)
	}

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
		OK   bool `json:"ok"`
		User struct {
			Profile struct {
				RealName string `json:"real_name"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode slack api response: %w", err)
	}

	if !response.OK {
		return "", fmt.Errorf("slack api returned an error")
	}

	return response.User.Profile.RealName, nil
}

// formatTimestamp タイムスタンプを日時にフォーマットする関数
func formatTimestamp(ts string) (string, error) {
	// タイムスタンプは "1234567890.123456" の形式
	parts := regexp.MustCompile(`\.`).Split(ts, 2)
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid timestamp format")
	}

	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Unixタイムスタンプを日時に変換
	t := time.Unix(seconds, 0)
	return t.Format("2006-01-02 15:04:05"), nil
}

// extractSlackLinkInfo Slack のメッセージ URL からチャンネル ID とタイムスタンプを抽出する関数
func extractSlackLinkInfo(link string) (string, string) {
	re := regexp.MustCompile(`https:\/\/([a-zA-Z0-9-]+)\.slack\.com\/archives\/(C[A-Za-z0-9]+)\/p([0-9]{10})([0-9]{6})`) // 正規表現パターン
	match := re.FindStringSubmatch(link)                                                                                 // 正規表現にマッチする部分を取得
	// マッチした部分が 5 つの場合はチャンネル ID とタイムスタンプを返す
	if len(match) == 5 {
		channelID := match[2]
		timestamp := fmt.Sprintf("%s.%s", match[3], match[4])
		return channelID, timestamp
	}
	return "", ""
}

// getThreadMessages 指定されたチャンネルと親メッセージのタイムスタンプから、スレッド内の全てのメッセージを取得する関数
func getThreadMessages(channelID, parentTimestamp string) ([]SlackMessage, error) {
	ctx := context.Background()                             // コンテキストを生成
	client := &http.Client{}                                // HTTP クライアントを生成
	apiURL := "https://slack.com/api/conversations.replies" // Slack API の URL

	var allMessages []SlackMessage
	cursor := ""

	// ページネーションを考慮して全てのメッセージを取得
	for {
		data := url.Values{}            // URL クエリパラメータを格納するための map を生成
		data.Set("channel", channelID)  // チャンネル ID をセット
		data.Set("ts", parentTimestamp) // 親メッセージのタイムスタンプをセット
		data.Set("inclusive", "true")   // 親メッセージを含む
		// ページネーション情報をセット
		if cursor != "" {
			data.Set("cursor", cursor)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil) // GET リクエストを生成
		// リクエストの生成に失敗した場合はエラーを返す
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+slackBotToken)          // ヘッダにトークンをセット
		req.Header.Set("Content-Type", "application/json; charset=utf-8") // ヘッダにコンテンツタイプをセット
		req.URL.RawQuery = data.Encode()                                  // URL クエリパラメータをセット

		resp, err := client.Do(req)
		// リクエストの送信に失敗した場合はエラーを返す
		if err != nil {
			return nil, fmt.Errorf("failed to call slack api: %w", err)
		}
		defer resp.Body.Close() // レスポンスのボディを閉じる

		// レスポンスのステータスコードが 200 以外の場合はエラーを返す
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("slack api request failed with status: %s", resp.Status)
		}

		var slackResponse SlackMessageResponse // Slack API のレスポンスを格納する構造体
		// レスポンスのボディをデコードして Slack API のレスポンスを取得
		if err := json.NewDecoder(resp.Body).Decode(&slackResponse); err != nil {
			return nil, fmt.Errorf("failed to decode slack api response: %w", err)
		}

		// API リクエストが成功していない場合はエラーを返す
		if !slackResponse.OK {
			return nil, fmt.Errorf("slack api returned an error: %v", slackResponse)
		}

		// 取得したメッセージを全てのメッセージに追加
		allMessages = append(allMessages, slackResponse.Messages...)

		// 次のページがない場合はループを抜ける
		cursor = slackResponse.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}

	// 全てのメッセージを返す
	return allMessages, nil
}
