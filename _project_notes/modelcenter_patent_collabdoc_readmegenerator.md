# 四项目架构资料（T6 晋升答辩素材）

> 整理日期：2026-08-19
> 素材来源：`/root/zhaoling/workspace` 下 teco-modelcenter、teco-patent、teccollabdoc、large_model_readme_generator 源码与 README 深读。
> 用途：员工 T6 晋升答辩架构梳理。

---

## 一、teco-modelcenter（太初元碁模型中心）

### 一句话定位
基于 Next.js 的模型手册门户，服务端直接扫描外部 Markdown 手册目录，公开展示模型列表，登录授权后查看完整文档，并支持申请、审核、密码找回与后台管理。

### 技术栈
- Next.js 16.2.6（App Router / Route Handlers）+ React 19 + TypeScript
- MySQL（mysql2）+ bcryptjs（密码哈希）
- Nodemailer（SMTP 邮件：审核通知 / 激活 / 找回密码）
- react-markdown + remark-gfm + rehype-highlight + highlight.js（Markdown 渲染）
- Docker（Next.js standalone 生产镜像）

### 核心模块 / 目录结构（简要）
```
teco-modelcenter/
├── app/
│   ├── page.tsx                     # 首页：模型列表 + 搜索筛选
│   ├── models/[slug]/page.tsx       # 模型详情页（登录后可看全文）
│   ├── login|apply|activate|forgot-password|reset-password/   # 认证流程页
│   ├── ops/console-7hK29mQa4p/admin # 混淆路径的后台入口
│   └── api/
│       ├── auth/*                   # login/apply/activate/request-reset/reset-password/logout
│       ├── admin/*                  # 用户管理、申请 approve/reject、重置密码、toggle-admin
│       ├── model-doc/route.ts       # 文档路径解析 + 307 重定向到详情
│       └── model-asset/route.ts     # 模型资源接口
├── lib/
│   ├── catalog.ts                   # 核心：目录扫描 / 摘要提取 / 领域标签推断 / 缓存
│   ├── auth.ts                      # 会话管理（token 哈希入库 + Cookie）
│   ├── db.ts / user-service.ts / email.ts / admin-path.ts / types.ts
├── scripts/
│   ├── import-users.mjs             # 旧用户 CSV 批量导入
│   └── get_user_list.py / spder_old_data.py
└── components/                      # detail-markdown、catalog-page、auth-forms、admin-console 等
```

### 架构亮点与关键技术点（含代码/文件锚点）
1. **文件系统即内容源（零录入文档治理）**
   - 服务端直接读取 `MODEL_README_ROOT`（即 large_model_readme_generator 产出的 modelreadme 目录），配合 `model-index.json` + `model-categories.json` 元数据（见 `lib/catalog.ts` 的 `getModelIndex()` / `getCategoryConfig()`），数据库只管用户、会话与统计，文档内容永不重复录入。
2. **智能摘要提取与噪音过滤**
   - `extractSummary()`：取 Markdown 首节正文行，经多层过滤（代码块、`url=`、`commit_id=`、命令行开头、列表、表格、`< >` 等），取第一条 ≥24 字符且含中文/英文的行，截断 120 字符；`isGenericTitle()` / `isNoiseTitle()` 过滤“概述 / 简介 / run scripts params”等无效标题。
3. **规则引擎推断元数据**
   - `inferDomain()`：9 大领域正则（大模型 / CV / NLP / 语音 / 推荐 / 强化学习 / 时序 / 图神经 / 其他）；
   - `inferScale()`：OS→硬件架构映射表（Ubuntu22.04→Intel/AMD、Kylin→海光、Loongnix→龙芯、UOS→申威 8A 等）；
   - `inferMode()`：task→训练模式（推理 / 微调_SFT / 微调_LoRA / 预训练）；
   - `TAG_PATTERNS`：27+ 标签正则（目标检测、OCR、人脸识别、语音合成…）。
4. **模型身份键与版本去重**
   - `getModelLineageKey()` / `getModelIdentityKey()`：NFKC 归一化 + 去分隔符，形成“领域|框架|任务|标题|OS”唯一身份；`getCatalog()` 按身份键只保留最新版本（`compareVersions` 数值语义排序）；slug 用 `sha1(relativePath)` 前 8 位保证稳定唯一（`createSlug()`）。
