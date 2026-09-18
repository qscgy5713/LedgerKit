# LedgerKit

一個相容 hledger/ledger 純文字複式記帳格式的 CLI，單一執行檔、零資料庫、記帳資料就是一份你自己可讀、可 diff、可用 git 版控的文字檔。

## 動機

複式記帳（double-entry bookkeeping）是最不容易記錯帳的記帳方式——每一筆交易都要讓借貸兩邊加總為零，錯誤（漏記、記重複）很容易在報表上現形。市面上的 hledger/ledger 已經是成熟工具，但功能龐大；LedgerKit 只實作日常個人記帳用得到的子集，換取一個更小、更容易看懂內部邏輯的實作，同時保留跟原生 journal 格式相容（可以互相搭配既有的 hledger 生態工具）。

## 運作方式

所有資料存在一份純文字 journal 檔案裡，每筆交易是「日期 + 描述」加上兩筆以上的「過帳（posting）」，每筆過帳是「帳戶 + 金額」。只要同一筆交易裡所有過帳金額加總為零，帳就是平的：

```
2026-01-03 * 買咖啡
    expenses:food:coffee    $120
    assets:cash
```

第二筆過帳省略了金額——LedgerKit 會自動推算成 `-$120`，讓整筆交易平衡。這是 hledger/ledger 的標準寫法（elided amount），也是最常用的省事寫法：大部分交易都只有兩隻腳，寫一邊金額、另一邊自動推算就好。

LedgerKit 讀進整份 journal 後，用帳戶名稱的 `:` 階層（例如 `assets:cash`、`expenses:food:coffee`）把過帳彙總成報表：`balance` 看某個時間點的餘額分佈，`register` 看逐筆流水帳與累計餘額。

## 安裝

```bash
go install ledgerkit/cmd/ledgerkit@latest   # 或 clone 下來 make build
```

## 快速上手

```bash
export LEDGERKIT_JOURNAL=~/finance.journal   # 不設的話預設用 ./ledger.journal

ledgerkit init                # 建立一份新的 journal
ledgerkit add                 # 互動式記一筆帳
ledgerkit balance             # 看目前各帳戶餘額
ledgerkit register assets:cash  # 看 assets:cash 的逐筆流水帳
```

## Journal 格式

LedgerKit 支援 hledger/ledger journal 語法的一個子集：

- 交易標頭：`YYYY-MM-DD` 或 `YYYY/MM/DD`，後面可加 `*`（已核對）或 `!`（待核對）狀態，再接描述文字
- 過帳行：縮排 + 帳戶（`:` 分層）+ 金額（可留空讓系統推算）
- 金額可以是幣別符號前綴（`$120`）或後綴代碼（`120 TWD`），也可以不寫幣別
- `;` 開頭或行內 `;` 之後的內容是註解
- 空白行分隔交易

**目前不支援**（有需要可以之後再加）：多幣別匯率轉換、budget、周期性交易、`include`、tag、期初/期末結算。

## 指令

- `ledgerkit init [path]` — 建立新 journal（`-force` 覆蓋既有檔案）
- `ledgerkit add` — 互動式問答新增一筆交易並附加到 journal
- `ledgerkit balance [account-pattern] [-from DATE] [-to DATE]` — 依帳戶階層顯示餘額與小計
- `ledgerkit register [account-pattern] [-from DATE] [-to DATE]` — 依時間顯示逐筆過帳與累計餘額
- `ledgerkit accounts` — 列出 journal 裡出現過的所有帳戶
- `ledgerkit check` — 驗證 journal：語法錯誤（附行號）、交易借貸不平衡
- `ledgerkit import -account ACCOUNT [flags] <csv-file>` — 從銀行對帳單 CSV 匯入交易

  v1 匯入邏輯是通用的欄位對應（`-date-col`/`-desc-col`/`-amount-col` 指定第幾欄），不是特定銀行格式解析器；每一列會匯入指定帳戶 + 對方帳戶（預設 `expenses:unclassified`，可用 `-to` 換）兩隻腳的交易，重複匯入同一份 CSV 不會產生重複交易（用每列內容的 hash 記錄在 `<csv 檔>.ledgerkit-import-state`）。

  ```bash
  ledgerkit import -account assets:bank:checking -header bank_export.csv
  ```

