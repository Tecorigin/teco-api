# 四个项目架构资料：tecode / tecotest / official-website / teco-skill-hub

> 用途：员工 T6 晋升答辩素材整理
> 整理日期：2026-08-19
> 说明：以下内容均基于仓库实际代码/文档探查所得，关键点均附文件锚点，可进一步展开核实。

---

## 一、tecode —— Tecorigin 效能融合平台

### 一句话定位
面向研发/测试团队的工程效能融合平台，把 Gerrit、Jenkins、Jira、GitLab、Confluence、禅道、钉钉、K8s、Prometheus 等工具链统一收口，提供需求流转、任务调度、测试用例、AI 辅助、统计看板等一站式能力。

### 技术栈
- 后端：Go 1.19 + Gin 1.7.4 + GORM + MySQL（多实例）+ MongoDB + RabbitMQ + k8s client-go + go-ldap（`backend/go.mod`）
- 前端：Vue2 + element-ui + echarts + axios + tinymce + relation-graph + gojs（`front/package.json`）
- 部署：单镜像 Docker（nginx + Go 一体，`deploy/single-image/`），或裸机 nginx 反代（README.md）

### 核心模块/目录结构（backend/）
- `main.go`：启动入口 + `startScheduledTask()` 三档定时任务
- `Router/router.go`（771 行）：路由注册与 `CheckPermission` 权限中间件（约 427 条路由，POST 为主）
- `Controllers/`（18 个子模块）：AI、Forum、Gerrit、Jenkins、MyJob、Order、Performance、Resource、Schedule、Statistics、System、TecoApi、TestingCase、ThirdParty 等，统一经 `Controllers/enter.go` 声明实例
- `Services/`（40+ 服务）：DingTalkService、GerritService、JiraTaskService、JenkinsTaskService、K8sService、RabbitMqConsumer、AiGraphService、AiSkillSearchService、TestingSdaartService、T2ScheduleService、PrometheusService、SoftStackService 等
- `Models/`（50+ 模型）：含 GerritChangeModel、JenkinsTaskModel、K8sScheduleResult、SdaartTask、SysPermissionModel 等
- `DataPool/`：Mysql/Mysql2.0、MongoDB、RabbitMq、ReviewMq、Spug、K8sTask（多集群：k8s/k8s2/k8s3/郑州，见 k8s*.yaml）
- `Task/Cron.go`、`Middlewares/corsMiddleware.go`、`Sessions/session.go`、`Common/Response|Constant`

### 架构亮点与关键技术点
1. **多工具链深度集成**：一套后端对接 8+ 外部系统——Gerrit（changes/comments/review 拉取）、Jenkins（多实例：jenkins_ci/ct/t2/k8s）、Jira（含 AS 售后单轮询提醒 `AlertAsJira()`）、GitLab、Confluence、禅道、钉钉（`oapi.dingtalk.com` 消息/审批）、K8s 多集群调度（nginx 里 `/schedule`、`/k2schedule`、`/k3schedule` 三个集群反代）。
2. **多级异步任务体系**：`main.go` 中 `startScheduledTask()` 用三个 ticker（5 分钟/60 分钟/2 小时）跑 `RefreshEfficiencyOrder`、`SyncJenkinsComputers`、`RefreshLdapUser`（LDAP 用户同步）；配合 `RabbitMqConsumer.go` 的 RabbitMQ 消费端（`ExchangeDeclare("mobile_job","direct")`），断线 10 秒自动重连，实现评审结果等消息的异步落库（`SaveMongo` 写 MongoDB）。
3. **AI 能力内建**：`AiGraphService.go` 定义 OpenAI 兼容请求结构（`openaiRequest`/`openaiMessage`），支持流式图表生成、对话历史、分享链接（`ShareId`）；另有 `AiSkillSearchService`（技能检索）、`AiAgentService`、AI 控制器与前端 `views/ai` 页，形成"AI 辅助效能"闭环。
4. **统一权限模型**：`Router/router.go` 的 `CheckPermission(permission)` 中间件——Cookie `x-token` → Session 反序列化 → 超级管理员直通 / `GetPermission` 逐权限校验，所有受控路由统一挂载，前端 `permission.js` + vuex 联动。
5. **测试用例对接硬件测试**：`TestingCase/Tecoal.go`、`TestingSdaartService.go` 直接对接算子测试结果（sdaaEvent 性能数据：bw/cardtime/cputime），`TestingCompareMethod.QueryByComponent()` 支持按组件（SDAART 等）配置对比方法，打通"硬件测试数据 → 效能平台"链路。
6. **单镜像交付**：Dockerfile 一次构建前后端，容器内 nginx 托管静态资源并反代 `/api`，配置通过宿主机挂载注入（config.yaml、k8s*.yaml），环境切换无需重新构建镜像。

