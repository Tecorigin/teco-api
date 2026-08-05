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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Database,
  FileText,
  Loader2,
  Plus,
  RefreshCw,
  Trash2,
  Upload,
  Eraser,
  Network,
} from 'lucide-react'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  listKb,
  createKb,
  deleteKb,
  clearKb,
  listKbDocs,
  deleteKbDocs,
  uploadKbDocs,
  runKbIndex,
  getKbTaskStatus,
} from '../api'
import type { KnowledgeBase, KbCreateParams, KbDoc } from '../types'

export function KbManage() {
  const { t } = useTranslation()
  const [kbs, setKbs] = useState<KnowledgeBase[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listKb()
      if (res.success) {
        const list = res.data || []
        setKbs(list)
        if (list.length > 0 && !selected) {
          setSelected(list[0].name)
        }
      } else {
        toast.error(res.message || t('加载失败'))
      }
    } catch {
      toast.error(t('加载失败'))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [t])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div className='flex h-full gap-4 pt-2'>
      {/* Left: knowledge base list */}
      <div className='flex w-72 shrink-0 flex-col gap-2'>
        <div className='flex items-center justify-between'>
          <span className='text-sm font-medium text-muted-foreground'>
            {t('知识库列表')}
          </span>
          <div className='flex gap-1'>
            <Button variant='ghost' size='icon' onClick={load} title={t('刷新')}>
              <RefreshCw className='h-4 w-4' />
            </Button>
            <CreateKbDialog onCreated={load} />
          </div>
        </div>
        <div className='flex-1 space-y-2 overflow-auto'>
          {(() => {
            if (loading) {
              return (
                <div className='flex justify-center py-8'>
                  <Loader2 className='h-5 w-5 animate-spin text-muted-foreground' />
                </div>
              )
            }
            if (kbs.length === 0) {
              return (
                <div className='rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground'>
                  {t('暂无知识库，点击右上角 + 创建')}
                </div>
              )
            }
            return kbs.map((kb) => (
              <button
                type='button'
                key={kb.name}
                onClick={() => setSelected(kb.name)}
                className={`w-full rounded-md border p-3 text-left transition-colors hover:bg-accent ${
                  selected === kb.name ? 'border-primary bg-accent' : ''
                }`}
              >
                <div className='flex items-center gap-2'>
                  <Database className='h-4 w-4 shrink-0 text-primary' />
                  <span className='truncate font-medium'>{kb.name}</span>
                </div>
                {kb.description && (
                  <p className='mt-1 line-clamp-2 text-xs text-muted-foreground'>
                    {kb.description}
                  </p>
                )}
                <div className='mt-2 flex gap-3 text-xs text-muted-foreground'>
                  <span>{t('文档')} {kb.document_count ?? 0}</span>
                  <span>{t('分块')} {kb.chunk_count ?? 0}</span>
                </div>
              </button>
            ))
          })()}
        </div>
      </div>

      {/* Right: selected kb detail */}
      <div className='min-w-0 flex-1 overflow-auto'>
        {selected ? (
          <KbDetail kbName={selected} onChanged={load} onDeleted={() => { setSelected(null); load() }} />
        ) : (
          <div className='flex h-full items-center justify-center text-muted-foreground'>
            {t('请选择或创建一个知识库')}
          </div>
        )}
      </div>
    </div>
  )
}