5. **分层 TTL 缓存**
   - catalog / categoryConfig / modelIndex 三级内存缓存，`CACHE_TTL_MS = 60_000`，避免每次请求全盘扫描。
6. **安全会话设计**
   - `lib/auth.ts`：随机 32 字节 token，库中只存 `sha256(token)`（`hashSessionToken`）；httpOnly + sameSite=lax Cookie；`shouldUseSecureCookie()` 按 BASE_URL 协议 / 显式配置自适应 Secure 标志；会话表带过期时间。
7. **邮件流程自动化 + 降级**
   - 申请→管理员邮件通知→审核→激活邮件→自动登录闭环；SMTP 未配置时邮件降级输出到服务端日志（便于内网联调）。
8. **生产可部署性**
   - Dockerfile 采用 Next.js standalone 产物，构建/运行职责分离；Nginx 反代 + PM2 部署文档完整；`import-users.mjs` 支持旧用户 CSV 导入（用户名自动生成、冲突后缀、邮箱/手机号去重、导入报告）。

### 对 T6 晋升答辩可用的素材点
- 「文件系统即内容源」：文档零录入治理，配合 JSON 索引与规则推断，直接扫描目录即可发布模型门户。
- 规则引擎 + 归一化身份键，自动完成多版本/多架构/多任务文档的归类、去重与稳定 slug。
- 面向内网的轻量安全体系：bcrypt 密码 + 会话 token 哈希入库 + 混淆后台路径 + 邮件激活链路。
- 60s TTL 三级缓存的目录扫描性能设计，支撑大目录即时检索。
- 完整生产链路：standalone 镜像、PM2/Nginx 部署、旧用户 CSV 导入，体现工程化落地能力。

---

## 二、teco-patent（专利审核系统）

### 一句话定位
前后端分离的专利全流程管理系统，覆盖专利申报审核、IP 分类、AI 辅助新颖性查新、竞对专利库（向量检索）、钉钉/站内通知与登录审计。

### 技术栈
- 后端：Go + Gin + GORM v2 + MySQL
- 前端：Vue 3 + Vite + Element Plus（node-forge 纯 JS 加密）
- 向量库：Qdrant（1024 维 embedding）
- LDAP 认证、JWT、RSA+AES-GCM 混合加密、钉钉 API、智慧芽 API、Tavily API、LLM（查新判定）

### 核心模块 / 目录结构（简要）
```
teco-patent/
├── backend/
│   ├── cmd/server/main.go                    # 入口
│   ├── internal/
│   │   ├── router/router.go                  # Gin 路由装配（注入全部 service）
│   │   ├── handler/                          # patent/novelty/auth/crypto/announcement/course/ip_category…
│   │   ├── service/
│   │   │   ├── transport_crypto_service.go   # RSA+AES-GCM 传输加密
│   │   │   ├── novelty_service.go            # AI 查新流水线（向量召回+LLM 判定）
│   │   │   ├── qdrant_novelty_store.go       # Qdrant 向量存取
│   │   │   ├── auth_service.go               # LDAP + JWT + 登录失败锁定 + 钉钉免密
│   │   │   ├── patent_reminder_scheduler.go  # 每日待办提醒
│   │   │   └── dingtalk/notification/login_log/dashboard/project/course…
│   │   ├── model/                            # 30+ GORM 表（patent/user/ip_category/competitor_patent/patent_semantic_embedding…）
│   │   └── middleware/                       # 加密中间件、CORS、RequestLogger、Recovery
│   └── pkg/response/
└── frontend/                                 # Vue3 视图（Dashboard/CompetitorPatentLibrary/IPCategory…）
```