### T6 答辩可用素材点（一句话表述）
- **平台化集成能力**：将 Gerrit/Jenkins/Jira/GitLab/禅道/Confluence/钉钉/K8s 8+ 工具链统一收口为单一效能平台，显著降低跨系统切换成本。
- **异步任务与消息架构**：三档定时任务 + RabbitMQ 断线自愈消费端 + MongoDB 结果落库，支撑大规模自动化任务稳定运行。
- **AI 原生集成**：以 OpenAI 兼容协议自建 AI 图表/技能检索/Agent 服务，把 AI 能力沉淀进日常研发流程。
- **统一权限与安全**：基于 gin 中间件 + Session + 角色权限模型的统一鉴权，覆盖全量路由，具备可扩展性。
- **多 K8s 集群调度**：单平台调度郑州/太初多套 K8s 集群与 Jenkins 节点同步（`SyncJenkinsComputers`），体现跨集群资源编排能力。
- **端到端数据闭环**：从工具链数据采集（Gerrit/Jenkins）→ 消息异步处理 → 效能统计看板，形成可量化研发效能度量。

---

## 二、tecotest —— 太初 AI 芯片算子自动化测试框架

### 一句话定位
面向太初自研 AI 芯片（SWAI）的算子级自动化测试框架：以 torch/paddle 在 GPU/CPU 上生成的 golden 结果为标准，验证 tecodnn/tecoblas/tecocustom 算子的正确性与性能。

### 技术栈
- C++14 + GoogleTest + CMake（`CMakeLists.txt`）
- 用例描述：protobuf（`test_proto/optest.pb.h`），text_format 解析
- 设备侧：SDAA（国产，`/opt/tecoai/sdaa`）与 CUDA 双后端
- 辅助：nlohmann/json、Python 脚本（用例生成/CI/日志对比）

### 核心模块/目录结构
- `main.cpp`：入口，初始化全局变量 → 版本检查 → GoogleTest 环境（`TestEnvironment`）
- `src/case/`：`parser`（解析用例 protobuf → 内部数据结构）、`evaluator`（判定通过/失败，输出 error、带宽、计算效率）、`criterion`（判定公式与阈值）、`case_collector`、`pb_tools`
- `src/device/`：`cuda/` 与 `sdaa/` 对称实现（memory/profiler/version），编译期按目标切换
- `src/suite/`：`env`、`executor`、`suite`
- `src/common/`：half/bf16 类型、md5、tecolog、tools
- `zoo/`：dnn / blas / custom 三大算子用例库 + `autoGenTecotestCode.py` 用例代码自动生成
- `gen_case/`：专项用例生成器（如 `1.19.0/slice_gen_cases.py`、`topkex_gen_cases.py`）
- `test_lib/`：设备端库（`hash_calc.swcu`、`check_hbm.swcu`、`ldm_init.swcu`，接口见 `tecotest_slave.h`）
- `tools/`：`diff_log.py`、`diff_hash.py`、`deduplicate_cases.py`、`ci_function_test.py`、`ci_perf_test.py`、`daily_unite_test.py`、`differ_perf.py` 等

