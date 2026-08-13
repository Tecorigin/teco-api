import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, ChevronDown, Clock3, Copy, Download, History, Loader2, Play, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { useVideoGenerationStore } from '@/stores/video-generation-store'
import { generateVideos, getUserModels, getVideoTask } from './api'
import type { VideoGenerationRequest, PersistedGeneratedVideo, VideoGenerationResponse, VideoGenerationTaskLogItem } from './types'

type GeneratedVideo = PersistedGeneratedVideo

const MAX_VIDEO_WIDTH = 1280
const MAX_VIDEO_HEIGHT = 720
const VIDEO_TASK_POLL_INTERVAL = 2000
const LEARNED_MODELS_STORAGE_KEY = 'video_generation_learned_models'
const VIDEO_MODEL_HINT_KEYWORDS = [
  'video',
  'wan',
  't2v',
  'kling',
  'sora',
  'veo',
  'hunyuan',
  'minimax',
]

function isLikelyVideoModel(modelName: string): boolean {
  const normalized = modelName.trim().toLowerCase()
  if (!normalized) return false
  return VIDEO_MODEL_HINT_KEYWORDS.some((keyword) => normalized.includes(keyword))
}

function ExpiredVideoPlaceholder({ label, description }: { label: string; description: string }) {
  return (
    <div className='text-muted-foreground flex min-h-[280px] flex-col items-center justify-center gap-3 bg-muted/30 px-6 text-center'>
      <div className='flex h-12 w-12 items-center justify-center rounded-full bg-background/80 shadow-sm'>
        <AlertCircle className='h-6 w-6 text-amber-500' />
      </div>
      <div className='space-y-1'>
        <div className='text-foreground text-sm font-medium'>{label}</div>
        <div className='text-xs'>{description}</div>
      </div>
    </div>
  )
}