所有指令都吃 `-journal <path>` 覆蓋要讀寫的 journal 檔案（預設看 `LEDGERKIT_JOURNAL` 環境變數，否則是 `./ledger.journal`）。

## 網頁 / HTTP API：`ledgerkit-server`

`cmd/ledgerkit-server` 是另一個執行檔，把同一份 journal 檔案透過 HTTP 開放出來，給網頁介面（也內建在同一個執行檔裡）跟手機 App 用。CLI 跟 server 讀寫同一份 journal 檔案，是同一份資料。

```bash
export LEDGERKIT_JOURNAL=~/finance.journal
export LEDGERKIT_API_TOKEN=$(openssl rand -hex 32)   # 必填，沒設會拒絕啟動
export LEDGERKIT_ADDR=:8080                            # 預設 :8080

make run-server
# 瀏覽器打開 http://localhost:8080/，在「Settings」分頁貼上上面那組 token
```

網頁有四個分頁：Balance（餘額報表）、Register（逐筆流水帳）、Add（記一筆帳）、Settings（設定 API token；如果網頁跟 API 不同源，例如手機 App，這裡還要填伺服器網址）。

**API 端點**（都需要 `Authorization: Bearer <token>`）：

| 方法/路徑 | 說明 |
|---|---|
| `GET /api/accounts` | 帳戶清單 |
| `GET /api/balance?account=&from=&to=` | 依帳戶階層的餘額報表 |
| `GET /api/register?account=&from=&to=` | 逐筆過帳與累計餘額 |
| `GET /api/check` | journal 驗證結果 |
| `POST /api/transactions` | 新增一筆交易，body 見 `cmd/ledgerkit-server/dto.go` |

**已知限制**：
- server 只講 HTTP，沒有做 TLS——要從家裡以外的網路連，自己接 Tailscale 或 Caddy 之類的反向代理補 TLS，不要把 `LEDGERKIT_API_TOKEN` 用明碼 HTTP 傳過公開網路。
- server 內部用一個 mutex 序列化自己的讀寫，但擋不住 CLI 跟 server 同時寫同一份 journal 檔案——個人使用情境下風險低，目前沒做跨行程鎖。

## 手機 App（Capacitor + Android）

`mobile/` 是一個 Capacitor 專案，把 `cmd/ledgerkit-server/web` 那份網頁包裝成 Android App（`capacitor.config.json` 的 `webDir` 直接指過去，不是另外複製一份維護）。

```bash
cd mobile
npm install
npx cap sync android   # 把最新的 web/ 內容同步進 android/ 專案
npx cap open android   # 用 Android Studio 開啟，接著跟一般 Android 專案一樣建置/簽名/安裝
```

**目前狀態**：專案骨架已經建好（`npx cap doctor` 通過），但實際編譯 APK需要 Android Studio + JDK + Android SDK——這幾個目前沒裝，所以還沒有人真的建置過、也還沒在實機上測試過。裝好 Android Studio 之後，打開 `mobile/android` 應該就能直接建置；App 執行時要在 Settings 分頁填伺服器網址（手機連得到的位址，例如你的 Tailscale IP）跟 API token。

## Development

```bash
make build          # 編譯出 ./ledgerkit
make build-server   # 編譯出 ./ledgerkit-server
make test           # go test ./... -race
make lint           # gofmt -l . && go vet ./...
```

核心邏輯分三層：`internal/journal` 負責 parse/format 與型別（金額用手刻定點小數避免浮點誤差）、`internal/ledger` 負責 balance/register 的彙總邏輯（`FlattenBalance` 這個走訪函式同時給 CLI 跟 HTTP API 用，兩邊輸出保證一致）、`internal/importer` 負責 CSV 匯入與去重。`cmd/ledgerkit` 只做參數解析與輸出排版（`internal/render`）；`cmd/ledgerkit-server` 只做 HTTP 路由與 JSON 轉換（`dto.go`），domain 邏輯一樣重用 `internal/journal`／`internal/ledger`，不重寫。

## License

MIT，見 [LICENSE](LICENSE)。
