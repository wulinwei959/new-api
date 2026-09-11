# 更新日志

> 本文档记录了 New API 在统一模型池（Unified Model）相关功能上的核心改动，涵盖路由策略、计费精度、性能优化与稳定性增强。

---

## 2026-09-10 ~ 2026-09-11：统一模型池路由与稳定性优化

本次更新围绕统一模型池的选路策略、计费准确性、错误处理与系统稳定性进行了一系列优化，主要包含以下内容：

### 新增渠道适配器

- 新增 **Agnes AI**、**NVIDIA NIM**、**SenseNova** 三个 OpenAI 兼容渠道适配器
- 支持国际版与国内版 Agnes（共用适配器，默认基础地址不同）
- 补充对应前端图标与渠道配置

### 统一模型池选路策略优化

#### 1. 多分组统计聚合

- `resolveUnifiedMemberGroup` 改为返回所有匹配的用户分组，而非仅第一个
- 成员统计从单分组改为跨所有匹配分组加权聚合（请求量、成功率、延迟、TTFT、TPS）
- 评分更准确，避免因单分组数据不足导致误判

#### 2. 冷启动惩罚

- 新成员或数据积累不足（<100 次请求）时，评分按 `sqrt(count/100) × 0.5` 打折
- 防止新加入的渠道在尚无历史数据时被过度选中，避免"羊群效应"

#### 3. 配置修改即时生效

- 统一模型配置保存后，自动清除成员统计缓存（`ClearMemberStatsCache`）
- 管理员修改权重、最小分数等配置后，下次请求立即生效，无需等待 TTL 过期

#### 4. 选路结果缓存

- 新增 `routeResultCache`，首次选路成功后缓存 `(channelId, group)` 结果
- 相同模型+分组在 TTL 内复用上次选路，减少头部抖动
- 重试场景（排除已选渠道）不读缓存，保证故障转移正确性

#### 5. 错误类型敏感度区分

- `Sample` 新增 `IsServerError` 字段，区分 4xx（客户端错误）与 5xx（服务端错误）
- 选路评分中只用 server error rate 作为健康度惩罚，4xx 不再影响渠道评分
- `ChannelStats` 新增 `ServerErrorCount` 字段，数据库增加 `server_error_count` 列

#### 6. 连续 Server Error 自动降级冷却

- 新增 `RecordConsecutiveFailure` / `ClearConsecutiveFailure` / `IsChannelInCooldown` 接口
- 渠道连续出现 5xx 错误达到阈值（默认 5 次）后进入冷却期
- 冷却期间该成员被 `suppressCooldDownMembers` 过滤，不再参与选路
- 恢复正常请求后自动清除计数，冷却期结束后自动恢复

### 性能与稳定性

- 统一模型池健康视图（`/api/unified_model/health`）按 `auto` 分组解析成员统计，修复管理端无法显示数据的问题
- 修复 `/v1/models` 接口返回统一模型 ID 的行为
- 统一模型池成员卡片编辑对话框展示实时健康指标
- 新增按模型独立 RPM/TPM 限速配置入口
- 补全 7 种语言（en/zh/zh-TW/fr/ja/ru/vi）的 i18n 翻译 key
- 修复时间单位显示（ms/s 统一使用 ms）

### 数据库变更

- `perf_metrics` 表新增 `server_error_count` 列（用于区分错误类型）
- 迁移脚本已处理，老数据不受影响

### 涉及文件

| 模块 | 文件 | 变更说明 |
|------|------|----------|
| 渠道适配器 | `relay/channel/agnes/`, `nvidia/`, `sensenova/` | 新增三个 OpenAI 兼容适配器 |
| 选路策略 | `service/unified_model_selector.go` | 多分组聚合、冷启动惩罚、选路缓存、冷却过滤 |
| 限速管理 | `service/model_rate_limit.go` | 新增连续失败计数与冷却接口 |
| 性能指标 | `pkg/perf_metrics/types.go`, `metrics.go`, `flush.go` | 新增 server error 统计字段 |
| 数据库模型 | `model/perf_metric.go` | `PerfMetric` 新增 `ServerErrorCount` 列 |
| 配置管理 | `model/option.go` | 配置修改时清除选路缓存 |
| 前端页面 | `web/src/features/unified-model/` | 统一模型管理页面、健康指标展示、i18n |
| 国际化 | `web/src/i18n/locales/*.json` | 补全缺失翻译 key |

---

### 2026-09-11：HTTPS 支持（无外部依赖）

本次更新为 New API 添加了完整的 HTTPS 支持，无需引入任何外部 Go 依赖，完全基于 Go 标准库实现。

### 功能说明

- **TLS/HTTPS 支持**：通过环境变量配置，无需修改核心代码即可启用 HTTPS
- **自动签发证书**：支持自动签发 RSA 2048 自签证书，适用于局域网部署
- **手动上传证书**：支持通过环境变量指定证书和私钥路径，适用于外网部署
- **HTTP→HTTPS 重定向**：可选开启，自动将 HTTP 请求重定向到 HTTPS
- **前端 TLS 设置页面**：在系统设置 - 安全与限制中新增 TLS 设置选项
- **多语言支持**：7 种语言完整翻译

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| TLS_ENABLED | false | 是否启用 TLS |
| TLS_CERT_FILE | 空 | 手动上传模式：证书文件路径 |
| TLS_KEY_FILE | 空 | 手动上传模式：私钥文件路径 |
| TLS_AUTO_CERT | false | 自动签发自签证书（覆盖证书路径配置） |
| TLS_MIN_VERSION | 1.2 | 最低 TLS 版本，可选 1.2 或 1.3 |
| HTTP_TO_HTTPS_REDIRECT | false | 是否开启 HTTP→HTTPS 301 重定向 |
| HTTP_REDIRECT_PORT | 8080 | 重定向监听端口 |
| TLS_HTTPS_HOST | localhost | HTTPS 主机名（用于重定向地址） |

### 自签证书说明

- 自动签发模式下，证书存储在 DATA_DIR/tls/ 目录下
- 证书有效期为 1 年，提前 30 天自动续期
- 证书仅包含 localhost、127.0.0.1、::1 的 SAN
- 浏览器会显示不安全警告，适用于内网环境

---

# 技术细节

### 选路评分公式

```
score = 0.40 × successRate + 0.30 × latencyScore + 0.20 × throughputScore + 0.10 × availabilityScore
```

其中：
- `successRate` = (总请求成功数 - server error 数) / 总请求数
- `latencyScore` = 1000 / (1000 + avg_latency_ms)
- `throughputScore` = min(avg_tps / 20, 1.0)
- `availabilityScore` = successRate（与成功率同值，反映可用性）

### 冷启动惩罚

```
penalty = sqrt(requestCount / 100) × 0.5,  当 requestCount < 100
```

### 冷却机制

- 连续 server error（5xx）计数达阈值后，渠道进入冷却期
- 冷却期时长 = 阈值秒数（默认 5 秒）
- 冷却期间该渠道不参与选路，避免持续向故障渠道发送请求
- 成功请求或 4xx 错误时自动清除计数

---

> [!NOTE]
> 以上优化均在 `dev` 分支开发，已通过编译验证与单元测试。多节点部署场景下，冷却计数依赖 Redis，需确保 `REDIS_CONN_STRING` 正确配置。