### 架构亮点与关键技术点（含代码/文件锚点）
1. **RSA+AES 混合传输加密（内网 http 环境可用）**
   - `transport_crypto_service.go`：启动生成 2048 位 RSA 密钥，前端 `GET /api/v1/crypto/public-key` 取公钥 → 生成随机 AES 密钥 RSA-OAEP(SHA256) 加密后 `key-exchange` 换取 sessionId（校验 AES 密钥长度必须 16/24/32）→ 业务请求体用 AES-GCM 加密（nonce 12 字节前置），响应同样加密。
   - 会话存 `sync.Map`，TTL 12h，过期自动清理；加密中间件统一拦截（ping / public-key / key-exchange 为明文接口）。
   - 前端用 node-forge 纯 JS 实现，不依赖 `crypto.subtle`，可在内网 http 协议下运行。
2. **AI 查新三阶段流水线（novelty_service.go）**
   - ① 向量召回：`semanticMatches()` 用 embedding（默认 text-embedding-v4）→ Qdrant 检索 top8（`noveltyCandidateLimit=8`）；
   - ② LLM 判定：`callNoveltyDecision()` 强约束 prompt（防幻觉规则：不得编造专利号/DOI/参数），输出 JSON 三态决策 `OVERLAP / CLEAR / NEED_REVIEW`，对每个候选按「技术问题 / 技术手段 / 技术效果」三维度分别给 overlap+evidence；
   - ③ 结果治理：决策后 `filterLLMMatches()` 过滤、`applyOverlapAnalyses()` 附重叠分析、`redactPlatformMatches()` 对平台内部申请脱敏（标题截断加 `***`、隐藏 ID/申请人）、自动生成智慧芽布尔检索式（`generateChineseSearchExpression()` 保证中文术语）。
3. **竞对专利批量导入任务引擎（断点续传）**
   - `CreateImportTask()` / `processImportTask()`：状态机 PENDING→RUNNING→COMPLETED/FAILED/PAUSED；`ProcessedCount` 进度持久化，20 条/批；支持暂停/继续/删除；服务启动 `ResumeImportTasks()` 自动恢复未完成任务；cancel channel + `sync.Once` 实现优雅取消。
4. **向量账本一致性设计**
   - MySQL `patent_semantic_embedding` 表记录 `source_type + source_id + content_hash(sha256)`，Qdrant 存向量；`embeddingFor()` 按 content hash 幂等跳过重复生成；删除时 Qdrant 与 MySQL 双删；导入失败 `cleanupFailedImport()` 回滚部分索引，保证用户不可见半成品。
5. **智慧芽全文抓取限流器**
   - `zhihuiyaFetchLimiter` 全局互斥锁 + `nextAllowedAt`：20s±3s 随机间隔防封；收到 HTTP 429 时全体暂停 2 分钟再重试；解析分享链接签名参数（patentId/shareId/signature/expire/date/version）调用免费分享接口抓权利要求 + 说明书（`fetchZhihuiyaPatentFullText()`）。
6. **LDAP 双模式认证 + 登录安全**
   - 固定 DN 直连 或 先搜索再 DN 校验两种模式；登录失败 10 分钟内 3 次锁定（`loginFailedWindow/loginFailedLimit`）；登录/登出日志带 `token_jti`；钉钉身份免密登录（browser ticket，TTL 2 分钟）；JWT 鉴权 + 管理员权限分级。
7. **定时任务与通知闭环**
   - `StartDailyPendingReminder()`：每日 9:30 分页扫描待处理专利并派发提醒给当前处理人；每日 12:00 LDAP 用户同步；通知链路支持钉钉（`SendDingTalkMessage` 开关可降级为仅落库）与站内通知。

### 对 T6 晋升答辩可用的素材点
- 自研「RSA-OAEP 密钥交换 + AES-GCM 会话加密 + 中间件透明接入」的传输加密方案，解决内网 http 明文环境的安全传输。
- 「向量召回 + LLM 判定 + 规则约束」的 AI 查新流水线：三维度比对、防幻觉提示工程、结果脱敏、检索式自动生成。
- 可暂停/续传/恢复的批量导入任务状态机，支撑海量竞对专利入库。
- 向量库与数据库一致性账本（content_hash 幂等 + 双删回滚），保障检索数据可靠。
- 外部 API（智慧芽）限流与 429 故障自愈设计，体现工程韧性。

---

## 三、teccollabdoc（文档协作平台 / 文档中心）

