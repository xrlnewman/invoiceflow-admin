# InvoiceFlow Admin

InvoiceFlow 是发票收款运营后台，覆盖客户抬头、开票申请、审核、开具、回款登记、自动核销、对账和归档。财务专员处理待办，销售查看客户回款，主管复核异常和审计事件。

## 财务流程

1. 销售提交客户抬头、税率和明细，生成 `待审核` 开票申请。
2. 财务审核通过后登记发票号码，状态进入 `已开具`；驳回原因写入事件时间线。
3. 收款人员登记到账金额，系统支持部分回款、差额提醒和按发票余额匹配。
4. 核销结果回流客户余额与月度对账，账期锁定后进入 `已归档`。
5. 所有写请求要求 `Idempotency-Key`，Redis 负责幂等结果和并发锁，MySQL 8.4 负责持久化。

```bash
# 一键启动 API + MySQL 8.4 + Redis 8（会自动加载合成演示数据）
docker compose -f deploy/docker-compose.yml up --build

# 或仅使用无外部依赖的内存模式运行 API
go run ./server

# 管理后台
cd web && npm install && npm run dev
```

前端默认请求 `/api/v1`，Vite 开发服务器会把 `/api` 和 `/healthz` 代理到 `http://localhost:8080`。部署到独立域名时，可在构建时设置 `VITE_API_BASE_URL=https://api.example.com`；客户端会自动补齐 `/api/v1`，所有创建、确认、状态推进和回访完成请求都会自动生成 `Idempotency-Key`。

后台的“发票队列”“回款核销”“对账归档”按钮会优先调用真实 API；API 暂不可用时保留内置演示数据并提示当前数据来源。侧栏“移动端体验”提供销售与财务的窄屏视图，支持提交开票、登记回款、查看余额和跟进提醒，便于用手机浏览器联调。

## API 示例

```bash
# 创建发票（重复发送相同 Idempotency-Key 只会创建一次）
curl -X POST http://localhost:8080/api/v1/appointments \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-create-001' \
  -d '{"patient":"演示客户","department":"全科门诊","doctor":"林负责人","scheduledAt":"2026-07-16T09:00:00+08:00"}'

# 推进状态：待确认 -> 已确认 -> 候诊中 -> 处理中 -> 已完成（将 AP-1001 替换为上一步返回的 id）
curl -X POST http://localhost:8080/api/v1/appointments/AP-1001/checkin -H 'Idempotency-Key: demo-checkin-001'
curl -X POST http://localhost:8080/api/v1/appointments/AP-1001/status \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-waiting-001' -d '{"status":"候诊中"}'

# 查看审计事件
curl http://localhost:8080/api/v1/appointments/AP-1001/events

# 完成回访
curl -X POST http://localhost:8080/api/v1/followups/FW-0716-001/complete -H 'Idempotency-Key: demo-followup-001'
```

演示数据均为虚构数据；项目不得用于真实医疗诊断、处方、支付或客户隐私存储。

## 运行范围

InvoiceFlow 覆盖开票、审核、开具、收款、核销、对账和归档的财务操作；所有演示数据均为虚构，不接入真实财务、人事或客户隐私。

