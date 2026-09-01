# 客户端 · 钱包

## 页面职责

充值余额、兑换码兑换、查看余额流水。不做订阅或 API 套餐。

| 项目 | 值 |
| --- | --- |
| 前端路由 | `/app/wallet` |
| 页面组件 | `frontend/src/pages/client/WalletPage.vue` |
| 分页 | 流水 `pageSize = 10` |

## 业务流程

```text
充值：选择档位 → 调起支付 → 回调入账 → 跳转支付结果页
兑换：输入兑换码 → POST redeem → 余额增加 + 流水
流水：GET transactions 分页列表
```

任务预扣/退款流水由任务域写入，本页只读展示。

## 页面区块

### 1. 页头余额

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `wallet.balance` | `number` | 可用余额 |

### 2. 充值档位 `products[]`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 如 `p-2` |
| `amount` | `number` | 面额（元） |
| `bonus` | `number` | 赠送金额 |
| `price` | `number` | 实付价（通常等于 amount） |

用户实收：`amount + bonus`。

### 3. 兑换

| 请求字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | `string` | 兑换码，大小写不敏感 |

### 4. 余额流水 `transactions[]`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `TX_*` |
| `type` | `string` | 任务消费 / 充值到账 / 任务退款 / 兑换到账 / 工单退款 |
| `amount` | `number` | 正为入账，负为支出 |
| `desc` | `string` | 说明 |
| `time` | `string` | 展示时间 |

## 推荐接口

### 钱包概览 + 档位

```
GET /api/v1/client/wallet
```

```json
{
  "balance": 256.0,
  "products": [
    { "id": "p-1", "amount": 50, "bonus": 0, "price": 50 },
    { "id": "p-2", "amount": 100, "bonus": 5, "price": 100 }
  ],
  "paymentMethods": ["wechat_pay"]
}
```

### 创建充值订单

```
POST /api/v1/client/wallet/recharge-orders
{ "productId": "p-2", "paymentMethod": "wechat_pay" }
```

```json
{
  "orderId": "ord_xxx",
  "payUrl": "weixin://...",
  "amount": 100,
  "bonus": 5,
  "expectedCredit": 105
}
```

支付成功后 webhook 入账；前端跳转 `/auth/payment-result?amount=105`。

### 兑换

```
POST /api/v1/client/wallet/redeem
{ "code": "S2V-WELCOME-20" }
```

```json
{
  "credited": 20,
  "balance": 276.0,
  "ledgerEntryId": "TX_90202"
}
```

错误：无效码、已使用、已过期 → `400`。

### 流水列表

```
GET /api/v1/client/wallet/transactions?page=1&pageSize=10
```

```json
{
  "items": [
    {
      "id": "TX_90182",
      "type": "任务消费",
      "amount": -4.5,
      "desc": "tsk_88921 雨夜街道霓虹漫步",
      "time": "2026-09-01T14:20:00+08:00"
    }
  ],
  "total": 4,
  "page": 1,
  "pageSize": 10
}
```

## 业务规则

1. 任务提交时预扣；失败、终止、工单核准退款写负向/正向流水。
2. 充值赠送 `bonus` 与本金一并进入可用余额，无单独冻结金（首期）。
3. 兑换码一用户一次（原型 `usedCodes` 记录）。
4. 扣费规则说明为静态文案，可链到帮助文档。

## 与当前 mock 对应

| 数据/操作 | 来源 |
| --- | --- |
| 余额 | `balance` |
| 档位 | `rechargeProducts`（mock.ts） |
| 充值 | `recharge(amount)` 模拟到账 |
| 兑换 | `redeem(code)` + `redeemCodes` |
| 流水 | `transactions` + `pushLedger` |

支付结果页：`/auth/payment-result`（非主导航）。