### 一句话定位
公司内部文档中心：Django 后端承载文档树、版本发布、审核流、全文检索、API 对比测试、PDF 导出，HocusPocus(Yjs) 提供多人实时协同编辑，MCP 服务把文档能力开放给 LLM 生态。

### 技术栈
- Django 4.1 + DRF + SimpleJWT + django-simpleui
- MySQL（dj-database-url）+ Redis（django-redis 缓存 + Celery broker）
- Celery + django-celery-beat（定时任务）
- Elasticsearch（IK 中文分词）
- HocusPocus（Yjs CRDT 协同，Node.js WS 服务，端口 2233）
- Vue 3 前端（vue-frontend）
- tecodataparser：Node.js 数据处理服务（TipTap 数据解析、PDF 生成）
- ldap3、tree_sitter、python-jenkins、jira、钉钉 API、APScheduler
- FastMCP（mcp/main.py）文档检索 MCP Server

### 核心模块 / 目录结构（简要）
```
teccollabdoc/
├── teccollabdoc/                  # Django 工程（settings：Celery 定时表/JWT/CORS/日志；celery.py；urls.py）
├── backend/
│   ├── views.py / view_sets.py    # DRF API（docView/docDetail/docContent/review/release/search…）
│   ├── models.py                  # Doc、DocReview、SoftVersion、InstallItem、DocViewDate…
│   ├── tasks.py                   # Celery 定时任务（阅读量聚合/用户同步/ES 重建/备份/PDF/通知/链接检查）
│   ├── elasticsearch.py           # ES 索引（ik_max_word 分词、md5 幂等文件 ID）
│   ├── parse_code_api.py          # tree_sitter 解析 C/C++/Python 代码 API
│   ├── api_compare_apscheduler.py # API 用例对比测试
│   ├── doc_pdf.py / doc_link_check.py / backup.py / email.py / dingtalk.py / jenkins.py / ldap_auth.py
│   └── tools/                     # gerrit_helper / jira_helper / tecode_helper / dingtalk_doc_helper / aes_helper / lock
├── hocuspocus-server/server.js    # Yjs 协同编辑服务（Django 为唯一数据源）
├── tecodataparser/                # Node：TipTap 数据解析 + PDF 生成
├── mcp/main.py                    # FastMCP：search_doc / list_doc / get_doc_tree / get_doc_html_content / search_package_vars
└── vue-frontend/                  # 前端
```

### 架构亮点与关键技术点（含代码/文件锚点）
1. **多服务分工协作架构**
   - Django（业务 API）+ HocusPocus（协同 WS）+ tecodataparser（数据处理）+ Vue（前端）+ Celery/Redis（异步）+ ES（检索）六类服务；uwsgi + Nginx 部署，`start.sh` / `celery_restart.sh` / Dockerfile 齐备。
2. **HocusPocus 协同编辑 + 保存状态广播（server.js）**
   - `@hocuspocus/extension-database`：fetch 时带 Bearer token 调 Django `docContent/{id}/` 拉取文档；store 时 PUT 回存，并通过 `document.broadcastStateless()` 向前端实时广播「自动保存中 / 已保存 / 保存失败」状态；`onAuthenticate` 调 Django `token/verify/` 校验 JWT —— 协同服务无独立存储，Django 是唯一事实源。
3. **JWT 双链路认证**
   - SimpleJWT：access 10min + refresh 20min + ROTATE_REFRESH_TOKENS；REST API 与协同 WS 共用同一 token 体系。
4. **Celery 定时任务矩阵（settings.py CELERY_BEAT_SCHEDULE）**
   - 阅读量每 20 分钟落库、每日 3:00 LDAP 用户同步、2:00 ES 重建、1:30 数据备份、2:30/3:00 PDF 批量生成、9:30 审核提醒、9:00 钉钉通知、每 10 分钟 API 用例测试 —— 自动化运维闭环。
5. **阅读量两级存储**
   - 前端计数先进 Redis / `DocViewTemp` 临时表，`serializer_view_data()` 用 groupby 按 doc+date 聚合后批量写入 `DocViewDate`，避免高频写库。