### 架构亮点与关键技术点
1. **双后端设备抽象**：`src/device/cuda` 与 `src/device/sdaa` 提供对称的 memory/profiler/version 实现，`CMakeLists.txt` 按构建目标选择 SDAA/CUDA 库（`target_link_libraries(... tecodnn/tecoblas/tecocustom)` vs `cudnn/cublas/cupti`），同一套用例与判定代码跨 GPU/国产芯片复用。
2. **protobuf 驱动的用例描述与解析**：`test_proto/optest.pb.h` 定义用例格式，`parser.h` 支持多种取值来源（`VALUE_F/I/L/H/BF/RANDOM/PATH`），用例文件与代码解耦，可脱离源码新增用例。
3. **多公式多阈值判定体系**：`criterion.h` 定义 `Formula` 枚举（DIFF1~DIFF4、MAPE、MAE、MSE_RELA、DIFF_ABS 等）与 `GOLDEN_THRESHOLD_FLOAT=1e-6` 等常量；`evaluator` 同时评估数值误差、带宽（io efficiency）、算力利用率（compute efficiency），正确性+性能双维度判定。
4. **设备端哈希校验**：`test_lib` 以 .swcu（SDAA 设备端代码）实现 `calc_hash`/`check_hbm`/`init_ldm`，用硬件哈希快速比对结果，降低大数据量比对开销。
5. **自动化配套完善**：`build.sh` 支持 `--build dnn/blas/custom -f op` 定向编译；`zoo/autoGenTecotestCode.py` 从 `dnn/ops` 自动生成用例目录骨架；`gen_case/` 按算子特性（slice/topkex）生成针对性用例；`tools/` 提供日志/哈希对比与去重。
6. **2.0.0 关键增强**（README 更新说明）：支持保存不稳定现场、clip 及原位测试；`sdaaEvent` 方法获取性能数据；autotune online 模式测试；优化内存分配以高效处理大量输入输出张量。

### T6 答辩可用素材点（一句话表述）
- **跨硬件测试抽象**：以 CUDA/SDAA 对称设备抽象 + 同一套用例与判定逻辑，实现 GPU golden 与国产芯片结果的可比验证，是芯片验证体系的关键一环。
- **用例与代码解耦**：protobuf 用例格式 + 多公式阈值判定（DIFF/MAPE/MSE 等），支撑 dnn/blas/custom 大规模算子回归。
- **低开销结果比对**：设备端 .swcu 哈希校验（calc_hash/check_hbm），解决大数据量结果比对性能瓶颈。
- **自动化与 CI 配套**：用例自动生成 + ci_function/ci_perf/daily_unite 测试脚本，形成可持续运行的算子质量流水线。
- **性能与正确性双维度评估**：evaluator 综合误差、带宽、算力效率，输出可量化的硬件/算子性能画像。

---

## 三、official-website —— 太初官网（多语言门户）

### 一句话定位
公司对外官网门户：基于 ASP.NET WebForms 的存量系统 + Python 代理旁路服务，提供多语言内容发布、栏目管理、后台权限与审计、外部平台触达等能力，并已沉淀成体系的技术亮点/难点文档。

### 技术栈
- 主站：ASP.NET WebForms（.NET Framework 4.6.1，`web.config` `targetFramework="4.6.1"`）+ SQLite（`Lb.Orm/DataBase`）
- 重写：IIS URL Rewrite（`web.config` rewrite rules）+ 历史 ISAPI Rewrite（`ISAPI/httpd.ini`）双套
- 旁路：Python Flask + BeautifulSoup（`proxy/main.py`）
- 第三方：NPOI（Excel/docx）、阿里云短信 SDK、微信 SDK（回调/网页授权）、百度统计 API

### 核心模块/目录结构
- `aspx/`：前台页面（`CN/`、`EN/` 双语言目录），大量 `Lb:Drepeater` 自定义控件
- `sysaspx/`：后台管理（登录 `Lblogin.aspx.cs`、`chkLogin.aspx.cs`、微信回调 `wxhandle.aspx.cs`、`wxOpenId.ashx`）
- `app_code/`：核心工具类（`ExcelHelper.cs`、`NpoiDocHelper.cs`、`FileDownHelper.cs`、`AliyunSendMess.cs`、`Lbtongji.cs`、`Mysafe.cs`）
- `Lb/`：自定义控件库（数据驱动模板）
- `sysconfig/`：配置中心（`urls.config`、`Module.xml`、`SeoConfig.xml`、`SiteInfo.xml` 等）
- `proxy/`：Python 代理服务（`main.py` + `start_service.sh`）
- `docs/`：13 篇技术亮点/难点文档（`00-技术亮点难点总览索引.md` 起），已可作答辩/汇报直接引用