async function saveRemoteFile(url: string, filename: string) {
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.target = '_blank'
  link.rel = 'noopener noreferrer'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

type ActiveVideoTask = {
  taskId: string
  model: string
  status: string
  progress: number
  createdAt?: number
}

function getTaskId(response?: VideoGenerationResponse | VideoGenerationTaskLogItem | null) {
  return response?.task_id || (response?.id != null ? String(response.id) : '')
}

function getVideoUrlCandidate(item?: {
  url?: string
  download_url?: string
  result_url?: string
} | null) {
  return item?.url || item?.download_url || item?.result_url || ''
}

function extractVideosFromResponse(
  response?: VideoGenerationResponse | VideoGenerationTaskLogItem | null,
  elapsed?: string
): GeneratedVideo[] {
  const videos: GeneratedVideo[] = []

  if ('data' in (response || {}) && Array.isArray((response as VideoGenerationResponse)?.data)) {
    for (const item of (response as VideoGenerationResponse).data) {
      const videoUrl = getVideoUrlCandidate(item)
      if (videoUrl) {
        videos.push({ src: videoUrl, elapsed })
      }
    }
  }

  const topLevelUrl = getVideoUrlCandidate(response)
  if (topLevelUrl && !videos.some((item) => item.src === topLevelUrl)) {
    videos.push({ src: topLevelUrl, elapsed })
  }

  const taskResponse = response && 'result_url' in response ? response as VideoGenerationTaskLogItem : null
  const metadataUrl = getVideoUrlCandidate(taskResponse)
  if (metadataUrl && !videos.some((item) => item.src === metadataUrl)) {
    videos.push({ src: metadataUrl, elapsed })
  }

  return videos
}

function normalizeTaskStatus(status?: string) {
  return (status || 'queued').toLowerCase()
}

function parseTaskProgress(progress?: string | number) {
  if (typeof progress === 'number') {
    return Math.max(0, Math.min(100, progress))
  }

  const raw = String(progress || '').trim().replace(/%$/, '')
  const value = Number(raw)
  if (Number.isNaN(value)) {
    return 0
  }
  return Math.max(0, Math.min(100, value))
}

function isTaskCompleted(status?: string) {
  const normalized = normalizeTaskStatus(status)
  return normalized === 'completed' || normalized === 'success'
}

function isTaskFailed(status?: string) {
  const normalized = normalizeTaskStatus(status)
  return normalized === 'failed' || normalized === 'failure'
}

function isInvalidSizeSeparatorError(error: unknown) {
  const err = error as {
    response?: { data?: { code?: string; message?: string } }
    message?: string
  }
  const code = err?.response?.data?.code || ''
  const message = err?.response?.data?.message || err?.message || ''
  return code === 'build_request_failed' && /invalid size/i.test(message)
}

function appendHistoryRecord(request: VideoGenerationRequest, nextVideos: GeneratedVideo[]) {
  useVideoGenerationStore.setState((state) => ({
    history: [
      {
        id: `${Date.now()}-${request.model}`,
        createdAt: Date.now(),
        request,
        videos: nextVideos,
      },
      ...state.history,
    ].slice(0, 20),
  }))
}

export function VideoGenerationPlayground() {
  const { t } = useTranslation()
  const { history, clearHistory } = useVideoGenerationStore()

  const [model, setModel] = useState('')
  const [prompt, setPrompt] = useState('')
  const [steps, setSteps] = useState(20)
  const [seed, setSeed] = useState(42)
  const [guidanceScale, setGuidanceScale] = useState(7.5)
  const [guidanceScale2, setGuidanceScale2] = useState(3.0)
  const [height, setHeight] = useState(160)
  const [width, setWidth] = useState(320)
  const [seconds, setSeconds] = useState('5')
  const [fps, setFps] = useState(16)
  const [sizeSeparator, setSizeSeparator] = useState<'x' | '*'>("x")
  const [showAllModels, setShowAllModels] = useState(false)
  const [learnedModels, setLearnedModels] = useState<string[]>([])
  const [isGenerating, setIsGenerating] = useState(false)
  const [videos, setVideos] = useState<GeneratedVideo[]>([])
  const [previewVideo, setPreviewVideo] = useState<GeneratedVideo | null>(null)
  const [pendingRequest, setPendingRequest] = useState<VideoGenerationRequest | null>(null)
  const [activeTask, setActiveTask] = useState<ActiveVideoTask | null>(null)
  const [generatedNotice, setGeneratedNotice] = useState<string | null>(null)
  const [showAdvancedSettings, setShowAdvancedSettings] = useState(false)

  const { data: models = [], isLoading: isLoadingModels } = useQuery({
    queryKey: ['video-generation-models', showAllModels],
    queryFn: () => getUserModels(showAllModels),
  })

  useEffect(() => {
    if (!isGenerating) return

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = t('Leaving this page will stop the current video generation task.')
      return event.returnValue
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [isGenerating, t])

  useEffect(() => {
    return () => {
      if (isGenerating) {
        toast.warning(t('Leaving this page will stop the current video generation task.'))
      }
    }
  }, [isGenerating, t])

  useEffect(() => {
    if (!activeTask?.taskId || !isGenerating) {
      return
    }

    let cancelled = false

    const pollTask = async () => {
      try {
        const response = await getVideoTask(activeTask.taskId)
        if (cancelled) return

        if (!response) {
          throw new Error(t('Failed to query video task status'))
        }

        const nextStatus = normalizeTaskStatus(response.status)
        const nextProgress = parseTaskProgress(response.progress)

        setActiveTask((current) =>
          current && current.taskId === activeTask.taskId
            ? {
                ...current,
                status: nextStatus,
                progress: nextProgress,
                createdAt: response.created_at ?? current.createdAt,
              }
            : current
        )

        if (isTaskFailed(nextStatus)) {
          setPendingRequest(null)
          setActiveTask(null)
          setIsGenerating(false)
          toast.error(response.fail_reason || t('Video generation failed'))
          return
        }

        if (isTaskCompleted(nextStatus)) {
          const nextVideos = extractVideosFromResponse(response)
          if (nextVideos.length === 0) {
            setPendingRequest(null)
            setActiveTask(null)
            setIsGenerating(false)
            toast.error(t('No videos were returned'))
            return
          }

          const request = pendingRequest
          if (request) {
            appendHistoryRecord(request, nextVideos)
          }

          setVideos(nextVideos)
          setGeneratedNotice(
            t('Generated videos use temporary links. If they expire, they will no longer be displayed, so please save them as soon as possible.')
          )
          setPendingRequest(null)
          setActiveTask(null)
          setIsGenerating(false)
          toast.success(t('Videos generated successfully'))
        }
      } catch (error: unknown) {
        if (cancelled) return
        const err = error as {
          response?: { data?: { message?: string } }
          message?: string
        }
        setPendingRequest(null)
        setActiveTask(null)
        setIsGenerating(false)
        toast.error(err?.response?.data?.message || err?.message || t('Failed to query video task status'))
      }
    }

    void pollTask()
    const timer = window.setInterval(() => {
      void pollTask()
    }, VIDEO_TASK_POLL_INTERVAL)

    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [activeTask?.taskId, isGenerating, pendingRequest, t])

  useEffect(() => {
    if (typeof window === 'undefined') return
    try {
      const raw = window.localStorage.getItem(LEARNED_MODELS_STORAGE_KEY)
      if (!raw) return
      const parsed = JSON.parse(raw) as unknown
      if (!Array.isArray(parsed)) return
      const sanitized = parsed
        .filter((item): item is string => typeof item === 'string')
        .map((item) => item.trim())
        .filter(Boolean)
      if (sanitized.length > 0) {
        setLearnedModels(Array.from(new Set(sanitized)))
      }
    } catch {
      // Ignore malformed local storage value.
    }
  }, [])

  const visibleModels = useMemo(() => {
    if (showAllModels) {
      return models
    }

    const learnedSet = new Set(learnedModels)
    const filtered = models.filter((option) => {
      if (learnedSet.has(option.value)) {
        return true
      }
      return isLikelyVideoModel(option.value)
    })

    if (model && !filtered.some((option) => option.value === model)) {
      const selected = models.find((option) => option.value === model)
      if (selected) {
        return [...filtered, selected]
      }
    }

    return filtered
  }, [learnedModels, model, models, showAllModels])

  useEffect(() => {
    if (visibleModels.length === 0) {
      if (model) {
        setModel('')
      }
      return
    }

    if (!model || !visibleModels.some((option) => option.value === model)) {
      setModel(visibleModels[0].value)
    }
  }, [model, visibleModels])

  const normalizedPayload: VideoGenerationRequest | null = useMemo(() => {
    if (!model || !prompt.trim()) {
      return null
    }

    return {
      model,
      prompt: prompt.trim(),
      size: `${Math.min(MAX_VIDEO_WIDTH, Math.max(1, Math.floor(width)))}${sizeSeparator}${Math.min(MAX_VIDEO_HEIGHT, Math.max(1, Math.floor(height)))}`,
      size_separator: sizeSeparator,
      seconds: String(Math.max(1, Math.floor(Number(seconds) || 1))),
      response_format: 'url',
      num_inference_steps: Math.max(1, Math.floor(steps)),
      seed: Math.floor(seed),
      guidance_scale: guidanceScale,
      guidance_scale_2: guidanceScale2,
      fps: Math.max(1, Math.floor(fps)),
    }
  }, [fps, guidanceScale, guidanceScale2, height, model, prompt, seconds, seed, sizeSeparator, steps, width])

  const handleGenerate = async () => {
    if (!model) {
      toast.error(t('Please select a model'))
      return
    }
    if (!prompt.trim()) {
      toast.error(t('Please enter a prompt'))
      return
    }
    if (!normalizedPayload) {
      return
    }

    setIsGenerating(true)
    setVideos([])
    setGeneratedNotice(null)
    setPendingRequest(normalizedPayload)
    setActiveTask({
      taskId: '',
      model: normalizedPayload.model,
      status: 'submitting',
      progress: 0,
    })

    try {
      let requestPayload = normalizedPayload
      let response: VideoGenerationResponse

      try {
        response = await generateVideos(requestPayload)
      } catch (error: unknown) {
        if (requestPayload.size_separator === 'x' && isInvalidSizeSeparatorError(error)) {
          requestPayload = {
            ...requestPayload,
            size_separator: '*',
            size: requestPayload.size.replace('x', '*'),
          }
          setSizeSeparator('*')
          toast.info(t('The current model requires `*` as the size separator. Automatically retried once.'))
          response = await generateVideos(requestPayload)
        } else {
          throw error
        }
      }

      const nextVideos = extractVideosFromResponse(response, response.elapsed_time)
      const taskId = getTaskId(response)
      const normalizedStatus = normalizeTaskStatus(response.status)

      if (typeof window !== 'undefined') {
        setLearnedModels((prev) => {
          if (prev.includes(requestPayload.model)) {
            return prev
          }
          const next = [...prev, requestPayload.model]
          try {
            window.localStorage.setItem(LEARNED_MODELS_STORAGE_KEY, JSON.stringify(next))
          } catch {
            // Ignore storage write failures.
          }
          return next
        })
      }

      if (nextVideos.length > 0) {
        setVideos(nextVideos)
        setGeneratedNotice(
          t('Generated videos use temporary links. If they expire, they will no longer be displayed, so please save them as soon as possible.')
        )
        appendHistoryRecord(requestPayload, nextVideos)
        setPendingRequest(null)
        setActiveTask(null)
        setIsGenerating(false)
        toast.success(t('Videos generated successfully'))
        return
      }

      if (taskId) {
        setActiveTask({
          taskId,
          model: response.model || requestPayload.model,
          status: normalizedStatus || 'queued',
          progress: parseTaskProgress(response.progress),
          createdAt: response.created_at,
        })
        toast.success(t('Video task submitted successfully'))
        return
      }

      if (isTaskFailed(normalizedStatus)) {
        setPendingRequest(null)
        setActiveTask(null)
        setIsGenerating(false)
        toast.error(response.error?.message || t('Video generation failed'))
        return
      }

      setPendingRequest(null)
      setActiveTask(null)
      setIsGenerating(false)
      toast.error(t('No videos were returned'))
    } catch (error: unknown) {
      const err = error as {
        response?: { data?: { message?: string } }
        message?: string
      }
      setPendingRequest(null)
      setActiveTask(null)
      setIsGenerating(false)
      toast.error(err?.response?.data?.message || err?.message || t('Generate failed'))
    }
  }

  const handleVideoLoadError = (targetVideo: GeneratedVideo) => {
    const markExpired = (list: GeneratedVideo[]) =>
      list.map((video) => (video.src === targetVideo.src ? { ...video, isExpired: true } : video))

    setVideos((current) => markExpired(current))
    useVideoGenerationStore.setState((state) => ({
      history: state.history.map((item) => ({
        ...item,
        videos: item.videos.map((video) =>
          video.src === targetVideo.src ? { ...video, isExpired: true } : video
        ),
      })),
    }))
    setPreviewVideo((current) =>
      current?.src === targetVideo.src ? { ...current, isExpired: true } : current
    )
  }

  const handleCopyVideoLink = async (video: GeneratedVideo) => {
    if (video.isExpired) {
      toast.error(t('Video expired'))
      return
    }
    try {
      await navigator.clipboard.writeText(video.src)
      toast.success(t('Video link copied'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  const handleSaveVideo = async (video: GeneratedVideo, index: number) => {
    if (video.isExpired) {
      toast.error(t('Video expired'))
      return
    }
    try {
      await saveRemoteFile(video.src, `generated-video-${index + 1}.mp4`)
      toast.success(t('Video saved locally'))
    } catch {
      toast.error(t('Save failed'))
    }
  }

  const renderVideoGrid = (items: GeneratedVideo[]) => (
    <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
      {items.map((video, index) => (
        <div key={`${video.src}-${index}`} className='group space-y-2'>
          <button
            type='button'
            className='w-full text-left'
            onClick={() => !video.isExpired && setPreviewVideo(video)}
          >
            <div className='bg-background relative overflow-hidden rounded-md border'>
            {video.isExpired ? (
              <ExpiredVideoPlaceholder
                label={t('Video expired')}
                description={t('This temporary video link has expired.')}
              />
            ) : (
              <video
                src={video.src}
                className='h-auto max-h-[70vh] w-full object-contain'
                muted
                playsInline
                preload='metadata'
                onError={() => handleVideoLoadError(video)}
              />
            )}
            <div className='absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent px-3 py-2 opacity-0 transition-opacity duration-200 group-hover:opacity-100'>
              <span className='text-xs text-white'>
                {video.isExpired ? t('Video expired') : t('Click to preview video')}
              </span>
            </div>
            {!video.isExpired && (
              <div className='pointer-events-none absolute inset-0 flex items-center justify-center'>
                <div className='flex h-12 w-12 items-center justify-center rounded-full bg-black/50 text-white'>
                  <Play className='ml-1 h-5 w-5 fill-current' />
                </div>
              </div>
            )}
            </div>
          </button>
          <div className='text-muted-foreground flex items-center justify-between gap-3 text-xs'>
            <div className='space-y-1'>
              <span className='block'>{t('Video')} #{index + 1}</span>
              <div className='flex flex-wrap gap-x-3 gap-y-1'>
                <span>{t('Resolution')}: {width}×{height}</span>
                <span>{t('guidance_scale')}: {guidanceScale}</span>
                <span>{t('guidance_scale_2')}: {guidanceScale2}</span>
              </div>
            </div>
            <div className='flex items-center gap-3'>
              <span>{t('Temporary video URL')}</span>
              {video.elapsed && <span>{video.elapsed}</span>}
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='h-7 w-7'
                onClick={() => void handleCopyVideoLink(video)}
                disabled={video.isExpired}
              >
                <Copy className='h-4 w-4' />
                <span className='sr-only'>{t('Copy video link')}</span>
              </Button>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='h-7 w-7'
                onClick={() => void handleSaveVideo(video, index)}
                disabled={video.isExpired}
              >
                <Download className='h-4 w-4' />
                <span className='sr-only'>{t('Save video')}</span>
              </Button>
            </div>
          </div>
        </div>
      ))}
    </div>
  )

  return (
    <ScrollArea className='min-h-0 flex-1'>
      <div className='mx-auto flex w-full max-w-6xl flex-col gap-4 p-4 md:p-6'>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>{t('Text-to-Video')}</h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Generate videos from prompts with model parameters')}
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t('Generation settings')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
            <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
              <div className='space-y-2'>
                <div className='flex items-center justify-between gap-2'>
                  <Label>{t('Model')}</Label>
                  <div className='flex items-center gap-2'>
                    <Label className='text-muted-foreground text-xs'>
                      {t('Show all models')}
                    </Label>
                    <Switch checked={showAllModels} onCheckedChange={setShowAllModels} />
                  </div>
                </div>
                <Select value={model} onValueChange={(value) => setModel(value ?? '')}>
                  <SelectTrigger>
                    <SelectValue
                      placeholder={
                        isLoadingModels ? t('Loading models...') : t('Select a model')
                      }
                    />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    {visibleModels.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='space-y-2'>
                <Label>{t('Prompt')}</Label>
                <Textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder={t('Text description of the desired video')}
                  rows={4}
                />
              </div>
            </div>

            <Collapsible open={showAdvancedSettings} onOpenChange={setShowAdvancedSettings}>
              <div className='flex items-center justify-between rounded-md border border-dashed px-4 py-3'>
                <div>
                  <div className='text-sm font-medium'>{t('Advanced settings')}</div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Configure steps, seed, guidance scales, duration, fps, and resolution.')}
                  </div>
                </div>
                <CollapsibleTrigger>
                  <Button type='button' variant='ghost' size='sm'>
                    {showAdvancedSettings ? t('Collapse') : t('Expand')}
                    <ChevronDown className={`ml-2 h-4 w-4 transition-transform ${showAdvancedSettings ? 'rotate-180' : ''}`} />
                  </Button>
                </CollapsibleTrigger>
              </div>

              <CollapsibleContent className='mt-4 space-y-4'>
                <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
                  <div className='space-y-2'>
                    <Label>{t('num_inference_steps')}</Label>
                    <Input type='number' min={1} value={steps} onChange={(e) => setSteps(Number(e.target.value || 1))} />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('seed')}</Label>
                    <Input type='number' value={seed} onChange={(e) => setSeed(Number(e.target.value || 0))} />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('guidance_scale')}</Label>
                    <Input type='number' step='0.1' value={guidanceScale} onChange={(e) => setGuidanceScale(Number(e.target.value || 0))} />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('guidance_scale_2')}</Label>
                    <Input type='number' step='0.1' value={guidanceScale2} onChange={(e) => setGuidanceScale2(Number(e.target.value || 0))} />
                  </div>
                </div>

                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('seconds')}</Label>
                    <Input type='number' min={1} value={seconds} onChange={(e) => setSeconds(String(e.target.value || '1'))} />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('fps')}</Label>
                    <Input type='number' min={1} value={fps} onChange={(e) => setFps(Number(e.target.value || 1))} />
                  </div>
                </div>

                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('Width')}</Label>
                    <Input type='number' min={1} max={MAX_VIDEO_WIDTH} value={width} onChange={(e) => setWidth(Math.min(MAX_VIDEO_WIDTH, Number(e.target.value || 1)))} />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('Height')}</Label>
                    <Input type='number' min={1} max={MAX_VIDEO_HEIGHT} value={height} onChange={(e) => setHeight(Math.min(MAX_VIDEO_HEIGHT, Number(e.target.value || 1)))} />
                  </div>
                </div>

                <div className='space-y-2'>
                  <Label>{t('Size separator')}</Label>
                  <Select value={sizeSeparator} onValueChange={(value) => setSizeSeparator(value === '*' ? '*' : 'x')}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='x'>{t('x (e.g. 320x160)')}</SelectItem>
                      <SelectItem value='*'>{t('* (e.g. 320*160)')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </CollapsibleContent>
            </Collapsible>

            <Button onClick={handleGenerate} disabled={isGenerating || isLoadingModels}>
              {isGenerating ? (
                <>
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  {t('Generating...')}
                </>
              ) : (
                t('Generate')
              )}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('Generated videos')}</CardTitle>
          </CardHeader>
          <CardContent>
            {generatedNotice && <p className='mb-4 text-sm text-amber-600'>{generatedNotice}</p>}
            {isGenerating ? (
              <div className='flex min-h-[280px] flex-col items-center justify-center rounded-lg border border-dashed bg-muted/20 px-6 py-10 text-center'>
                <Loader2 className='text-primary h-8 w-8 animate-spin' />
                <p className='mt-4 text-sm font-medium'>{t('Generating videos...')}</p>
                <p className='text-muted-foreground mt-2 text-sm'>
                  {t('Please wait while the model generates your videos.')}
                </p>
                {activeTask && (
                  <div className='mt-4 flex flex-col items-center gap-2 text-xs'>
                    <div className='text-muted-foreground flex items-center gap-2'>
                      <Clock3 className='h-3.5 w-3.5' />
                      <span>{t('Task ID')}: {activeTask.taskId}</span>
                      <span>{t('Status')}: {activeTask.status}</span>
                    </div>
                    <div className='h-2 w-full max-w-sm overflow-hidden rounded-full bg-muted'>
                      <div
                        className='bg-primary h-full transition-all'
                        style={{ width: `${activeTask.progress}%` }}
                      />
                    </div>
                    <p className='text-muted-foreground'>{t('Progress')}: {activeTask.progress}%</p>
                  </div>
                )}
                {pendingRequest && (
                  <div className='text-muted-foreground mt-4 flex items-center gap-2 text-xs'>
                    <Clock3 className='h-3.5 w-3.5' />
                    <span>{t('Current task')}: {pendingRequest.model} · {pendingRequest.size}</span>
                  </div>
                )}
              </div>
            ) : videos.length === 0 ? (
              <p className='text-muted-foreground text-sm'>
                {t('No videos yet. Enter a prompt and click generate.')}
              </p>
            ) : (
              renderVideoGrid(videos)
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='flex flex-row items-center justify-between gap-3'>
            <CardTitle className='flex items-center gap-2'>
              <History className='h-4 w-4' />
              {t('Generation history')}
            </CardTitle>
            <Button type='button' variant='outline' size='sm' onClick={clearHistory} disabled={history.length === 0}>
              <Trash2 className='mr-2 h-4 w-4' />
              {t('Clear history')}
            </Button>
          </CardHeader>
          <CardContent className='space-y-4'>
            {history.length === 0 ? (
              <p className='text-muted-foreground text-sm'>{t('No generation history yet.')}</p>
            ) : (
              history.map((item) => (
                <div key={item.id} className='space-y-3 rounded-lg border p-4'>
                  <div className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
                    <div className='space-y-1'>
                      <div className='text-sm font-medium'>{item.request.model}</div>
                      <div className='text-muted-foreground text-xs'>{new Date(item.createdAt).toLocaleString()}</div>
                    </div>
                    <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                      <span>{item.request.size}</span>
                      <span>{t('guidance_scale')}: {item.request.guidance_scale}</span>
                      <span>{t('guidance_scale_2')}: {item.request.guidance_scale_2}</span>
                      <span>{t('seconds')}: {item.request.seconds}</span>
                      <span>{t('fps')}: {item.request.fps}</span>
                    </div>
                  </div>
                  <p className='text-sm break-words'>{item.request.prompt}</p>
                  <p className='text-amber-600 text-xs'>
                    {t('Some historical videos use temporary URLs and may expire.')}
                  </p>
                  {renderVideoGrid(item.videos)}
                </div>
              ))
            )}
          </CardContent>
        </Card>

        <Dialog open={Boolean(previewVideo)} onOpenChange={(open) => !open && setPreviewVideo(null)}>
          <DialogContent className='w-[98vw] max-w-[1400px]'>
            <DialogHeader>
              <DialogTitle>{t('Video Preview')}</DialogTitle>
            </DialogHeader>
            {previewVideo && (
              <div className='space-y-4'>
                <div className='flex flex-wrap items-center justify-between gap-3'>
                  <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-2 text-sm'>
                    <span>{t('Resolution')}: {width}×{height}</span>
                    <span>{t('guidance_scale')}: {guidanceScale}</span>
                    <span>{t('guidance_scale_2')}: {guidanceScale2}</span>
                    {previewVideo.elapsed && <span>{t('Elapsed time')}: {previewVideo.elapsed}</span>}
                  </div>
                  <div className='flex items-center gap-2'>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() => void handleCopyVideoLink(previewVideo)}
                      disabled={previewVideo.isExpired}
                    >
                      <Copy className='mr-2 h-4 w-4' />
                      {t('Copy video link')}
                    </Button>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={() => void handleSaveVideo(previewVideo, 0)}
                      disabled={previewVideo.isExpired}
                    >
                      <Download className='mr-2 h-4 w-4' />
                      {t('Save video')}
                    </Button>
                  </div>
                </div>
                <div className='max-h-[80vh] overflow-auto rounded-md border bg-muted/10'>
                {previewVideo.isExpired ? (
                  <div className='min-h-[60vh]'>
                    <ExpiredVideoPlaceholder
                      label={t('Video expired')}
                      description={t('This temporary video link has expired.')}
                    />
                  </div>
                ) : (
                  <video
                    src={previewVideo.src}
                    className='mx-auto block max-h-[80vh] w-full object-contain'
                    controls
                    autoPlay
                    playsInline
                    onError={() => handleVideoLoadError(previewVideo)}
                  />
                )}
              </div>
              </div>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </ScrollArea>
  )
}