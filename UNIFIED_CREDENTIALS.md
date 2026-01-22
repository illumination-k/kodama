# Unified Credentials Management

## 概要

Kodamaは、GitHub PAT、Claude Code認証、クラウドプロバイダーの認証情報など、**すべての認証情報を.envファイルで統一管理**できます。

## 基本的なアプローチ

```bash
# すべての認証情報を.envファイルに記載
cat > .env <<EOF
GITHUB_TOKEN=ghp_xxx
CLAUDE_CODE_AUTH_TOKEN=sk-ant-xxx
DATABASE_URL=postgresql://...
EOF

# .kodama.yamlで設定
cat > .kodama.yaml <<EOF
env:
  dotenvFiles:
    - .env
    - .env.local
EOF

# セッション開始（自動的にすべての認証情報が利用可能）
kubectl kodama start dev --sync .
```

## 利点

✅ **シンプル** - すべての認証情報を1つの.envファイルで管理
✅ **統一** - GitHub、Claude、DB、APIキーなどすべて同じ方法
✅ **移植性** - .envファイルをコピーするだけで別の環境でも動作
✅ **セキュア** - セッション終了時に自動的にシークレットが削除される
✅ **標準** - 業界標準の.envファイル形式を使用
✅ **開発者フレンドリー** - 慣れ親しんだワークフロー

## 基本的な使い方

### 1. .envファイルを作成

```bash
# .env
GITHUB_TOKEN=ghp_your_github_personal_access_token
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_claude_api_key
DATABASE_URL=postgresql://localhost:5432/mydb
API_KEY=your_api_key
```

### 2. .kodama.yamlで設定

```yaml
# .kodama.yaml
env:
  dotenvFiles:
    - .env
    - .env.local
```

### 3. .gitignoreに追加（重要！）

```bash
echo ".env" >> .gitignore
echo ".env.*" >> .gitignore
echo "!.env.example" >> .gitignore
```

### 4. セッション開始

```bash
kubectl kodama start dev --sync .
```

GitHubからのクローン、Claude Code認証、すべて自動的に動作します！

## 対応している認証情報

### Git認証

- `GITHUB_TOKEN` - GitHub Personal Access Token
- `GH_TOKEN` - 代替のGitHub token変数

### Claude Code認証

- `CLAUDE_CODE_AUTH_TOKEN` - Claude APIキー
- `ANTHROPIC_API_KEY` - 代替のAnthropic APIキー変数

### クラウドプロバイダー

- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` - AWS認証情報
- `GOOGLE_APPLICATION_CREDENTIALS` - Google Cloud認証情報
- `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET` - Azure認証情報

### その他

その他の環境変数も自由に設定可能:

- データベースURL
- APIキー
- サービスエンドポイント
- アプリケーション設定

## セッションテンプレートサポート

リポジトリルートに`.kodama.yaml`を配置することで、自動的に.envファイルを読み込みます:

```yaml
# .kodama.yaml
env:
  dotenvFiles:
    - .env
    - .env.local
  excludeVars:
    - DEBUG_MODE  # 除外する変数（オプション）

resources:
  cpu: "2"
  memory: "4Gi"

sync:
  useGitignore: true
  exclude:
    - .env
    - .env.*
```

セッション開始時、自動的に設定が適用されます:

```bash
cd /path/to/repo  # .kodama.yamlがあるディレクトリ
kubectl kodama start dev --sync .
# ✅ .envファイルが自動的に読み込まれる
```

## 環境別の管理

異なる環境で異なる認証情報を使用:

```bash
# 開発環境
.env.development:
  GITHUB_TOKEN=ghp_dev_token
  CLAUDE_CODE_AUTH_TOKEN=sk-ant-dev-key
  DATABASE_URL=postgresql://localhost/dev

# 本番環境
.env.production:
  GITHUB_TOKEN=ghp_prod_token
  CLAUDE_CODE_AUTH_TOKEN=sk-ant-prod-key
  DATABASE_URL=postgresql://prod-db/prod
