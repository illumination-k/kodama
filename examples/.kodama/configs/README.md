# Editor Configuration

このディレクトリにはKodamaセッション用のエディタ設定サンプルが含まれています。

## 使い方

この`.kodama`ディレクトリをリポジトリルートにコピーしてカスタマイズできます:

```bash
cp -r examples/.kodama /path/to/your/repo/
```

## 設定ファイル

- `helix/config.toml` - Helixエディタの基本設定
  - テーマ、行番号、カーソル形状、LSP設定など

- `helix/languages.toml` - 言語別設定とフォーマッター
  - Go、Python、Rust、TypeScript、JavaScriptなどの設定
  - 各言語のauto-format設定

- `zellij/config.kdl` - Zellijターミナルマルチプレクサー設定
  - tmuxスタイルのキーバインディング（Ctrl+a）
  - マウスサポート、コピー設定
  - Helixとの統合（scrollback_editor）

## デフォルト設定

設定ファイルが存在しない場合、Kodamaはクラウド開発に最適化されたデフォルト設定を自動的に使用します。カスタム設定は必須ではありません。

## カスタマイズ例

### Helixのテーマを変更

```toml
# helix/config.toml
theme = "dracula" # または "gruvbox", "nord"など
```

### 追加の言語サポート

```toml
# helix/languages.toml
[[language]]
name = "ruby"
auto-format = true
formatter = { command = "rubocop", args = ["--autocorrect-all", "--stdin", "%"] }
```

### Zellijのキーバインディング変更

```kdl
// zellij/config.kdl
keybinds {
    normal {
        bind "Ctrl b" { SwitchToMode "tmux"; }  // tmuxと同じプレフィックス
    }
}
```

## トラブルシューティング

設定が反映されない場合:

```bash
# Podに接続して設定を確認
kubectl exec -it kodama-<session-name> -- bash
ls -la /root/.config/helix/
cat /root/.config/helix/config.toml

# セッションを再作成
kubectl kodama delete <session-name>
kubectl kodama start <session-name> --sync
```

## 参考リンク

- [Helix Editor Documentation](https://docs.helix-editor.com/)
- [Zellij Documentation](https://zellij.dev/documentation/)