### 架构亮点与关键技术点
1. **经典 WebForms 与伪静态融合**：`.html` 结尾 + 多层级栏目 + 分页参数映射（web.config rewrite 规则），保留老系统稳定性的同时获得 SEO 友好的对外 URL。
2. **数据驱动模板与自定义控件**：`Lb:Drepeater` 在页面内声明 `TableName/Where/OrderBy/FieldName`，配合 `{Lb_News.News_Title}` 占位符，运营改数据即可改内容，低代码化内容编排（`aspx/CN/index.aspx`）。
3. **多语言与版本切换**：CN/EN 双目录 + `sysconfig/urls.config`（`langflag`/`defaultpage`）+ 角色语言范围（`Role_CateLang`），登录后写 Session/Cookie（`LangFlag`、`LangName`）。
4. **后台鉴权与审计**：验证码 + 登录失败 5 次阻断 + 登录日志 `Lb_AdminLog` + 超级管理员/角色管理员双路径（`Lb/Lblogin.aspx.cs`）。
5. **内容流转链路**：`ExcelHelper`（NPOI 导入导出）、`NpoiDocHelper`（docx 解析 + 图片抽取落盘 `/upload/wordimages/yyyy/MM/`）、`FileDownHelper`（分块下载 `DownLoad`、断点续传与限速 `ResponseFile`、`HtmlToWord` 导出）。
6. **混合架构与 Python 代理**：`proxy/main.py` 提供 `/officer_proxy/wx_article_content`，用 BeautifulSoup 对微信/新华报业/经济日报等多来源做脚本剔除、相对路径转绝对路径，把高变更抓取逻辑旁路出主站。
7. **外部平台集成闭环**：阿里云短信（模板验证码、国内/港澳台）、微信（回调 `wxhandle.aspx.cs` + OpenID `wxOpenId.ashx`）、邮件（SMTP）、百度统计（`Lbtongji.cs`），覆盖"内容发布→触达→统计反馈"。
8. **风险治理有沉淀**：docs/ 已整理数据访问 SQL 拼接风险、凭据管理、遗留系统安全治理、可观测性等难点及 P0/P1 治理优先级，体现系统化治理思路。

### T6 答辩可用素材点（一句话表述）
- **存量系统现代化治理**：在不重构前提下通过伪静态重写、数据驱动模板、Python 旁路服务让老官网保持 SEO 友好与可扩展，展示"低成本演进"工程判断。
- **低代码内容编排**：自定义控件 + 数据驱动模板支撑运营自助改版，减少前端开发介入，沉淀为可复用组件库。
- **混合架构设计**：.NET 主站 + Flask 代理旁路解耦高变更抓取逻辑，给出跨技术栈协同的架构取舍实例。
- **大文件传输优化**：分块下载 + 断点续传 + 限速（`FileDownHelper.cs`），解决内容站大文件交付稳定性问题。
- **安全与可观测治理**：主动梳理 SQL 拼接、凭据暴露等遗留风险并给出分级治理路线（docs/ 文档体系），体现风险识别与工程治理能力。

---

## 四、teco-skill-hub —— 公司级 CoPaw Skill 中央仓库与分发平台

### 一句话定位
太初元碁内部 CoPaw Skill 生态的中央仓库与分发平台：统一规范、注册、分发 105+ 个 Skill 与 MCP、聚合知识库，并提供 Web 门户浏览与投稿审核流。

### 技术栈
- 数据层：**无数据库**，"文件即数据"，Git 仓库即真相源
- 前端：纯静态 HTML/JS/CSS（`docs/index.html` + `docs/app.js`）
- 后端：轻量 Python HTTP 服务（`scripts/local_registry_server.py`）
- 工具链：Node 脚本（`scripts/generate-index.js`）+ Python 脚本（`scripts/auto_sync_index.py`）+ Shell/Bat（`setup-local-registry.sh/.bat`）
- 集成：CoPaw/OpenClaw CLI（`copaw skill add/install`）、GitLab Registry URL

