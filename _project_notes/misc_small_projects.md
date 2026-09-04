# 小项目架构资料（T6 晋升答辩素材整理）

> 调研范围：`/root/zhaoling/workspace` 下 custom_test、agent_test、modelreadme、neo4j、neo4j_test、playwright_test、pytorch_learn、new-api-github（仅对比）。
> 调研日期：2026-08-19。判断基于目录结构、README 与关键文件快览，未读大目录（node_modules/.git/dist 等）。

---

## 1. custom_test

- **一句话定位**：太初 SDAA C 自定义算子开发与验证的测试工程 —— 实现 TensorList 全局范数裁剪（梯度裁剪）算子，手写 kernel 与 TecoCUSTOM 内置 API 双版本对照。
- **技术栈**：SDAA C（`tecocc` 编译器 + SDAARuntime）、TecoCustom 库（`tecocustom.h` / `libtecocustom.so`）、Makefile。
- **核心内容/结构**：
  - `tensorlist_norm.scpp`：手写 `__global__` kernel 版本（平方和 Kernel + 缩放 Kernel，SPE 线程并行、共享内存树形归约），TecoCC 编译。
  - `tensorlist_norm_tecocustom.cpp`：直接调用 `tecocustomFusedNorm()` API 的版本，g++ 编译。
  - `test_tensorlist_norm.scpp`：验证程序（4 个 tensor、7680 元素、阈值裁剪后范数 ≈ 5.0，输出 PASS）。
  - `README.md`：算法流程、编译/运行方法、SDAA C 语法速查（`__global__`、`__local__`、`threadIdx/threadDim`、`sdaaMalloc` 等）、TensorList 数据结构说明、扩展建议（半精度、L1 范数、异步流水线、kernel 融合）。
- **亮点/特点**：双实现对照（手写 kernel vs 厂商 API）；对"梯度裁剪"有清晰业务定位（防大模型训练梯度爆炸）；文档质量高，含 ASCII 流程图与性能扩展建议。
- **T6 素材点**：
  - 芯片底层算子开发能力：SPE 并行、共享内存归约、kernel 启动/同步语义。
  - 优化意识：README 中已列出半精度（halfv16）、kernel 融合、多 stream 异步流水线等优化方向，答辩可展开。
  - 正确的算法定位：把算子映射到大模型训练场景（梯度裁剪）。

---

## 2. agent_test

- **一句话定位**：custom_test 的规范化工程整理版 —— 将算子代码重组为标准算子工程（src/test 分离）并打包，用作 Agent（AI 助手）开发测试的交付样例。
- **技术栈**：SDAA C / TecoCUSTOM（同 custom_test）、shell 构建脚本、tar.gz 打包。
- **核心内容/结构**：
  - `tensorlist_norm_op/`：规范算子工程 —— `src/tensorlist_norm.scpp`（kernel+launcher）、`src/tensorlist_norm.h`（公共接口）、`src/tensorlist_norm_tecocustom.cc`（TecoCustom wrapper 示例）、`test/test_tensorlist_norm.cpp`、`test/build_and_run.sh`（一键构建运行）、`README.md`（英文，接口约定 + 期望输出 PASS 条件）。
  - `custom_test.tar.gz`、`tensorlist_norm_op.tar.gz`：两版代码的打包备份。
- **亮点/特点**：工程规范（接口/实现/测试分离、可复现的一键脚本）；对比 custom_test 可见"从随手测试到规范交付"的整理过程。
- **T6 素材点**：
  - 工程化/交付规范意识：接口抽象（.h）、wrapper 封装、自动化测试脚本。
  - 若答辩中涉及"AI Agent 辅助算子开发"，本目录即为验证样例（Agent 产出的算子工程）。

---

## 3. modelreadme

- **一句话定位**：空目录（仅 `.`/`..`，无任何文件），为模型 README 生成任务的输出/占位目录。
- **技术栈**：无（空目录）。
- **核心内容/结构**：空。相关能力实际落在 workspace 下 `large_model_readme_generator` 项目及各 `teco-*-modelreadme-*` skill 中，本目录疑为生成产物存放位。
- **亮点/特点**：无独立内容。
- **T6 素材点**：无（不作为独立素材；如需讲"模型文档生成"，应引用 large_model_readme_generator 与模型文档相关 skill）。

---

## 4. neo4j

- **一句话定位**：Neo4j 图数据库实例数据目录（Docker 卷挂载结构），承载代码/知识图谱数据，总 11G。
- **技术栈**：Neo4j 2026.03.1（debug.log 版本线索）、Cypher 25、Bolt 7687 / HTTP 7474。
- **核心内容/结构**：
  - `data/`（594M）：`databases/neo4j`（79M，图数据）+ `databases/system` + `transactions`（515M，事务日志，txId 已到 8 万+，说明有长期真实写入）。
  - `import/`（9.5G）：Neo4j 批量导入暂存目录，体量极大，说明做过大规模数据导入。
  - `config/`、`logs/`（2.4M，2026-04-22 起运行）、`plugins/`（空）。
  - 目录结构与 `neo4j_test/docker-compose.yml` 的卷挂载（./data、./logs、./config、./plugins）一一对应，判定为该 compose 启动的实例。
- **亮点/特点**：与 neo4j_test（Joern 代码分析）配套，疑为太初代码属性图（CPG）分析结果落库后的图数据库；import 目录体量表明处理过海量代码/知识数据。
- **T6 素材点**：
  - 知识图谱基础设施搭建与运维（Docker 化部署、批量导入、长期运行）。
  - 与 Agent 系统（如 mcp-lightrag 等）联动时的图谱数据底座能力。
  - 注：库内具体图模型需连库查询确认，答辩引用前建议先导出节点/关系统计。

