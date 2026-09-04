# T6 晋升答辩项目架构资料

> 整理范围：`/root/zhaoling/workspace` 下五个项目
> 整理人：诸葛小亮（AI 资料探子）｜整理日期：2026-08-19
> 用途：员工 T6 晋升答辩素材
> 说明：loongnix_file_server、dify-on-dingding、emu_autotest 为自研项目；mcp-lightrag、mcp-memory-service 为基于开源项目的本地增强/部署，答辩时建议如实标注"基于开源 + 本地二次开发/增强"。

---

## 目录

1. [loongnix_file_server（龙芯文件中转服务）](#1-loongnix_file_server)
2. [dify-on-dingding（Dify 钉钉机器人集成）](#2-dify-on-dingding)
3. [emu_autotest（EMU 自动化保存现场/重置/重烧服务）](#3-emu_autotest)
4. [mcp-lightrag（LightRAG MCP 服务）](#4-mcp-lightrag)
5. [mcp-memory-service（多智能体共享记忆后端）](#5-mcp-memory-service)

---

## 1. loongnix_file_server

### 一句话定位
面向龙芯（Loongnix）环境的 Flask 文件中转服务：模型生成结果上传后签发带有效期、防伪造的临时下载链接，支撑多实例共享存储部署。

### 技术栈
- **语言/框架**：Python 3 + Flask 3.1.1（`requirements.txt`：Flask、itsdangerous 2.2.0、gunicorn 23.0.0、Werkzeug、filetype 1.2.0、python-dotenv）
- **签名**：itsdangerous `URLSafeTimedSerializer`
- **文件类型识别**：`filetype`（魔数识别）
- **部署**：Gunicorn（4 worker/4 thread）+ Docker（`Dockerfile`），外部 Nginx/APISIX 网关

### 核心模块/目录结构
```
loongnix_file_server/
├── app.py               # 唯一入口：/api/upload（上传）、/d/<token>（下载）、/healthz
├── README.md            # 使用文档
├── TECHNICAL_SPEC.md    # 技术设计文档（背景、目标、核心设计、部署架构）
├── UPLOAD_API_GUIDE.md  # 上传接口对接指南
├── Dockerfile / .env.example / requirements.txt
```

### 架构亮点与关键技术点
1. **签名临时链接（核心亮点）**：`app.py` 中下载 token 由 `URLSafeTimedSerializer` 生成，payload 含 `path`、`exp`（过期时间戳）、`nonce`（`secrets.token_urlsafe(8)` 随机噪声，防链接被猜测/重放）；下载时 `serializer.loads(..., max_age=MAX_LINK_EXPIRE_SECONDS)` 做防重放硬上限，再用 `exp` 做真实过期点，双保险。
2. **三重文件校验**：`is_allowed_file()` 同时校验扩展名白名单（`ALLOWED_IMAGE_EXTENSIONS`/`ALLOWED_VIDEO_EXTENSIONS`）、HTTP Content-Type、文件头魔数（`detect_magic_mime()` 读前 261 字节用 `filetype.guess()` 识别并 `seek(0)` 复原流位置）；MIME 前缀必须匹配 `image/` 或 `video/`。
3. **路径穿越防护**：下载时 `(storage_root / path).resolve()` 后校验 `storage_root not in full_path.parents`，从根上杜绝 `../` 逃逸；文件不存在返回 404、签名错误 403、过期 410。
4. **多实例水平扩展**：共享存储（如 GPFS 挂载 `/files`）+ 一致 `SECRET_KEY`/`TOKEN_SALT`/`UPLOAD_API_KEY`，A 机上传 B 机可验可下（详见 TECHNICAL_SPEC.md 第 4 节）。
5. **工程化细节**：按 `YYYYMM` 月目录归档 + UUID 文件名；`TimedRotatingFileHandler` 日志按天切分保留 30 天；上传鉴权支持 `X-Upload-Token` 与 `Authorization: Bearer` 两种方式，且用 `secrets.compare_digest` 防时序攻击；超限返回 413 并提示可上传上限。

### T6 答辩素材点
- **安全设计完整度**：签名 + 过期 + nonce + 魔数校验 + 路径穿越防护 + 时序安全比较，一套文件服务安全闭环可完整讲述。
- **业务落地价值**：解决龙芯环境文生图/文生视频结果分发痛点，把"模型输出到用户拿到链接"链路压缩为一次 HTTP 调用。
- **可运维性**：多实例 + 共享存储 + 一致密钥的水平扩展方案，有清晰的部署架构与网关限流建议。
- **文档完备**：README/TECHNICAL_SPEC/UPLOAD_API_GUIDE 三份文档分工明确，体现工程化交付习惯。

---

## 2. dify-on-dingding

### 一句话定位
轻量级 Dify × 钉钉机器人集成网关：通过钉钉 Stream 模式接收群聊/单聊消息，对接 Dify 聊天助手/工作流/文本生成三类应用，实现 AI 卡片流式打字机输出与用户级会话上下文管理。

### 技术栈
- **语言**：Python 3.10+
- **依赖**（`requirements.txt`）：`dingtalk_stream`（钉钉 Stream 模式）、`websockets`、`requests`、`sseclient-py`（解析 Dify SSE 流）、`loguru`、`python-dotenv`、`PyYAML`
- **并发模型**：`concurrent.futures.ThreadPoolExecutor`，每 bot 独立监听线程（`max_workers` 可配）
- **部署**：Docker + docker-compose（`docker/docker-compose.yaml`）

### 核心模块/目录结构
```
dify-on-dingding/
├── app.py                # 入口：加载 .bots.yaml 多 bot 配置，按 app 类型建 client，线程池启动
├── configs.py            # 环境变量 + .bots.yaml 加载
├── core/
│   ├── handlers.py       # HandlerFactory + DifyAiCardBotHandler（消息处理主逻辑）
│   ├── dify_client.py    # DifyClient 基类 + ChatClient/CompletionClient/WorkflowClient
│   └── cache.py          # 简易带过期时间的内存缓存（用户会话上下文）
├── tests/dify_client_test.py
└── .bots.yaml.example    # 多机器人 → Dify 应用映射配置模板
```

### 架构亮点与关键技术点
1. **AI 卡片流式打字机**：`handlers.py` 先 `AICardReplier.create_and_send_card()` 投卡（`callback_type="STREAM"`），快速返回 Ack 避免钉钉超时重试，再 `asyncio.create_task(update_card())` 异步持续 `card_instance.streaming()` 更新卡片，实现打字机效果——"先 ack 后流式"解决长任务超时的经典手法。
2. **SSE 流式事件解析引擎**：`_consume_stream_text()` 自带流式状态机，专门处理 `<think>...</think>` 思考标签的跨 chunk 拆包（`_tag_prefix_tail_len()` 保留标签前缀尾巴防误吞），区分"思考中"与"正式回答"两种渲染态，并做增量节流（思考态每 1 字符、回答态每 10 字符更新）。
3. **多应用类型适配**：`HandlerFactory` + 三类 client（`ChatClient`/`CompletionClient`/`WorkflowClient`）统一 `query()` 接口，适配 Dify 三种应用类型；约定工作流/文本生成应用输入变量名必须为 `query`。
4. **用户级会话上下文**：以 `sender_staff_id` 为 key，在 `cache.py` 内存缓存中保存 `conversation_id`，会话过期时间可配（`DIFY_CONVERSATION_REMAIN_TIME`）；群聊中每用户独立上下文。
5. **多机器人水平扩展**：`app.py` 从 `.bots.yaml` 加载任意多 bot（client_id/secret → dify_app → handler），每 bot 可配线程数，单进程承载多机器人。
6. **富文本（门禁）定制扩展**：自定义改造支持解析钉钉富文本图片消息（提取 `downloadCode` + 文本），拼装 `ding_id`/`nick_name` 注入 Dify `inputs.sys_user_id`，实现"图片 + 门禁请求"类业务（git log：`门禁信息中加钉钉id`、`支持门禁图片消息`）。

### T6 答辩素材点
- **流式协议工程能力**：从钉钉 Stream 回调 → Dify SSE → AI 卡片增量渲染的整条流式链路，含思考标签跨包解析的边界处理，技术含量足。
- **异步超时控制**："快速 Ack + 后台异步任务更新卡片"规避平台超时重试，是可复用的平台集成设计模式。
- **多租户机器人架构**：配置驱动的多 bot 多 Dify 应用映射，展示"配置化、可扩展"的系统设计思路。
- **真实业务落地**：对接门禁申请等内部业务场景（富文本图片 + 用户身份透传），体现从通用工具到业务闭环的能力。

---

## 3. emu_autotest

### 一句话定位
EMU（模拟器）测试环境故障恢复自动化服务：把"人工登录烧录机操作 xeDebug GUI 保存现场/重置/重烧 + 登录 BMC 重启测试机"的人工链路，封装成一次 HTTP 调用。

### 技术栈
- **语言/框架**：Python + Flask（HTTP 层）+ Waitress（生产启动器）
- **GUI 自动化**：`pyautogui` + `ewmh`（EWMH 窗口枚举/激活）
- **BMC 自动化**：`requests` 直连 BMC CGI 接口（绕过浏览器）
- **部署**：虚拟环境压缩包 `emu.tar.gz` + shell 脚本（`start_service.sh`/`start_proxy.sh`），多账号多实例

### 核心模块/目录结构
```
emu_autotest/
├── main.py            # 后端 HTTP 入口：/save_and_reset、/save_and_burn
├── proxy.py           # 统一代理：按来源 IP 转发到 5007/5008/5009
├── xedebug_tool.py    # xeDebug GUI 自动化：save_env/reset_emu/close_emu/burn_emu
├── bmc_tool.py        # BMC CGI 封装：登录→取 Cookie→CSRF Token→重启→登出
├── constant.py        # asic_dir / bmc_host 配置
├── start_service.sh / start_proxy.sh / run_test.sh
└── README.md          # 详尽的技术说明文档
```

### 架构亮点与关键技术点
1. **无 API 场景的 GUI 自动化**：xeDebug 只有图形界面，`xedebug_tool.py` 通过 EWMH 枚举桌面窗口（`ewmh.getClientList()` 匹配标题含 `xeDebug`）→ `setActiveWindow()` 激活 → `pyautogui.write()` 输入命令；`save_env()` 自动拼 `stop -emulation / save -environment Initial_YYYYMMDDHHMMSS.tcl / run -nowait` 保存现场，`reset_emu()` 注入 `rst_assert.tcl`/`rst_desert.tcl`。
2. **BMC 无浏览器 CGI 自动化**：`bmc_tool.py` 直接模拟浏览器请求 BMC 的 `login.cgi` → 解析 `Set-Cookie` 提取 `__Host-SID` → `csrf_tokens.cgi` 获取 CSRF Token → 维持心跳 → 调 `server_power_control.cgi` 重启测试机；比 Selenium 全浏览器自动化更轻量。
3. **统一代理 + 多实例路由**：`proxy.py` 按 `request.remote_addr`（来源 IP 白名单 10.0.19.13/15/23 → 5007/5008/5009）转发，非白名单直接 403；测试脚本只感知一个统一入口。
4. **流程编排**：`/save_and_reset`（保存现场 → EMU 重置 → BMC 重启）与 `/save_and_burn`（保存现场 → 关闭 EMU → 后台线程烧录 `make run` → 等待 600s → BMC 重启），`threading.Thread` 将烧录放到后台不阻塞主流程。
5. **与测试框架联动闭环**：README 提供 `run_test.sh` 失败触发示例 + systemd 用户服务自动拉起方案，把"失败恢复 → 测试重启 → 续跑"闭环打通。

### T6 答辩素材点
- **攻坚无标准 API 的硬骨头**：xeDebug 无 CLI/API 场景下用 GUI 自动化+窗口管理实现"伪 API"，体现解决现实约束的工程智慧。
- **端到端自动化链路**：测试失败 → HTTP 调用 → 保存现场 → 重烧 → BMC 重启 → 测试重拉，完整闭环设计。
- **轻量级自动化选型**：BMC 直接 CGI 请求替代浏览器驱动，展示"按场景选最轻方案"的技术判断力。
- **多实例隔离部署**：一账号一实例一端口 + 来源 IP 路由代理，多租户隔离的实用做法。

---

## 4. mcp-lightrag

### 一句话定位
LightRAG 知识图谱的 MCP（Model Context Protocol）服务桥：把 LightRAG 的查询/文档/图谱操作封装为标准化 MCP 工具，供 AI 助手通过 stdio 直接调用；本地增强重点是智能 upsert 文档同步机制。

> 注：fork 自 `github.com/enriquecatala/mcp-lightrag`（v0.2.2），本地 git log 含 `support for upsert`、`add find a document`、`fix: Initialize LightRAGApiClient with get_settings()` 等增强提交。

### 技术栈
- **语言**：Python 3.11+
- **依赖**（`pyproject.toml`）：`mcp>=1.2.0`（FastMCP 框架）、`httpx>=0.28.1`、`pydantic>=2.11`、`python-dotenv`、`attrs`
- **API 客户端**：基于 LightRAG `openapi.json` 用 `openapi-python-client` 自动生成（`src/mcp_lightrag/client/light_rag_server_api_client/`）
- **工程配套**：`uv` 管理依赖、`ruff` lint、pytest（含 integration 标记）、hatchling 打包

### 核心模块/目录结构
```
mcp-lightrag/
├── src/mcp_lightrag/
│   ├── cli.py            # 入口：argparse → mcp.run(transport="stdio")
│   ├── mcp_tools.py      # 全部 MCP 工具定义（16+ 个 @mcp.tool）
│   ├── api_client.py     # LightRAGApiClient：重试/异常封装 + upsert 逻辑
│   ├── settings.py       # 环境变量配置（LIGHTRAG_HOST/PORT/API_KEY）
│   ├── models.py         # ServerSettings/QueryParams/OperationResult/BatchResult
│   └── client/…          # openapi-python-client 自动生成代码
├── tests/                # 单元 + 集成测试
└── openapi.json
```

### 架构亮点与关键技术点
1. **智能 upsert 文档同步（本地增强核心）**：`api_client.py::upsert_document()` 三态决策——文档不存在则上传（`created`）；存在且内容长度一致（含去尾空白、±2 字节容差处理 `\n` vs `\r\n`）则跳过（`skipped`）；存在但已修改则先 `delete_by_doc` 再重传（`updated`）。专为 Obsidian Vault 等持续演进知识库设计，避免重复索引。
2. **工具层统一封装**：`mcp_tools.py` 用 `format_output` 装饰器把所有工具响应统一为 `OperationResult.success/failure`，异常不逃逸到 MCP 协议层；`lifespan` 上下文管理 API client 生命周期。
3. **健壮的 API 通信**：`with_retry` 装饰器实现指数退避重试（3 次、base_delay=1s），且对 400/401/403/404 这类确定性错误不重试；支持无 API Key 时用普通 `Client` 避免发送空 `Bearer` 头。
4. **客户端代码生成**：基于 LightRAG `openapi.json` 自动生成 API client，LightRAG 接口演进时一条命令重新生成，消除手写对接代码的维护成本。
5. **工具面完整**：覆盖查询（`query_knowledge_graph` 支持 mix/semantic/keyword/global/hybrid/local/naive 七种模式）、文档（ingest/upload/batch/find/list/status）、图谱（entity/relation 增删改、merge、purge）、系统（health/schema）四类 16+ 工具。

### T6 答辩素材点
- **MCP 协议实践**：完整实现 MCP server（FastMCP + stdio transport + lifespan），走在 AI 工具标准化前沿。
- **增量同步算法设计**：upsert 三态判断 + 内容长度容差处理，是"知识库持续同步"场景的实用解法，可迁移到任意文档索引系统。
- **工程化完备度**：自动生成 client、统一响应封装、重试策略、测试矩阵，展示高质量开源工程习惯。
- **与知识图谱结合**：MCP 工具直连 LightRAG 图查询/实体关系管理，体现 RAG 与图数据库结合的技术视野。

---

## 5. mcp-memory-service

### 一句话定位
多智能体系统的持久化共享记忆后端：Agent 以 5ms 级延迟存取决策与知识，共享带因果类型边的知识图谱，支持本地 ONNX 向量化、自动记忆整合（consolidation）与 REST/MCP 双协议访问。

> 注：clone 自开源项目 `github.com/doobidoo/mcp-memory-service`（Apache-2.0，本地为 v10.39.x 基线），本地有 `docs/mcp-memory-service-installation-and-comparison.md` 等未跟踪文档，答辩时建议定位为"开源方案评估/部署/本地化增强"。

### 技术栈
- **语言/框架**：Python + FastAPI + uvicorn（REST/SSE）、FastMCP（MCP 协议）
- **存储**：`sqlite-vec`（本地向量检索扩展）+ aiosqlite；Graph storage（知识图谱）；Cloudflare 同步后端；混合检索（BM25 + vector）
- **向量化**：ONNX Runtime 本地 embedding（`onnx_embeddings.py`，基于 ONNXMiniLM_L6_V2，SHA256 校验模型文件），可选外部 API / sentence-transformers
- **可选能力**：OAuth 2.0 + DCR、HTTPS/mDNS、Streamable HTTP MCP、SSE 事件总线

### 核心模块/目录结构（src/mcp_memory_service/）
```
src/mcp_memory_service/
├── web/                # FastAPI 应用：app.py + api/ 下 17+ 路由（memories/search/documents/quality/
│                       #   consolidation/backup/analytics/harvest/mcp/oauth_status/conflicts…）
├── server/             # MCP server 层：__main__.py、handlers/（memory/graph/documents/quality/
│                       #   consolidation/utility）、cache_manager、client_detection、environment
├── storage/            # base/factory + sqlite_vec.py（本地向量检索）、graph.py、hybrid.py、
│                       #   cloudflare.py、http_client.py、migrations/
├── embeddings/         # onnx_embeddings.py（本地 ONNX）、external_api.py
├── consolidation/      # consolidator.py + decay/compression/clustering/associations/forgetting/
│                       #   relationship_inference/health/scheduler（"梦境式"记忆整合）
├── harvest/            # 记忆收割：classifier/parser/extractor/harvester
├── services/memory_service.py
├── quality/ reasoning/ sync/ backup/ discovery/ ingestion/
├── mcp_server.py       # FastAPI 原生 MCP（Streamable HTTP），22 个核心记忆操作
└── models/memory.py    # Memory 数据模型
```

### 架构亮点与关键技术点
1. **本地优先 + 零云依赖**：`storage/sqlite_vec.py` 用 sqlite-vec 扩展做向量相似检索，`embeddings/onnx_embeddings.py` 用 ONNX Runtime 本地跑 embedding（显式 `WANDB_DISABLED` 规避 protobuf 冲突、SHA256 校验模型文件防投毒），记忆数据不出内网——对比 Mem0/Zep 商业方案的私有化卖点。
2. **多智能体共享设计**：`X-Agent-ID` 请求头自动给记忆打 `agent:<id>` 标签实现按智能体作用域检索；`conversation_id` 绕过去重做增量会话存储；SSE 实时广播任一智能体存/删记忆事件（`web/sse.py`），支持"tag 即消息总线"的智能体间通信模式。
3. **自动记忆整合（consolidation）**：`consolidation/` 模块化实现指数衰减（decay）、语义压缩（compression）、聚类（clustering）、创意关联（associations）、受控遗忘（forgetting）、关系推断（relationship_inference）+ 调度器（scheduler）与健康监控（health），类比人类"睡眠巩固记忆"，解决上下文窗口遗忘问题。
4. **双协议访问**：REST API（15+ 端点，任何 HTTP client 可对接 LangGraph/CrewAI/AutoGen）+ 原生 MCP（FastMCP Streamable HTTP，远程 claude.ai 可连，含 OAuth 2.0 + DCR 企业认证）。
5. **知识图谱 + 类型化边**：`storage/graph.py` + typed edges（causes/fixes/contradicts），智能体共享的不只是事实还有因果链；`models/memory.py` 记忆模型统一承载。
6. **混合检索**：`storage/hybrid.py` BM25 + 向量混合检索，兼顾关键词精确匹配与语义召回。

### T6 答辩素材点
- **多智能体记忆体系**：共享记忆 + 图谱 + 因果边 + 智能体作用域标签，直击多智能体协作的"记忆孤岛"痛点，架构叙事完整。
- **自主记忆管理**：衰减/压缩/聚类/遗忘的自动整合机制，体现"系统自维护"的高级设计。
- **私有化部署方案**：本地 ONNX embedding + sqlite-vec + 零云依赖，满足数据不出内网的安全诉求，有明确商业对比（Mem0/Zep）。
- **协议与生态覆盖**：REST + MCP + OAuth + SSE 多协议全栈，兼容主流 Agent 框架，展示平台化思维。

---

## 附：五项目横评（答辩可用的"一盘棋"叙事）

| 维度 | loongnix_file_server | dify-on-dingding | emu_autotest | mcp-lightrag | mcp-memory-service |
|---|---|---|---|---|---|
| 定位 | 文件分发中间层 | 对话机器人集成网关 | 测试环境恢复自动化 | RAG 知识图谱 MCP 化 | 多智能体共享记忆 |
| 核心亮点 | 签名链接+安全校验 | 流式卡片+SSE 解析 | GUI 自动化+CGI 直连 | 智能 upsert 同步 | 本地向量+记忆整合 |
| 技术共性 | Flask/Python | Python 异步+流式 | Python 进程编排 | MCP 协议 | FastAPI+MCP |
| 服务对象 | 模型推理产物 | 内部员工 | 测试团队 | AI Agent | AI Agent 集群 |

**横评叙事建议**：五个项目覆盖"文件分发 → 对话集成 → 测试自动化 → RAG 服务化 → 多智能体记忆"的完整 AI 工程链路，可提炼为"从单点工具到 AI 基础设施"的成长主线，适合作为 T6 答辩的整体框架。
