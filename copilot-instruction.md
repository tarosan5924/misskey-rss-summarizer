# Copilot Instructions for misskeyRSSbot

## プロジェクト概要

このプロジェクトは、RSSフィードを取得してMisskeyに自動投稿するGoアプリケーションです。
クリーンアーキテクチャを採用し、domain/application/infrastructure/interfacesの4層構造で実装されています。

## アーキテクチャ

### ディレクトリ構成

- `internal/domain/entity/`: ビジネスロジックを含むエンティティ
- `internal/domain/repository/`: リポジトリインターフェース定義
- `internal/application/`: アプリケーションサービス層
- `internal/infrastructure/`: 外部サービスとの接続実装（Misskey API、RSS取得、ストレージ）
- `internal/interfaces/config/`: 設定管理

### 依存関係ルール

- domain層は他のどの層にも依存しない
- application層はdomain層のみに依存する
- infrastructure層とinterfaces層はdomain層に依存する
- main.goですべての依存関係を注入する

## コーディング規約

### 基本方針

- コード内にコメントは記載しない（関数名や変数名で意図を明確にする）
  - 例外：公開API（exportされた関数・型・メソッド）のGoDocは省略可（プロジェクト方針により任意）
  - パッケージレベルのdoc.goは不要
- シンプルで読みやすいコードを優先する
- エラーハンドリングは必ず実装する
- Go標準のコーディングスタイルに従う（gofmt, golint, go vet準拠）

### 命名規則

- パッケージ名：小文字のみ（`misskey`, `storage`）
- エクスポートする識別子：PascalCase（`FeedEntry`, `NewNoteRepository`）
- プライベート識別子：camelCase（`rateLimiter`, `loadRSSURLs`）
- インターフェース名：名詞形（`FeedRepository`, `NoteRepository`）
- コンストラクタ：`New<TypeName>`の形式（`NewFeedEntry`, `NewRSSFeedService`）

### エラーハンドリング

- エラーは`fmt.Errorf`で詳細なコンテキストを付与してラップする
- `%w`を使用してエラーチェーンを保持する
- ログ出力は`log.Printf`を使用する
- 致命的なエラーは`log.Fatal`で終了する

### テストコード

- テストファイル名：`<ファイル名>_test.go`
- テスト関数名：`Test<関数名>`または`Test<型名>_<メソッド名>`
- テーブル駆動テストを使用する
- テストケースには`name`フィールドで説明を記載する

### 構造体とメソッド

- コンストラクタ関数で構造体を初期化する
- フィールドはプライベートにし、必要に応じてゲッターメソッドを提供する
- レシーバー名は1〜2文字の短縮形を使用する（`s *RSSFeedService`, `rl *rateLimiter`）

### 依存性注入

- すべての依存関係はコンストラクタで注入する
- インターフェースを使用して疎結合を実現する
- `main.go`で具象型を生成し、依存関係を組み立てる

### 並行処理

- `context.Context`を第一引数として受け取る
- `sync.Mutex`でクリティカルセクションを保護する
- ゴルーチン起動時は終了処理を明確にする

### 設定管理

- 環境変数で設定を管理する（`envconfig`パッケージ使用）
- `.env`ファイルのロードは`godotenv`を使用する
- デフォルト値を適切に設定する
- 設定値の変換用メソッドを提供する（例：`GetFetchInterval()`）

## レビュー時の観点

### [critical] 必須修正事項

- セキュリティ上の問題（認証情報のハードコード、インジェクション脆弱性、安全でない乱数生成）
- 致命的なバグ（nil参照、データ競合、デッドロック、リソースリーク、ゴルーチンリーク）
- アーキテクチャ違反（依存関係の逆転、層の責務違反）
- context.Contextの不適切な使用（context.Background()の過度な使用、contextの保存）

### [important] 重要な改善提案