---

## 5. neo4j_test

- **一句话定位**：Joern（代码分析）+ Neo4j 的代码图谱实验环境 —— 对太初底层软件栈源码生成 Code Property Graph（CPG）并导入图数据库。
- **技术栈**：Joern CLI（Scala/SLF4J 工具链，`joern-cli.zip` 1.7G）、Neo4j（docker-compose，`m.daocloud.io/docker.io/neo4j:latest`）、Shell。
- **核心内容/结构**：
  - `docker-compose.yml`：一键起 Neo4j（7474/7687 端口，neo4j/Teconeo2026!）。
  - `joern-cli.zip` + `joern-install.sh`：Joern 官方安装脚本与离线包。
  - `repos/`（5.7G）：太初软件栈源码 —— `sdaart`、`tccl`、`tecoal`、`tecocc`、`tecocustom`、`tecokernel`、`teco-ai-sys`、`aicard-dev`，以及 **`cpg.bin`（832M，Joern 序列化的 Code Property Graph 产物）**。
- **亮点/特点**：将 Joern 静态分析引入太初自研芯片软件栈，从源码生成代码依赖图谱；cpg.bin 为关键中间产物，可对接 Neo4j 做图谱查询。
- **T6 素材点**：
  - 软件工程智能化探索：代码属性图（CPG）+ 图数据库 + 语义检索，属于"代码智能"前沿方向。
  - 对太初全栈代码（驱动/通信库/编译工具链/自定义算子）的体系化分析能力。
  - 环境工程化：离线安装、docker-compose 一键部署，便于复现与推广。

---

## 6. playwright_test

- **一句话定位**：Playwright 浏览器自动化的截图产物目录（网页设计稿截图 + 元数据 JSON）。
- **技术栈**：Playwright（浏览器自动化），产物为 PNG + JSON。
- **核心内容/结构**：
  - `design-canvas.png`（426K）+ `design-canvas-meta.json`（视口 1428x845、元素 rect 坐标）。
  - `design-canvas-zoomed.png`（752K）+ 对应 meta（缩放视图）。
- **亮点/特点**：仅两个页面截图像素级定位元数据，无脚本源码；判定为自动化测试/页面抓取的一次性输出。
- **T6 素材点**：弱。若答辩涉及"Web 自动化验证"（如对 new-api 前端、官网做回归截图），可作辅助佐证，不宜独立成章。

---

## 7. pytorch_learn

- **一句话定位**：PyTorch 入门学习合集 —— 5 个带详细中文注释的小模型训练脚本，由浅入深。
- **技术栈**：Python、PyTorch（requirements.txt）、MLP/CNN/LSTM/CRNN+CTC，CPU 训练推理。
- **核心内容/结构**：
  - `models/01_mlp_mnist.py`：MLP 手写数字分类。
  - `models/02_cnn_fashion_mnist.py`：CNN FashionMNIST 分类。
  - `models/03_lstm_text_classification.py`：LSTM 文本情感二分类（内置小样本）。
  - `models/04_crnn_ocr_ctc.py`：CRNN+CTC 简化文字识别。
  - `models/05_cpu_train_infer_demo.py`：电商退款风险预测 CLI 工具（数据生成/训练/批量推理，强制 CPU，含示例数据与已训练模型）。
  - `data/`：示例 CSV、模型权重、预测输出。
- **亮点/特点**：覆盖面广（CV/NLP/OCR/表格业务）；注释教学向；05 号脚本是"可落地小工具"而非玩具，模拟真实业务"表格特征+模型服务"。
- **T6 素材点**：
  - 深度学习基础能力佐证（模型选型、训练循环、CPU 推理落地）。
  - 教学/文档能力：可作为新人培养材料。
  - 注意：属学习性质，答辩中宜作"基础能力"辅助，不为主打项目。

---

## 8. new-api-github（仅对比）

- **一句话定位**：**是 new-api 的上游原版** —— 来自 GitHub 的 New API 开源仓库（`module github.com/QuantumNous/new-api`，go 1.25.1，README 与官方多语言一致），用作公司内部改造版（workspace/new-api）的对比基线。
- **技术栈**：Go（relay/middleware/controller/router 分层、OAuth、i18n、Electron 桌面端、Docker）。
- **T6 素材点**：无独立素材；其价值在于 diff 对比 —— 可展示"基于上游开源 LLM Gateway 的二次开发与自研增强"，具体差异点需结合 new-api 主项目梳理。

---

## 汇总表

| 目录 | 性质 | T6 素材价值 |
|------|------|------------|
| custom_test | 太初 SDAA C 算子开发测试（梯度裁剪，双实现） | ★★★ 底层算子能力 |
| agent_test | 算子工程规范化 + Agent 测试样例打包 | ★★ 工程规范/Agent 验证 |
| modelreadme | 空目录（占位） | — 无 |
| neo4j | Neo4j 图数据库实例（11G，docker） | ★★ 图谱基础设施 |
| neo4j_test | Joern+Neo4j 代码图谱实验环境（含 832M CPG） | ★★★ 代码智能探索 |
| playwright_test | 浏览器自动化截图产物 | ★ 辅助佐证 |
| pytorch_learn | PyTorch 学习合集（5 模型） | ★★ 基础能力佐证 |
| new-api-github | new-api 上游原版（对比基线） | — 对比用 |
