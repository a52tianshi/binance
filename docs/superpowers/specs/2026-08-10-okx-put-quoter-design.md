# OKX PUT 卖单自动改价工具 — 设计

## 背景与目标

账户在 OKX 期权上持有若干 PUT 卖单（sell to open/close，即报出的卖出挂单）。希望有一个自动化脚本持续维护这些挂单的价格，遵循两条规则：

1. **不在卖一时**：如果卖一价格 (`px1`) 高于标记价格 (`markPx`)，把我们的挂单价改到卖一价，去抢占卖一位置。
2. **已在卖一、且独占卖一时**：如果卖一与卖二的价差大于一个 tick，把我们的挂单价往下改到「卖二价 - 1 tick」，即在仍保持最优价的前提下少让利。

只针对 ETH 期权。语言用 Go，独立轮询脚本，5 秒一轮，支持 dry-run。

## 范围假设

- 只处理 ETH 标的、`side=sell`、期权类型为 `P`（PUT）的挂单。
- 假设同一个合约(instId)上我们自己最多只有一笔挂单。如果发现同一 instId 上有多笔，跳过该 instId 并打印警告日志（不处理，避免逻辑歧义）。
- 「独占卖一」的判定：订单簿卖一档总挂单量 `sz1` 等于我们自己这笔挂单的剩余数量 `ourSz`。

## 架构

新建独立文件夹 `okx_put_quoter/`（Go `package main`，复用仓库根 `module main`，`go build ./okx_put_quoter` 单独产出二进制，不影响仓库其它程序）。

```
okx_put_quoter/
  main.go            // 轮询主循环、优雅退出(Ctrl+C)
  config.go          // 读取 .env + 命令行参数(--dry-run, --interval)
  okx_client.go       // REST 请求封装：HMAC-SHA256 签名、GET/POST
  orders.go           // 拉取 pending orders、过滤 PUT+sell(ETH)
  market.go           // 订单簿(asks[0..1])/标记价/tick bands 获取与缓存
  quoting.go          // 规则A/B 核心决策与改价动作
  .env.example        // OKX_API_KEY / OKX_API_SECRET / OKX_API_PASSPHRASE
  README.md
```

主循环（每 `poll_interval_sec` 秒一次，默认5秒）：

1. `GET /api/v5/trade/orders-pending?instType=OPTION` 拉出所有未成交订单，过滤出 `instId` 含 `ETH-` 且 `optType=P`、`side=sell` 的。
2. 按 `instId` 分组；若某 instId 下我们有 >1 笔挂单，记警告日志、跳过。
3. 对每个（instId, 唯一挂单）：
   - 若该 instId 的 tick 分档表未缓存过，调用 `GET /api/v5/public/instrument-tick-bands?instType=OPTION&instId=...` 拉取一次并缓存到内存（进程生命周期内不再重复请求同一 instId）。
   - `GET /api/v5/market/books?instId=...&sz=2` 拿 asks[0]、asks[1]（价格+数量）。
   - `GET /api/v5/public/mark-price?instType=OPTION&instId=...` 拿标记价。
   - 根据当前 asks[0] 价格所在区间，从缓存的分档表里取对应 tickSz。
4. 执行决策逻辑（见下）。
5. 需要改价时调用 `POST /api/v5/trade/amend-order`（`instId`, `ordId`, `newPx`，只改价不改量）；`dry_run=true` 时只打日志不发请求。

## 决策逻辑（`quoting.go`）

输入：`ourPx`（我们的挂单价）、`ourSz`（剩余数量）、`px1/sz1`（卖一价/量）、`px2`（卖二价）、`markPx`、`tickSz`。

```
if ourPx == px1 {
    // 我们在卖一
    if sz1 == ourSz {
        // 独占卖一
        if px2 - px1 > tickSz {
            newPx := px2 - tickSz
            amend(newPx)  // 规则B：收窄让利，仍保持最优
        }
        // 否则不动
    }
    // 非独占：不动
} else {
    // 我们不在卖一
    if px1 > markPx {
        amend(px1)  // 规则A：追平卖一
    } else {
        amend(markPx + tickSz)  // 边界：卖一异常(<=标记价)，兜底挂到标记价+1tick
    }
}
```

价格比较需用最小单位（分档 tickSz 的整数倍）做定点运算，避免浮点误差；建议将价格转换为「相对 tickSz 的整数格数」或用 `decimal`/放大成最小单位的 int64 处理。

## 错误处理

- 单个合约在某一轮的行情/改价请求失败：记录日志、跳过该合约，不影响其它合约、不中断主循环。
- OKX 鉴权失败（401等）：记录日志，continue；连续失败达到阈值（如连续10轮）时打印明显的高亮警告，避免静默失效。
- `amend-order` 返回业务错误（订单已成交/已撤销等）：视为正常的竞态情况，记录日志，不告警。

## 日志

- 每次真实改价或 dry-run 模拟改价，打印一行：`instId`、触发规则(A/B/边界)、旧价→新价、当前 tickSz。
- 同一 instId 检测到多笔挂单时打印一次警告并跳过。

## 配置与凭证

- `.env`（不入库）：`OKX_API_KEY` / `OKX_API_SECRET` / `OKX_API_PASSPHRASE`。
- 命令行参数：`--dry-run`（默认 false）、`--interval`（秒，默认5）。
- 根目录 `.gitignore` 补充 `.env`。

## 测试思路

- `quoting.go` 的决策函数是纯函数（价格/数量输入 → 是否改价+新价），可以直接写单元测试覆盖：规则A触发/不触发、规则B触发/不触发（含独占判断）、边界情况(px1<=markPx)。
- REST 客户端签名逻辑可以用 OKX 文档给出的示例向量做单元测试。
- 端到端行为通过 dry-run 模式人工核对日志验证，不做 mock OKX 服务器的集成测试（YAGNI）。
