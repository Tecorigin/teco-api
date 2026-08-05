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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Database, FileText, Loader2, MessageSquare, BookOpen } from 'lucide-react'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { getKbStatus, enableKb } from './api'
import type { KbStatus } from './types'
import { KbManage } from './components/kb-manage'
import { KbChat } from './components/kb-chat'
import { KbApiDocs } from './components/kb-api-docs'

export function KnowledgeBasePage() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<KbStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [enabling, setEnabling] = useState(false)

  const loadStatus = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getKbStatus()
      if (res.success) {
        setStatus(res.data)
      } else {
        toast.error(res.message || t('加载失败'))
      }
    } catch {
      toast.error(t('加载失败'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    loadStatus()
  }, [loadStatus])

  const handleEnable = async () => {
    setEnabling(true)
    try {
      const res = await enableKb()
      if (res.success) {
        toast.success(t('知识库开通成功'))
        await loadStatus()
      } else {
        toast.error(res.message || t('开通失败'))
      }
    } catch {
      toast.error(t('开通失败'))
    } finally {
      setEnabling(false)
    }
  }

  if (loading) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('知识库')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='flex h-full items-center justify-center'>
            <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  // Feature not configured on server
  if (!status?.configured) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('知识库')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <EmptyState
            icon={<Database className='h-10 w-10 text-muted-foreground' />}
            title={t('知识库功能未启用')}
            description={t('管理员尚未配置知识库服务，请联系管理员配置 KNOWLEDGE_BASE_URL 和 KNOWLEDGE_BASE_API_KEY。')}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  // Not enabled for this user: show enable button
  if (!status.enabled) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('知识库')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='flex h-full flex-col items-center justify-center gap-4'>
            <BookOpen className='h-12 w-12 text-muted-foreground' />
            <div className='text-center'>
              <h3 className='text-lg font-semibold'>{t('开通知识库')}</h3>
              <p className='mt-1 max-w-md text-sm text-muted-foreground'>
                {t('开通后即可创建和管理自己的知识库，并通过对话问答检索知识。系统会为您自动创建专属的知识库空间。')}
              </p>
            </div>
            <Button onClick={handleEnable} disabled={enabling} size='lg'>
              {enabling && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('开通知识库')}
            </Button>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  // Enabled: show manage + chat tabs
  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('知识库')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <Tabs defaultValue='manage' className='flex h-full flex-col'>
          <TabsList className='w-fit'>
            <TabsTrigger value='manage' className='gap-1.5'>
              <Database className='h-4 w-4' />
              {t('知识库管理')}
            </TabsTrigger>
            <TabsTrigger value='chat' className='gap-1.5'>
              <MessageSquare className='h-4 w-4' />
              {t('知识库问答')}
            </TabsTrigger>
            <TabsTrigger value='api-docs' className='gap-1.5'>
              <FileText className='h-4 w-4' />
              {t('API 文档')}
            </TabsTrigger>
          </TabsList>
          <TabsContent value='manage' className='min-h-0 flex-1 overflow-auto'>
            <KbManage />
          </TabsContent>
          <TabsContent value='chat' className='min-h-0 flex-1 overflow-hidden'>
            <KbChat />
          </TabsContent>
          <TabsContent value='api-docs' className='min-h-0 flex-1 overflow-hidden'>
            <KbApiDocs />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function EmptyState(props: {
  icon: React.ReactNode
  title: string
  description: string
}) {
  return (
    <div className='flex h-full flex-col items-center justify-center gap-3'>
      {props.icon}
      <h3 className='text-lg font-semibold'>{props.title}</h3>
      <p className='max-w-md text-center text-sm text-muted-foreground'>
        {props.description}
      </p>
    </div>
  )
}