- エラーハンドリングの不足や不適切な処理（エラー無視、エラーメッセージの情報不足）
- テストカバレッジの不足（特にエッジケース、エラーパス）
- パフォーマンスに影響する問題（N+1問題、不要なアロケーション、非効率なループ）
- 保守性を著しく低下させる実装（過度な複雑さ、責務の不明確さ）
- defer、close、cancelの適切な使用漏れ
- インターフェースの不適切な定義（大きすぎる、使われていない）

### [nitpick] 軽微な改善提案

- 命名規則の統一（Goの慣用的な命名への準拠）
- 冗長なコードの削減（不要な変数、重複ロジック）
- より適切な標準ライブラリの使用（strings.Builder、sync.Pool等）
- コードの可読性向上（早期リターン、ガード節の活用）
- Goの慣用句への準拠（errors.Is/As、型アサーションのcomma-ok、スライスの事前確保）

### [question] 確認事項

- 仕様が不明確な実装
- 設計意図の確認
- 代替案の提案

### [👀great] 称賛すべき実装

- 優れた設計パターンの適用
- 保守性・拡張性の高い実装
- パフォーマンスとリソース管理の最適化
- エレガントで読みやすいコード
- プロジェクトのベストプラクティスの体現

## 称賛すべき実装例


### クリーンアーキテクチャの忠実な実装

依存関係が一方向に保たれ、各層の責務が明確に分離されています。特に、domain層が完全に独立しており、ビジネスロジックが外部依存から保護されている点が優れています。

```go
type RSSFeedService struct {
	feedRepo  repository.FeedRepository
	noteRepo  repository.NoteRepository
	cacheRepo repository.CacheRepository
}
```

### レートリミッターの実装

並行処理の安全性を保ちながら、効率的なリソース管理を実現しています。`sync.Mutex`による排他制御と、`context.Context`によるキャンセル対応が適切に実装されています。

```go
func (rl *rateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	permitsToAdd := int(elapsed / rl.refillRate)
	if permitsToAdd > 0 {
		rl.permits = min(rl.permits+permitsToAdd, rl.maxPermits)
		rl.lastRefill = now
	}
	
	if rl.permits <= 0 {
		waitTime := rl.refillRate - (now.Sub(rl.lastRefill) % rl.refillRate)
		rl.mu.Unlock()
		
		timer := time.NewTimer(waitTime)
		defer timer.Stop()
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			rl.mu.Lock()
			rl.permits = 1
			rl.lastRefill = time.Now()
			rl.permits--
			rl.mu.Unlock()
			return nil
		}
	}
	
	rl.permits--
	rl.mu.Unlock()
	return nil
}
```

### グレースフルシャットダウン

シグナルハンドリングとcontextによる適切な終了処理が実装されています。リソースのクリーンアップが保証されています。

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

go func() {
	<-sigCh
	log.Println("Shutdown signal received")
	cancel()
}()

for {
	select {
	case <-ctx.Done():
		log.Println("Shutting down...")
		return
	case <-ticker.C:
		// 処理
	}
}
```

### 初回実行時の特別処理

初回起動時にスパムを防ぐため、最新のエントリのみを投稿する配慮がされています。ユーザー体験への配慮が見られます。

```go
isFirstRun := latestPublished.IsZero()

if isFirstRun {
	var mostRecent *entity.FeedEntry
	for _, entry := range entries {
		if mostRecent == nil || entry.Published.After(mostRecent.Published) {
			mostRecent = entry
		}
	}
	if mostRecent != nil {
		newEntries = append(newEntries, mostRecent)
	}
}
```

### テーブル駆動テスト

保守性が高く、拡張しやすいテストパターンが使用されています。

```go
tests := []struct {
	name      string
	published time.Time
	compare   time.Time
	expected  bool
}{
	{
		name:      "newer than",
		published: baseTime.Add(1 * time.Hour),
		compare:   baseTime,
		expected:  true,
	},
	// ...
}