6. **全文检索**
   - `elasticsearch.py`：ES 自定义 mapping（title/content 用 `ik_max_word` 中文分词），`gen_file_id()` 用 `md5(group-version-file_id)` 幂等文档 ID，定时全量同步。
7. **tree_sitter 代码 API 解析**
   - `parse_code_api.py`：编译 C/C++/Python 三语言 parser（`Language.build_library` 缓存 .so），AST 提取类型/函数节点，支撑 API 用例对比测试。
8. **API 对比测试**
   - `api_compare_apscheduler.py`（django-apscheduler）：定时比对接口用例与线上 API 差异，结果落 `ApiTestCompareResult` 表。
9. **MCP 服务化开放**
   - `mcp/main.py`（FastMCP）暴露 `search_doc / list_doc / get_doc_tree / get_doc_html_content / search_package_vars`，文档中心能力可被 AI Agent 生态直接调用。

### 对 T6 晋升答辩可用的素材点
- 多服务协同架构（Django + HocusPocus + Node parser + Vue + Celery + ES），Django 为唯一数据事实源，各服务职责清晰。
- 实时协同编辑与保存状态广播机制（broadcastStateless），JWT 在 REST/WS 双链路复用。
- Celery 定时任务矩阵：阅读量聚合、LDAP 同步、ES 重建、备份、PDF 生成、通知的自动化闭环。
- ES IK 分词 + 幂等文件 ID 的中文全文检索设计。
- 以 MCP 方式把文档检索能力开放给 LLM 生态，体现平台前瞻性。

---

## 四、large_model_readme_generator（大模型手册生成器）

### 一句话定位
把太初各软件栈仓库（Megatron-DeepSpeed / Megatron-LM / vLLM / TecoInference / ModelZoo 等）的内部训练、推理、微调文档与交付制品（Docker 镜像）自动转换为对外用户手册，并产出文档中心数据（docs_data.json）与打包 zip。

### 技术栈
- Python 3（argparse、Jinja2、markdown/html）
- requests（拉取镜像/制品信息）
- 配置驱动：`config.py`（版本 + 镜像 URL + os_dic 架构映射）、`docker_config_*.json`、`docs_data_*.json`
- 产出：`modelreadme/`（按 版本/框架/任务/OS 分层的 Markdown）、upload.zip、model-index.json

### 核心模块 / 目录结构（简要）
```
large_model_readme_generator/
├── config.py                     # 版本与镜像 URL 中心配置（SMALL_MODEL_TRAIN_IMG / os_dic…）
├── large_model_pretrain.py       # Megatron-DeepSpeed 预训练手册
├── large_model_finetune.py       # 微调手册
├── large_model_infer.py          # 大模型推理手册
├── large_model_vllm_infer.py     # teco-vllm 推理手册
├── megatron_model_pretrain.py / megatron_model_finetune.py
├── small_model_train_torch.py / small_model_train_paddle.py / small_model_infer.py / modelzoo_mm_infer.py
├── generate_user_manual.py       # vLLM 大推手册生成（模板化安装章节）
├── search_docker_images.py       # 从交付制品表格自动搜索镜像 → docker_config.json
├── search_vllm_docker_images.py
├── gen_label.py                  # 模型→领域/类别标签映射（label1/label2）
├── gen_zip.py                    # 打包 upload.zip（md→html + 图片 + config）
├── gen_docs_data_v3.2.0.py       # model-index.json → docs_data.json（复用已注册 UUID）
├── model_summary.py / perf_pre_data.py
├── utils.py                      # Markdown 净化 + uuid5 确定性 ID
├── md2html.py                    # Markdown → 带样式 HTML
├── modelreadme/                  # 产出目录（model-index.json + 各版本手册）
└── modelzoo/                     # ModelZoo 源码（Dockerfile-smt、PyTorch/PaddlePaddle/TecoInference…）
```

### 架构亮点与关键技术点（含代码/文件锚点）
1. **配置驱动 + 多场景生成器矩阵**
   - `config.py` 集中管理版本（VERSION）与镜像 URL（含国产 OS 映射 `os_dic`：Ubuntu/8A+UOS/海光 Kylin/龙芯 Loongnix/飞腾 Kylin）；每个场景一个生成器脚本，输入 `--version + --docker-config` 即输出手册，形成可复制的流水线。
