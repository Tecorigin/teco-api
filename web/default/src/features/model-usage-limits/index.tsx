import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LimitsDeleteDialog } from './components/limits-delete-dialog'
import { LimitsMutateDrawer } from './components/limits-mutate-drawer'
import { LimitsPrimaryButtons } from './components/limits-primary-buttons'
import { LimitsProvider, useLimits } from './components/limits-provider'
import { LimitsTable } from './components/limits-table'
import { HitsTable } from './components/hits-table'

function ModelUsageLimitsContent() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useLimits()
  const [activeTab, setActiveTab] = useState('limits')

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Model Usage Limits')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('Configure per-user and per-model usage limits (daily/weekly/monthly) and view violation records.')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <LimitsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Tabs value={activeTab} onValueChange={setActiveTab} className='space-y-4'>
            <TabsList>
              <TabsTrigger value='limits'>{t('Limits')}</TabsTrigger>
              <TabsTrigger value='hits'>{t('Hit Records')}</TabsTrigger>
            </TabsList>
            <TabsContent value='limits'>
              <LimitsTable />
            </TabsContent>
            <TabsContent value='hits'>
              <HitsTable />
            </TabsContent>
          </Tabs>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <LimitsMutateDrawer
        open={open === 'create' || open === 'update'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={open === 'update' ? (currentRow as any) : undefined}
      />
      <LimitsDeleteDialog />
    </>
  )
}

export function ModelUsageLimits() {
  return (
    <LimitsProvider>
      <ModelUsageLimitsContent />
    </LimitsProvider>
  )
}
