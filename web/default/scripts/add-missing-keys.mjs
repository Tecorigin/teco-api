import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
    return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
    en: {
        'Model Performance Statistics': 'Model Performance Statistics',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': 'Compare model-level latency, throughput, success rate and request volume over the selected time range.',
        'Select model': 'Select model',
        'Selected model throughput': 'Selected model throughput',
        'Top models by request volume': 'Top models by request volume',
        'No performance data available for the selected range.': 'No performance data available for the selected range.',
        'Group-level metrics for {{model}}': 'Group-level metrics for {{model}}',
        'No group performance data available for the selected model.': 'No group performance data available for the selected model.',
        'Overall average latency': 'Overall average latency',
        'Overall average success rate': 'Overall average success rate',
        'Model SLO Analytics': 'Model SLO Analytics',
        'Filter by model name': 'Filter by model name',
        'P99 Latency': 'P99 Latency',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': 'No model SLO data available for the selected range.',
        'SLO threshold': 'SLO threshold',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': 'Sort by',
    },
    zh: {
        'Model Performance Statistics': '模型性能统计',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': '对比所选时间范围内各模型的延迟、吞吐、成功率与请求量。',
        'Select model': '选择模型',
        'Selected model throughput': '当前模型吞吐量',
        'Top models by request volume': '按请求量排序的热门模型',
        'No performance data available for the selected range.': '所选时间范围内暂无性能数据。',
        'Group-level metrics for {{model}}': '{{model}} 的分组指标',
        'No group performance data available for the selected model.': '当前所选模型暂无分组性能数据。',
        'Overall average latency': '整体平均延迟',
        'Overall average success rate': '整体平均成功率',
        'Model SLO Analytics': '模型 SLO 统计',
        'Filter by model name': '按模型名筛选',
        'P99 Latency': 'P99 时延',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': '所选时间范围内暂无模型 SLO 数据。',
        'SLO threshold': 'SLO 阈值',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': '排序方式',
    },
    'zh-TW': {
        'Model Performance Statistics': '模型效能統計',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': '比較所選時間範圍內各模型的延遲、吞吐量、成功率與請求量。',
        'Select model': '選擇模型',
        'Selected model throughput': '目前模型吞吐量',
        'Top models by request volume': '按請求量排序的熱門模型',
        'No performance data available for the selected range.': '所選時間範圍內沒有可用的效能資料。',
        'Group-level metrics for {{model}}': '{{model}} 的群組指標',
        'No group performance data available for the selected model.': '目前所選模型沒有可用的群組效能資料。',
        'Overall average latency': '整體平均延遲',
        'Overall average success rate': '整體平均成功率',
        'Model SLO Analytics': '模型 SLO 統計',
        'Filter by model name': '按模型名稱篩選',
        'P99 Latency': 'P99 延遲',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': '所選時間範圍內沒有模型 SLO 資料。',
        'SLO threshold': 'SLO 閾值',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': '排序方式',
    },
    fr: {
        'Model Performance Statistics': 'Statistiques de performance des modèles',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': 'Comparez la latence, le débit, le taux de réussite et le volume de requêtes des modèles sur la période sélectionnée.',
        'Select model': 'Sélectionner un modèle',
        'Selected model throughput': 'Débit du modèle sélectionné',
        'Top models by request volume': 'Modèles les plus sollicités par volume de requêtes',
        'No performance data available for the selected range.': 'Aucune donnée de performance n’est disponible pour la période sélectionnée.',
        'Group-level metrics for {{model}}': 'Métriques par groupe pour {{model}}',
        'No group performance data available for the selected model.': 'Aucune donnée de performance par groupe n’est disponible pour le modèle sélectionné.',
        'Overall average latency': 'Latence moyenne globale',
        'Overall average success rate': 'Taux de réussite moyen global',
        'Model SLO Analytics': 'Analyse SLO des modèles',
        'Filter by model name': 'Filtrer par nom de modèle',
        'P99 Latency': 'Latence P99',
        'P99 TPOT': 'TPOT P99',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': 'Aucune donnée SLO de modèle n’est disponible pour la période sélectionnée.',
        'SLO threshold': 'Seuil SLO',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': 'Trier par',
    },
    ja: {
        'Model Performance Statistics': 'モデル性能統計',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': '選択した期間内で、モデルごとのレイテンシ、スループット、成功率、リクエスト数を比較します。',
        'Select model': 'モデルを選択',
        'Selected model throughput': '選択中モデルのスループット',
        'Top models by request volume': 'リクエスト数上位のモデル',
        'No performance data available for the selected range.': '選択した期間に利用可能な性能データはありません。',
        'Group-level metrics for {{model}}': '{{model}} のグループ別メトリクス',
        'No group performance data available for the selected model.': '選択したモデルに利用可能なグループ別性能データはありません。',
        'Overall average latency': '全体平均レイテンシ',
        'Overall average success rate': '全体平均成功率',
        'Model SLO Analytics': 'モデル SLO 分析',
        'Filter by model name': 'モデル名で絞り込み',
        'P99 Latency': 'P99 レイテンシ',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': '選択した期間に利用可能なモデル SLO データはありません。',
        'SLO threshold': 'SLO 閾値',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': '並び替え',
    },
    ru: {
        'Model Performance Statistics': 'Статистика производительности моделей',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': 'Сравнивайте задержку, пропускную способность, успешность и объём запросов по моделям за выбранный период.',
        'Select model': 'Выбрать модель',
        'Selected model throughput': 'Пропускная способность выбранной модели',
        'Top models by request volume': 'Популярные модели по числу запросов',
        'No performance data available for the selected range.': 'Для выбранного диапазона нет данных о производительности.',
        'Group-level metrics for {{model}}': 'Метрики по группам для {{model}}',
        'No group performance data available for the selected model.': 'Для выбранной модели нет данных о производительности по группам.',
        'Overall average latency': 'Общая средняя задержка',
        'Overall average success rate': 'Общий средний процент успеха',
        'Model SLO Analytics': 'Аналитика SLO моделей',
        'Filter by model name': 'Фильтр по имени модели',
        'P99 Latency': 'Задержка P99',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': 'Для выбранного диапазона нет данных SLO по моделям.',
        'SLO threshold': 'Порог SLO',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} мс',
        'Sort by': 'Сортировать по',
    },
    vi: {
        'Model Performance Statistics': 'Thống kê hiệu năng mô hình',
        'Compare model-level latency, throughput, success rate and request volume over the selected time range.': 'So sánh độ trễ, thông lượng, tỷ lệ thành công và số lượng yêu cầu của từng mô hình trong khoảng thời gian đã chọn.',
        'Select model': 'Chọn mô hình',
        'Selected model throughput': 'Thông lượng của mô hình đã chọn',
        'Top models by request volume': 'Các mô hình hàng đầu theo số lượng yêu cầu',
        'No performance data available for the selected range.': 'Không có dữ liệu hiệu năng cho khoảng thời gian đã chọn.',
        'Group-level metrics for {{model}}': 'Chỉ số theo nhóm cho {{model}}',
        'No group performance data available for the selected model.': 'Không có dữ liệu hiệu năng theo nhóm cho mô hình đã chọn.',
        'Overall average latency': 'Độ trễ trung bình tổng thể',
        'Overall average success rate': 'Tỷ lệ thành công trung bình tổng thể',
        'Model SLO Analytics': 'Phân tích SLO mô hình',
        'Filter by model name': 'Lọc theo tên mô hình',
        'P99 Latency': 'Độ trễ P99',
        'P99 TPOT': 'P99 TPOT',
        'RPM': 'RPM',
        'TPM': 'TPM',
        'SLO': 'SLO',
        'No model SLO data available for the selected range.': 'Không có dữ liệu SLO mô hình cho khoảng thời gian đã chọn.',
        'SLO threshold': 'Ngưỡng SLO',
        'SLO ≤ {{ms}} ms': 'SLO ≤ {{ms}} ms',
        'Sort by': 'Sắp xếp theo',
    },
}

async function main() {
    let totalAdded = 0

    for (const [locale, trans] of Object.entries(newKeys)) {
        const filePath = path.join(LOCALES_DIR, `${locale}.json`)
        const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

        let count = 0
        for (const [key, value] of Object.entries(trans)) {
            if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
                json.translation[key] = value
                count++
            } else if (json.translation[key] !== value) {
                json.translation[key] = value
                count++
            }
        }

        if (count > 0) {
            json.translation = Object.fromEntries(
                Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
            )
            await fs.writeFile(filePath, stableStringify(json), 'utf8')
        }

        console.log(`${locale}: ${count} translations applied`)
        totalAdded += count
    }

    console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => {
    console.error(err)
    process.exitCode = 1
})