2. **交付制品自动发现**
   - `search_docker_images.py`：从太初软件栈交付制品表格中按 OS 标签映射（`OS_LABEL_MAP`：Ubuntu22.04→ubuntu、8A+UOS→8auos、龙芯3C6000→loongarch64…）搜索 11 种镜像类型（light/torch/paddle/tecoinference/vllm/slm_train/llm_train…），产出 `docker_config.json` 供生成器消费，消除手工维护镜像链接。
3. **确定性 UUID 打通下游系统**
   - `utils.generate_uuid_from_name()`：`uuid5(namespace, name+model_type+system)` 生成模型 ID，跨版本/跨 OS 稳定，并追加写入 `uuid.csv` 审计表；`gen_docs_data_v3.2.0.py` 直接复用已注册 UUID（FRAMEWORK_TO_TYPE / OS_TO_DOC_OS 映射），保证文档中心与模型中心数据关联一致。
4. **Markdown 净化流水线**
   - `utils.py`：`remove_markdown_comments` / `remove_demo_comments` / `remove_first_line` / `remove_last_line`，把内部文档（含演示注释、示例占位、首行模板信息）清洗为对外可发布手册。
5. **模板化章节生成 + 平台差异分支**
   - `generate_user_manual.py`：`generate_installation_section()` 按 OS 分支生成「安装推理环境」章节（龙芯无 conda 环境走独立校验命令），`SYSTEM_INFO` 映射架构/系统版本；`load_docker_config()` 校验必需 OS 与字段。
6. **结构化元数据产出（下游门户数据底座）**
   - `model-index.json`（versions/domains/frameworks/tasks/os/tags/models 全维度索引）+ `docs_data.json`（framework→type、OS→doc_os 映射），直接支撑 teco-modelcenter 与文档中心展示。
7. **打包与资源收集**
   - `gen_zip.py`：`get_imgs()/get_img_tar()` 收集图片资源、md→html 转换、config.yaml 模板生成，输出 upload.zip；`gen_label.py` 维护模型→标签映射。
8. **正则驱动的模型清单抽取**
   - `large_model_pretrain.py`：从 run 脚本正则解析 `--model_name` 自动发现模型列表，按 `--master_addr` 判断单机/多机，避免手工枚举。

### 对 T6 晋升答辩可用的素材点
- 面向多版本（v1.3.0~v3.2.x）、多框架（Megatron / vLLM / PyTorch / Paddle / TecoInference）、多国产 OS 的文档生成流水线，配置驱动 + 生成器矩阵。
- 交付制品自动发现（镜像搜索脚本 + OS 标签映射），消除手工维护镜像链接，保证文档与制品一致。
- 确定性 UUID（uuid5）打通模型中心 / 文档中心跨系统数据关联，ID 跨版本稳定。
- Markdown 净化与模板化章节生成，把内部文档安全转换为对外手册。
- 结构化元数据（model-index / docs_data）作为下游门户的数据底座，形成「生成→索引→发布」链路。

---

## 附：四项目之间的协同关系（可作整体叙事素材）

```
large_model_readme_generator ──产出──▶ modelreadme/（Markdown + model-index.json）
        │                                      │ 服务端扫描（MODEL_README_ROOT）
        ▼                                      ▼
docs_data.json ──▶ 文档中心/模型中心数据底座 ──▶ teco-modelcenter（模型门户，登录授权展示）
                                                    │
teccollabdoc（内部文档协作/发布/检索平台）◀──MCP── AI Agent 生态
teco-patent（专利全流程 + AI 查新，独立业务闭环）
```
- 手册生成器产出 → 模型中心直接消费，形成「生成 → 索引 → 发布」的自动化文档链路；
- 文档中心以 MCP 开放检索能力；专利系统独立承担专利业务闭环。
- 答辩时可强调：既有 AI 文档生产流水线（generator + modelcenter），又有企业级文档协作平台（collabdoc），还有 AI 辅助业务系统（patent 查新），覆盖「生产、治理、协作、业务智能化」全链路。