function CreateKbDialog(props: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [chunkTokenNum, setChunkTokenNum] = useState('')
  const [autoKeywords, setAutoKeywords] = useState('')
  const [autoQuestions, setAutoQuestions] = useState('')
  const [tocExtraction, setTocExtraction] = useState(false)
  const [html4excel, setHtml4excel] = useState(false)

  const reset = () => {
    setName('')
    setDescription('')
    setChunkTokenNum('')
    setAutoKeywords('')
    setAutoQuestions('')
    setTocExtraction(false)
    setHtml4excel(false)
  }

  const submit = async () => {
    if (!name.trim()) {
      toast.error(t('请输入知识库名称'))
      return
    }
    setSubmitting(true)
    try {
      const params: KbCreateParams = { kb_name: name.trim() }
      if (description.trim()) params.description = description.trim()
      if (chunkTokenNum) params.chunk_token_num = Number(chunkTokenNum)
      if (autoKeywords) params.auto_keywords = Number(autoKeywords)
      if (autoQuestions) params.auto_questions = Number(autoQuestions)
      if (tocExtraction) params.toc_extraction = true
      if (html4excel) params.html4excel = true
      const res = await createKb(params)
      if (res.success) {
        toast.success(t('创建成功'))
        setOpen(false)
        reset()
        props.onCreated()
      } else {
        toast.error(res.message || t('创建失败'))
      }
    } catch {
      toast.error(t('创建失败'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Button variant='ghost' size='icon' onClick={() => setOpen(true)} title={t('创建知识库')}>
        <Plus className='h-4 w-4' />
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className='max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('创建知识库')}</DialogTitle>
            <DialogDescription>{t('填写基础信息创建一个新的知识库')}</DialogDescription>
          </DialogHeader>
          <div className='space-y-4'>
            <div className='space-y-1.5'>
              <Label>{t('名称')} *</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t('知识库名称')} />
            </div>
            <div className='space-y-1.5'>
              <Label>{t('描述')}</Label>
              <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder={t('知识库描述（可选）')} rows={2} />
            </div>
            <div className='grid grid-cols-3 gap-3'>
              <div className='space-y-1.5'>
                <Label>{t('分块 token 数')}</Label>
                <Input type='number' value={chunkTokenNum} onChange={(e) => setChunkTokenNum(e.target.value)} placeholder='512' />
              </div>
              <div className='space-y-1.5'>
                <Label>{t('自动关键词')}</Label>
                <Input type='number' value={autoKeywords} onChange={(e) => setAutoKeywords(e.target.value)} placeholder='0' />
              </div>
              <div className='space-y-1.5'>
                <Label>{t('自动问题')}</Label>
                <Input type='number' value={autoQuestions} onChange={(e) => setAutoQuestions(e.target.value)} placeholder='0' />
              </div>
            </div>
            <div className='flex gap-6'>
              <label className='flex items-center gap-2 text-sm'>
                <Switch checked={tocExtraction} onCheckedChange={setTocExtraction} />
                {t('目录提取 (PageIndex)')}
              </label>
              <label className='flex items-center gap-2 text-sm'>
                <Switch checked={html4excel} onCheckedChange={setHtml4excel} />
                {t('Excel 转 HTML')}
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setOpen(false)}>{t('取消')}</Button>
            <Button onClick={submit} disabled={submitting}>
              {submitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('创建')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

function KbDetail(props: { kbName: string; onChanged: () => void; onDeleted: () => void }) {
  const { t } = useTranslation()
  const { kbName } = props
  const [docs, setDocs] = useState<KbDoc[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedDocs, setSelectedDocs] = useState<Set<string>>(new Set())
  const [confirmAction, setConfirmAction] = useState<null | 'delete' | 'clear'>(null)
  const [acting, setActing] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const docsRequestIdRef = useRef(0)

  const loadDocs = useCallback(async () => {
    const requestId = ++docsRequestIdRef.current
    setLoading(true)
    try {
      const res = await listKbDocs(kbName)
      if (requestId !== docsRequestIdRef.current) return
      if (res.success) {
        setDocs(res.data || [])
      } else {
        toast.error(res.message || t('加载失败'))
      }
    } catch {
      if (requestId !== docsRequestIdRef.current) return
      toast.error(t('加载失败'))
    } finally {
      if (requestId === docsRequestIdRef.current) {
        setLoading(false)
      }
    }
  }, [kbName, t])

  useEffect(() => {
    setSelectedDocs(new Set())
    loadDocs()
  }, [loadDocs])

  const pollTask = useCallback(async (taskId: string) => {
    setUploadProgress(null)
    for (let i = 0; i < 120; i++) {
      await new Promise((r) => setTimeout(r, 3000))
      try {
        const res = await getKbTaskStatus(taskId)
        if (res.success && res.data) {
          setUploadProgress(res.data.progress ?? null)
          if (res.data.status === 'completed') {
            toast.success(t('处理完成'))
            setUploadProgress(null)
            await loadDocs()
            props.onChanged()
            return
          }
          if (res.data.status === 'failed') {
            toast.error(res.data.message || t('处理失败'))
            setUploadProgress(null)
            return
          }
        }
      } catch {
        // ignore polling errors
      }
    }
    setUploadProgress(null)
  }, [t, loadDocs, props])

  const handleUpload = async (files: FileList | null) => {
    if (!files || files.length === 0) return
    setUploading(true)
    try {
      const res = await uploadKbDocs(kbName, [...files])
      if (res.success && res.data?.task_id) {
        toast.success(t('上传任务已创建，正在解析…'))
        pollTask(res.data.task_id)
      } else {
        toast.error(res.message || t('上传失败'))
      }
    } catch {
      toast.error(t('上传失败'))
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleDeleteKb = async () => {
    setActing(true)
    try {
      const res = await deleteKb(kbName)
      if (res.success) {
        toast.success(t('已删除'))
        props.onDeleted()
      } else {
        toast.error(res.message || t('删除失败'))
      }
    } catch {
      toast.error(t('删除失败'))
    } finally {
      setActing(false)
      setConfirmAction(null)
    }
  }

  const handleClearKb = async () => {
    setActing(true)
    try {
      const res = await clearKb(kbName)
      if (res.success) {
        toast.success(t('已清空'))
        loadDocs()
        props.onChanged()
      } else {
        toast.error(res.message || t('清空失败'))
      }
    } catch {
      toast.error(t('清空失败'))
    } finally {
      setActing(false)
      setConfirmAction(null)
    }
  }

  const handleDeleteDocs = async () => {
    if (selectedDocs.size === 0) return
    setActing(true)
    try {
      const res = await deleteKbDocs(kbName, [...selectedDocs])
      if (res.success) {
        toast.success(t('已删除所选文档'))
        setSelectedDocs(new Set())
        loadDocs()
        props.onChanged()
      } else {
        toast.error(res.message || t('删除失败'))
      }
    } catch {
      toast.error(t('删除失败'))
    } finally {
      setActing(false)
    }
  }

  const handleRunIndex = async (indexType: string) => {
    try {
      const res = await runKbIndex(kbName, indexType)
      if (res.success && res.data?.task_id) {
        toast.success(t('索引构建已触发'))
        pollTask(res.data.task_id)
      } else {
        toast.error(res.message || t('触发失败'))
      }
    } catch {
      toast.error(t('触发失败'))
    }
  }

  const toggleDoc = (id: string) => {
    setSelectedDocs((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const runBadge = (run?: string) => {
    switch (run) {
      case 'DONE':
        return <Badge variant='default'>{t('已完成')}</Badge>
      case 'RUNNING':
        return <Badge variant='secondary'>{t('解析中')}</Badge>
      case 'FAIL':
        return <Badge variant='destructive'>{t('失败')}</Badge>
      default:
        return <Badge variant='outline'>{t('未开始')}</Badge>
    }
  }

  return (
    <Card>
      <CardHeader className='pb-3'>
        <div className='flex items-center justify-between'>
          <CardTitle className='flex items-center gap-2 text-base'>
            <Database className='h-4 w-4 text-primary' />
            {kbName}
          </CardTitle>
          <div className='flex items-center gap-2'>
            <input
              ref={fileInputRef}
              type='file'
              multiple
              className='hidden'
              onChange={(e) => handleUpload(e.target.files)}
            />
            <Button size='sm' variant='outline' onClick={() => fileInputRef.current?.click()} disabled={uploading}>
              {uploading ? <Loader2 className='mr-1.5 h-4 w-4 animate-spin' /> : <Upload className='mr-1.5 h-4 w-4' />}
              {t('上传文档')}
            </Button>
            <Button size='sm' variant='outline' onClick={() => handleRunIndex('graph')} title={t('构建知识图谱索引')}>
              <Network className='mr-1.5 h-4 w-4' />
              {t('图谱索引')}
            </Button>
            <Button size='sm' variant='outline' onClick={() => setConfirmAction('clear')}>
              <Eraser className='mr-1.5 h-4 w-4' />
              {t('清空')}
            </Button>
            <Button size='sm' variant='destructive' onClick={() => setConfirmAction('delete')}>
              <Trash2 className='mr-1.5 h-4 w-4' />
              {t('删除')}
            </Button>
          </div>
        </div>
        {uploadProgress != null && (
          <div className='mt-2'>
            <div className='mb-1 flex justify-between text-xs text-muted-foreground'>
              <span>{t('正在解析文档…')}</span>
              <span>{uploadProgress}%</span>
            </div>
            <Progress value={uploadProgress} />
          </div>
        )}
      </CardHeader>
      <CardContent>
        <div className='mb-2 flex items-center justify-between'>
          <span className='text-sm font-medium text-muted-foreground'>{t('文档列表')}</span>
          {selectedDocs.size > 0 && (
            <Button size='sm' variant='destructive' onClick={handleDeleteDocs} disabled={acting}>
              <Trash2 className='mr-1.5 h-3.5 w-3.5' />
              {t('删除所选')} ({selectedDocs.size})
            </Button>
          )}
        </div>
        {(() => {
          if (loading) {
            return (
              <div className='flex justify-center py-8'>
                <Loader2 className='h-5 w-5 animate-spin text-muted-foreground' />
              </div>
            )
          }
          if (docs.length === 0) {
            return (
              <div className='rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground'>
                <FileText className='mx-auto mb-2 h-8 w-8' />
                {t('暂无文档，点击上方"上传文档"添加')}
              </div>
            )
          }
          return (
            <div className='space-y-1.5'>
              {docs.map((doc) => (
                <div key={doc.id} className='flex items-center gap-3 rounded-md border p-2.5'>
                  <Checkbox checked={selectedDocs.has(doc.id)} onCheckedChange={() => toggleDoc(doc.id)} />
                  <FileText className='h-4 w-4 shrink-0 text-muted-foreground' />
                  <span className='min-w-0 flex-1 truncate text-sm'>{doc.name}</span>
                  <span className='text-xs text-muted-foreground'>{t('分块')} {doc.chunk_count ?? 0}</span>
                  {runBadge(doc.run)}
                </div>
              ))}
            </div>
          )
        })()}
      </CardContent>

      <AlertDialog open={confirmAction != null} onOpenChange={() => setConfirmAction(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {confirmAction === 'delete' ? t('删除知识库') : t('清空知识库')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {confirmAction === 'delete'
                ? t('将删除知识库 "{{name}}" 及其所有文档，此操作无法撤销。', { name: kbName })
                : t('将清空知识库 "{{name}}" 中的所有文档，此操作无法撤销。', { name: kbName })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('取消')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmAction === 'delete' ? handleDeleteKb : handleClearKb}
              disabled={acting}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              {acting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('确认')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
