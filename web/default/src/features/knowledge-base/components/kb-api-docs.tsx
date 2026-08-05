/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

const baseExample = 'https://your-new-api.example.com'

export function KbApiDocs() {
  const { t } = useTranslation()

  return (
    <ScrollArea className='h-full pt-2'>
      <div className='mx-auto max-w-4xl space-y-7 pb-6 text-sm'>
        <section className='space-y-3'>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <h3 className='text-base font-semibold'>{t('使用 API Key 调用知识库')}</h3>
            <Button size='sm' render={<Link to='/keys' />}>
              {t('Create API Key')}
            </Button>
          </div>
          <p className='text-muted-foreground'>
            {t('使用本平台创建的 API Key 作为 Bearer Token。接口始终使用该令牌所属用户已开通的知识库空间，无需传递组织或三方服务认证信息。')}
          </p>
          <p className='text-muted-foreground'>
            {t('请先在页面开通知识库；API Token 不能代替开通操作。')}
          </p>
          <p className='text-muted-foreground'>
            {t('点击“Create API Key”前往 API 密钥页面，创建后将 sk- 开头的密钥作为 Bearer Token 传入请求头。')}
          </p>
        </section>

        <ApiSection title={t('认证')}>
          <CodeBlock>{`Authorization: Bearer 你的APIKey
Content-Type: application/json`}</CodeBlock>
        </ApiSection>

        <ApiSection title={t('知识库管理接口')}>
          <p className='text-muted-foreground'>
            {t('以下接口均以 POST 调用，基础路径为 /v1/knowledge。')}
          </p>
          <EndpointTable
            rows={[
              ['/list', t('列出当前令牌所属用户的知识库')],
              ['/create', t('创建知识库，JSON 参数：kb_name、description 及可选解析配置')],
              ['/delete', t('删除知识库，JSON 参数：kb_name')],
              ['/clear', t('清空知识库中的全部文档，JSON 参数：kb_name')],
              ['/list_docs', t('列出文档，JSON 参数：kb_name')],
              ['/delete_docs', t('删除文档，JSON 参数：kb_name、doc_ids')],
              ['/upload_docs', t('上传文档，multipart 参数：kb_name、files；响应返回 task_id')],
              ['/task_status', t('查询上传或索引任务，JSON 参数：task_id')],
              ['/run_index', t('构建 graph 或 raptor 索引，JSON 参数：kb_name、index_type')],
            ]}
          />
          <CodeBlock>{`curl ${baseExample}/v1/knowledge/create \\
  -H "Authorization: Bearer 你的APIKey" \\
  -H "Content-Type: application/json" \\
  -d '{"kb_name":"产品手册","description":"产品文档"}'`}</CodeBlock>
          <CodeBlock>{`curl ${baseExample}/v1/knowledge/upload_docs \\
  -H "Authorization: Bearer 你的APIKey" \\
  -F "kb_name=产品手册" \\
  -F "files=@./产品手册.pdf"`}</CodeBlock>
          <p className='text-muted-foreground'>
            {t('上传为异步任务。请每隔 2 至 8 秒调用 /v1/knowledge/task_status，直到 status 为 completed 或 failed；完成后再用 /list_docs 确认文档 run 为 DONE。')}
          </p>
        </ApiSection>

        <ApiSection title={t('会话与问答接口')}>
          <EndpointTable
            rows={[
              ['/chat/create_session', t('创建会话')],
              ['/chat/list_sessions', t('列出会话')],
              ['/chat/delete_session', t('删除会话，JSON 参数：session_id')],
              ['/chat/session_chat', t('原始 SSE 问答，JSON 参数：query、session_id、stream、return_docs')],
              ['/chat/completions', t('OpenAI 风格问答接口，支持 query 或 messages')],
            ]}
          />
          <CodeBlock>{`curl ${baseExample}/v1/knowledge/chat/completions \\
  -H "Authorization: Bearer 你的APIKey" \\
  -H "Content-Type: application/json" \\
  -d '{
    "knowledge_base":"产品手册",
    "messages":[{"role":"user","content":"产品有哪些核心功能？"}],
    "stream":true,
    "return_docs":true
  }'`}</CodeBlock>
          <p className='text-muted-foreground'>
            {t('问答接口返回 SSE。stream 为 true 时返回 OpenAI 风格的 chat.completion.chunk 事件；stream 为 false 时返回单个 OpenAI 风格 JSON 响应。return_docs 为 true 时会返回参考来源。')}
          </p>
        </ApiSection>
      </div>
    </ScrollArea>
  )
}

function ApiSection(props: { title: string; children: React.ReactNode }) {
  return (
    <section className='space-y-3 border-t pt-5'>
      <h3 className='font-semibold'>{props.title}</h3>
      {props.children}
    </section>
  )
}

function EndpointTable(props: { rows: Array<[string, string]> }) {
  const { t } = useTranslation()
  return (
    <div className='overflow-x-auto rounded-md border'>
      <table className='w-full min-w-[520px] text-left text-sm'>
        <thead className='bg-muted/50 text-muted-foreground'>
          <tr>
            <th className='px-3 py-2 font-medium'>{t('接口')}</th>
            <th className='px-3 py-2 font-medium'>{t('说明')}</th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map(([path, description]) => (
            <tr key={path} className='border-t'>
              <td className='px-3 py-2 font-mono text-xs'>{path}</td>
              <td className='px-3 py-2 text-muted-foreground'>{description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CodeBlock(props: { children: string }) {
  return (
    <pre className='overflow-x-auto rounded-md bg-muted p-3 font-mono text-xs leading-5'>
      <code>{props.children}</code>
    </pre>
  )
}