```

環境を指定して起動:

```bash
# 開発
kubectl kodama start dev --env-file .env.development --sync .

# 本番
kubectl kodama start prod --env-file .env.production --sync .
```

## CLIオーバーライド

テンプレート設定をCLIフラグで上書き:

```bash
# 異なる.envファイルを使用
kubectl kodama start dev --env-file .env.custom --sync .

# 複数ファイル（後のファイルが優先）
kubectl kodama start dev \
  --env-file .env \
  --env-file .env.local \
  --sync .

# 特定の変数を除外
kubectl kodama start dev \
  --env-exclude VERBOSE \
  --env-exclude DEBUG_MODE \
  --sync .
```

## セキュリティ

### 保護されている変数

以下のシステムクリティカルな変数は**常に除外**されます:

- システム変数: `PATH`, `HOME`, `USER`, `SHELL`, `TERM`, `PWD`
- Kubernetes変数: `KUBERNETES_SERVICE_*`, `KUBERNETES_PORT_*`

### 認証情報は除外されない

以下の認証情報変数は.envファイルから**読み込み可能**です:

- `GITHUB_TOKEN`, `GH_TOKEN`
- `CLAUDE_CODE_AUTH_TOKEN`, `ANTHROPIC_API_KEY`
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- その他すべての認証情報

### ベストプラクティス

1. **.envファイルをバージョン管理にコミットしない**
   ```bash
   # .gitignore
   .env
   .env.*
   !.env.example
   ```

2. **.env.exampleをドキュメントとして使用**
   ```bash
   # .env.example（gitにコミット）
   GITHUB_TOKEN=ghp_your_token_here
   CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_key_here

   # .env（gitにコミットしない - 実際の値）
   GITHUB_TOKEN=ghp_actual_secret_token
   CLAUDE_CODE_AUTH_TOKEN=sk-ant-actual_secret_key
   ```

3. **定期的に認証情報をローテーション**

4. **環境別にファイルを分ける**

## 動作の仕組み

1. **dotenvファイル読み込み** - ローカルマシンから.envを読み込み
2. **Kubernetesシークレット作成** - `kodama-env-{session-name}`という名前でシークレットを作成
3. **Podに注入** - `envFrom`を使用してすべての環境変数を注入
4. **認証が自動的に動作** - Git操作とClaude Code認証が自動的に機能
5. **自動クリーンアップ** - セッション削除時にシークレットも自動削除

## 完全な例

```bash
# 1. リポジトリのセットアップ
cd /path/to/project

# 2. .envファイル作成
cat > .env <<EOF
GITHUB_TOKEN=ghp_your_github_token
CLAUDE_CODE_AUTH_TOKEN=sk-ant-your_claude_token
DATABASE_URL=postgresql://localhost:5432/mydb
API_KEY=your_api_key
EOF

# 3. .kodama.yaml作成
cat > .kodama.yaml <<EOF
env:
  dotenvFiles:
    - .env
    - .env.local
EOF

# 4. .gitignoreに追加
cat >> .gitignore <<EOF
.env
.env.*
!.env.example
EOF

# 5. セッション開始（すべての認証情報が自動利用可能）
kubectl kodama start dev --repo https://github.com/myorg/private-repo --sync .
# ✅ GitHubからプライベートリポジトリをクローン（GITHUB_TOKEN使用）
# ✅ Claude Code認証済み（CLAUDE_CODE_AUTH_TOKEN使用）
# ✅ すべての環境変数がPodで利用可能

# 6. 検証
kubectl kodama attach dev
env | grep GITHUB_TOKEN
env | grep CLAUDE_CODE_AUTH_TOKEN

# 7. クリーンアップ
kubectl kodama delete dev  # シークレットも自動削除
```

## 詳細情報

完全なガイドとexampleは以下を参照:

- `examples/unified-credentials/` - 完全なセットアップガイド
- `examples/dotenv-template/` - .envファイルの使用例
- `README.md` - 基本的な使い方
- `.kodama.yaml.example` - テンプレート設定の例

🎉 **すべての認証情報が.envファイルで統一管理できます！**