### 核心模块/目录结构
- `skills/`：105 个 Skill（dingtalk-meeting-creator、teco-* 文档生成系列、health-check 系列、k8s-ops-skill-py、tccl-check、multi-node-vllm 等）
- `mcps/`：4 个 MCP（cmdb-helper、docs-map-server、lightrag-query、teco-knowledge-graph）
- `knowledge/`：知识库（basic/advanced/platform/archive 分类）
- `docs/`：Web 门户（含 tecoapi-leaderboard 数据）
- `scripts/`：注册表生成/同步/本地服务
- `index.json`：Skill 注册表（name/displayName/description/version/author/tags/category/path/url/downloads）
- `review-submissions/`：投稿待审核区（pending/）
- `SKILL_SPEC.md`：Skill 编写规范；`skill-hub.md`：平台手册（1049 行，含总体架构 mermaid 图）

### 架构亮点与关键技术点
1. **"文件即数据、仓库即真相源"设计**：Skill/MCP/知识/投稿全部以文件落盘，Python 服务只做扫描聚合转发（无状态），避免引入数据库与框架后端，维护成本极低（`skill-hub.md` 3.1 设计原则）。
2. **统一 Skill 规范**：`SKILL_SPEC.md` 强制 SKILL.md frontmatter（name/description/author/version/tags/category）+ 目录结构约定，保证 105 个 Skill 可被 CoPaw 统一导入，解决"官方 MCP 只能以 API 归属人开会"等实际痛点（dingtalk-meeting-creator 即为例）。
3. **注册表自动化流水线**：`generate-index.js` 扫描 skills/ 目录从 frontmatter 生成 `index.json`；`auto_sync_index.py` 实现 git pull → 检测缺失 skill → 从 SKILL.md 提取描述、从 git log 提取作者（AUTHOR_MAP 映射中文名）→ 补全 index.json → commit & push，全自动同步。
4. **三种分发方式**：GitLab Registry URL（推荐）、本地 registry（`setup-local-registry.sh` 起 8000 端口，适合内网/离线）、直接 URL 导入——适配不同网络环境。
5. **Web 门户 + 投稿审核流**：前端调用 `/api/skills`、`/api/mcps`、`/api/knowledge`、`/api/markdown` 等接口；投稿 ZIP/Markdown 校验后落 `review-submissions/pending/<target>/<timestamp>-<slug>/` 并生成 metadata.json，管理员审核后移入正式目录。
6. **生态覆盖广**：skills 覆盖文档生成（teco-vllm/megatron/small-model 系列）、CI/CD（build-publish、package_build）、运维巡检（jenkins/gerrit/tecode/hostmanager health-check）、测试（driver_*_dailytest）、办公协作（dingtalk/ldap/gitlab/gerrit sync），并配 MCP 分发与知识库聚合，形成完整平台。

### T6 答辩可用素材点（一句话表述）
- **从 0 到 1 搭建平台**：牵头搭建公司级 Skill 分发平台（105+ Skill、4 MCP、知识库、Web 门户），构建公司 AI Agent 能力生态。
- **无数据库的轻量架构**：以"文件即数据 + Git 即真相源 + 无状态 Python 服务"支撑全平台，体现极简架构设计能力。
- **自动化运维流水线**：index.json 自动生成与同步（git log 作者映射、缺失检测、自动 commit&push），平台内容零人工维护成本。
- **规范先行**：制定 SKILL_SPEC.md 统一规范并落地为强制结构，保障多作者协作下 105 个 Skill 的质量与可导入性。
- **多环境分发方案**：GitLab/本地/URL 三种分发模式适配内外网与离线场景，扩展性强。

---

## 附：四个项目横向对比（答辩可强调的共性主线）

| 项目 | 角色 | 共性主线 |
|---|---|---|
| tecode | 效能平台（收口） | 把工具链/AI/数据统一成产品化平台 |
| tecotest | 芯片验证（底座） | 面向国产硬件的自动化测试工程化 |
| official-website | 对外门户（存量治理） | 遗留系统低成本现代化与治理 |
| teco-skill-hub | AI 生态（赋能） | 平台化分发 AI Agent 能力，反哺前两者（tecode-health-check、jenkins-node-manager 等 Skill 即运维 tecode 体系） |

> 建议主线：**"从芯片验证（tecotest）到效能平台（tecode）再到 AI 生态（teco-skill-hub）"**，official-website 作为存量系统治理的代表作补充工程判断力维度。