for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		// テスト実行
	})
}
```

### 環境変数の柔軟な管理

単一URLだけでなく、複数URLを番号付きで管理できる拡張性の高い設計です。

```go
func loadRSSURLs() []string {
	var urls []string
	
	for i := 1; ; i++ {
		key := fmt.Sprintf("RSS_URL_%d", i)
		url := os.Getenv(key)
		if url == "" {
			break
		}
		urls = append(urls, url)
	}
	
	return urls
}
```

### 一貫したエラーハンドリング

すべてのエラーに適切なコンテキストが付与され、エラーチェーンが保持されています。

```go
entries, err := s.feedRepo.Fetch(ctx, rssURL)
if err != nil {
	return fmt.Errorf("failed to fetch RSS feed [%s]: %w", rssURL, err)
}
```

## 具体的な実装パターン

### エンティティの実装例

```go
type FeedEntry struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	GUID        string
}

func NewFeedEntry(title, link, description string, published time.Time, guid string) *FeedEntry {
	return &FeedEntry{
		Title:       title,
		Link:        link,
		Description: description,
		Published:   published,
		GUID:        guid,
	}
}

func (f *FeedEntry) IsNewerThan(t time.Time) bool {
	return f.Published.After(t)
}
```

### リポジトリインターフェースの定義例

```go
type FeedRepository interface {
	Fetch(ctx context.Context, url string) ([]*entity.FeedEntry, error)
}
```

### サービス層の実装例

```go
type RSSFeedService struct {
	feedRepo  repository.FeedRepository
	noteRepo  repository.NoteRepository
	cacheRepo repository.CacheRepository
}

func NewRSSFeedService(
	feedRepo repository.FeedRepository,
	noteRepo repository.NoteRepository,
	cacheRepo repository.CacheRepository,
) *RSSFeedService {
	return &RSSFeedService{
		feedRepo:  feedRepo,
		noteRepo:  noteRepo,
		cacheRepo: cacheRepo,
	}
}
```

### エラーハンドリングの実装例

```go
entries, err := s.feedRepo.Fetch(ctx, rssURL)
if err != nil {
	return fmt.Errorf("failed to fetch RSS feed [%s]: %w", rssURL, err)
}
```

## 見逃してはいけないGoのベストプラクティス

### 必ずチェックすべき項目

- エラー処理: すべてのエラーを適切に処理（無視していないか、`_`で捨てていないか）
- リソース管理: `defer`による確実なクリーンアップ（ファイル、接続、ロック等）
- 並行処理の安全性: データ競合の可能性、適切な同期機構の使用
- context伝播: context.Contextが適切に伝播されているか
- nil チェック: ポインタやインターフェースのnil参照の可能性
- スライス操作: インデックス範囲外アクセス、容量の事前確保
- 型アサーション: comma-okイディオムの使用
- ゴルーチン: リークの可能性、適切な終了処理

### プロジェクト固有の制約

プロジェクト方針を尊重しつつ、以下は指摘すべきではない：

- 公開APIへのGoDocコメントがないこと（プロジェクト方針により省略可）
- 実装内の説明コメントがないこと（意図的な設計）

ただし、以下はプロジェクト方針に反するため指摘すべき：

- `panic`の使用（エラーは`error`として返す）
- グローバル変数の使用（依存性注入を使用する）
- `init()`関数の使用（明示的な初期化を優先する）
- 実装コード内への説明的コメントの追加

## レビューコメントの記載方針

- すべて日本語で記載する
- 適切なタグを付与する（`[critical]`, `[important]`, `[nitpick]`, `[question]`, `[👀great]`）
- 具体的な修正案を提示する
- なぜその修正が必要かを簡潔に説明する

### プロジェクト方針とベストプラクティスのバランス

- プロジェクト固有の制約（コメントなし等）は尊重する
- ただし、セキュリティやバグに関するGoの標準的なベストプラクティスは必ず指摘する
- `[question]`タグで設計意図を確認しつつ、改善の余地があれば提案する
- 優れた実装には`[👀great]`で積極的に称賛し、プロジェクト内で共有すべきパターンを明示する
