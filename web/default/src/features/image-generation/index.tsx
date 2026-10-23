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
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, ChevronDown, Clock3, Download, History, Loader2, Search, Trash2, ZoomIn, ZoomOut } from 'lucide-react'
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
import { useImageGenerationStore } from '@/stores/image-generation-store'
import { generateImages, getUserModels } from './api'
import type { ImageGenerationRequest } from './types'

type GeneratedImage = {
  src: string
  elapsed?: string
  sourceType?: 'url' | 'b64_json'
  isExpired?: boolean
}

const IMAGE_SIZE_OPTIONS = ['256x256', '512x512', '1024x1024', '2048x2048']
const LEARNED_MODELS_STORAGE_KEY = 'image_generation_learned_models'
const IMAGE_MODEL_HINT_KEYWORDS = [
  'stable-diffusion',
  'sdxl',
  'sd3',
  'sd-3',
  'flux',
  'dall-e',
  'image',
  'diffusion',
  'midjourney',
  'recraft',
  'wan',
  'hunyuan',
  'kandinsky',
]

function getGeneratedImageSource(item: { url?: string; b64_json?: string }): {
  src: string
  sourceType: GeneratedImage['sourceType']
} {
  if (item.url?.startsWith('file://')) {
    const base64 = item.url.slice('file://'.length)
    return {
      src: base64 ? `data:image/png;base64,${base64}` : '',
      sourceType: 'b64_json',
    }
  }
  if (item.url) {
    return { src: item.url, sourceType: 'url' }
  }
  if (item.b64_json) {
    return { src: `data:image/png;base64,${item.b64_json}`, sourceType: 'b64_json' }
  }
  return { src: '', sourceType: undefined }
}

function isLikelyImageModel(modelName: string): boolean {
  const normalized = modelName.trim().toLowerCase()
  if (!normalized) return false
  return IMAGE_MODEL_HINT_KEYWORDS.some((keyword) => normalized.includes(keyword))
}

function ExpiredImagePlaceholder({ label, description }: { label: string; description: string }) {
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

export function ImageGenerationPlayground() {
  const { t } = useTranslation()
  const { history, clearHistory } = useImageGenerationStore()
  const MIN_PREVIEW_SCALE = 0.25
  const MAX_PREVIEW_SCALE = 4
  const PREVIEW_SCALE_STEP = 0.25

  const [model, setModel] = useState('')
  const [prompt, setPrompt] = useState('')
  const [n, setN] = useState(2)
  const [size, setSize] = useState('512x512')
  const [steps, setSteps] = useState(20)
  const [seed, setSeed] = useState(42)
  const [cfg, setCfg] = useState(7.5)
  const [showAllModels, setShowAllModels] = useState(false)
  const [learnedModels, setLearnedModels] = useState<string[]>([])
  const [isGenerating, setIsGenerating] = useState(false)
  const [images, setImages] = useState<GeneratedImage[]>([])
  const [previewImage, setPreviewImage] = useState<GeneratedImage | null>(null)
  const [previewScale, setPreviewScale] = useState(1)
  const [generatedNotice, setGeneratedNotice] = useState<string | null>(null)
  const [showAdvancedSettings, setShowAdvancedSettings] = useState(false)

  const [pendingRequest, setPendingRequest] = useState<ImageGenerationRequest | null>(null)

  const { data: models = [], isLoading: isLoadingModels } = useQuery({
    queryKey: ['image-generation-models', showAllModels],
    queryFn: () => getUserModels(showAllModels),
  })

  useEffect(() => {
    if (!isGenerating) {
      return
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = t('Leaving this page will stop the current image generation task.')
      return event.returnValue
    }

    window.addEventListener('beforeunload', handleBeforeUnload)

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    }
  }, [isGenerating, t])

  useEffect(() => {
    return () => {
      if (isGenerating) {
        toast.warning(t('Leaving this page will stop the current image generation task.'))
      }
    }
  }, [isGenerating, t])

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

  useEffect(() => {
    setPreviewScale(1)
  }, [previewImage])

  const visibleModels = useMemo(() => {
    if (showAllModels) {
      return models
    }

    const learnedSet = new Set(learnedModels)
    const filtered = models.filter((option) => {
      if (learnedSet.has(option.value)) {
        return true
      }
      return isLikelyImageModel(option.value)
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

  const normalizedPayload: ImageGenerationRequest | null = useMemo(() => {
    if (!model || !prompt.trim()) {
      return null
    }

    return {
      model,
      prompt: prompt.trim(),
      n: Math.max(1, Math.floor(n)),
      size,
      response_format: 'url',
      num_inference_steps: Math.max(1, Math.floor(steps)),
      seed: Math.floor(seed),
      cfg,
    }
  }, [cfg, model, n, prompt, seed, size, steps])

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
    setImages([])
    setGeneratedNotice(null)
    setPendingRequest(normalizedPayload)

    try {
      const response = await generateImages(normalizedPayload)

      const elapsedTimes = Array.isArray(response.elapsed_time)
        ? response.elapsed_time
        : []
      const nextImages = (response.data || [])
        .map<GeneratedImage | null>((item, index) => {
          const { src, sourceType } = getGeneratedImageSource(item)
          if (!src) {
            return null
          }
          return {
            src,
            elapsed: elapsedTimes[index] ?? (typeof response.elapsed_time === 'string' ? response.elapsed_time : undefined),
            sourceType,
          }
        })
        .filter((item): item is GeneratedImage => item !== null)

      if (nextImages.length === 0) {
        toast.error(t('No images were returned'))
        return
      }

      if (typeof window !== 'undefined') {
        setLearnedModels((prev) => {
          if (prev.includes(normalizedPayload.model)) {
            return prev
          }
          const next = [...prev, normalizedPayload.model]
          try {
            window.localStorage.setItem(
              LEARNED_MODELS_STORAGE_KEY,
              JSON.stringify(next)
            )
          } catch {
            // Ignore storage write failures.
          }
          return next
        })
      }

      setImages(nextImages)
      if (nextImages.some((image) => image.sourceType === 'url')) {
        setGeneratedNotice(t('Generated images use temporary links. If they expire, they will no longer be displayed, so please save them as soon as possible.'))
      }
      if (pendingRequest ?? normalizedPayload) {
        const request = pendingRequest ?? normalizedPayload
        useImageGenerationStore.setState((state) => ({
          history: [
            {
              id: `${Date.now()}-${request.model}`,
              createdAt: Date.now(),
              request,
              images: nextImages.map((image) => ({
                src: image.src,
                elapsed: image.elapsed,
                sourceType: image.sourceType ?? 'url',
              })),
            },
            ...state.history,
          ].slice(0, 20),
        }))
      }
      toast.success(t('Images generated successfully'))
    } catch (error: unknown) {
      const err = error as {
        response?: { data?: { message?: string } }
        message?: string
      }
      toast.error(err?.response?.data?.message || err?.message || t('Generate failed'))
    } finally {
      setPendingRequest(null)
      setIsGenerating(false)
    }
  }

  const updatePreviewScale = (updater: (current: number) => number) => {
    setPreviewScale((current) => {
      const next = updater(current)
      return Math.min(MAX_PREVIEW_SCALE, Math.max(MIN_PREVIEW_SCALE, next))
    })
  }

  const handlePreviewWheel = (event: React.WheelEvent<HTMLDivElement>) => {
    if (!event.ctrlKey) {
      return
    }

    event.preventDefault()
    updatePreviewScale((current) =>
      Number((current + (event.deltaY < 0 ? PREVIEW_SCALE_STEP : -PREVIEW_SCALE_STEP)).toFixed(2))
    )
  }

  const handleImageLoadError = (targetImage: GeneratedImage) => {
    const markExpired = (list: GeneratedImage[]) =>
      list.map((image) =>
        image.src === targetImage.src ? { ...image, isExpired: true } : image
      )

    setImages((current) => markExpired(current))

    useImageGenerationStore.setState((state) => ({
      history: state.history.map((item) => ({
        ...item,
        images: item.images.map((image) =>
          image.src === targetImage.src ? { ...image, isExpired: true } : image
        ),
      })),
    }))

    setPreviewImage((current) =>
      current?.src === targetImage.src ? { ...current, isExpired: true } : current
    )
  }

  const handleSaveImage = async (image: GeneratedImage, index: number) => {
    if (image.isExpired) {
      toast.error(t('Image expired'))
      return
    }

    try {
      if (image.src.startsWith('data:')) {
        const link = document.createElement('a')
        link.href = image.src
        link.download = `generated-image-${index + 1}.png`
        document.body.appendChild(link)
        link.click()
        link.remove()
      } else {
        await saveRemoteFile(image.src, `generated-image-${index + 1}.png`)
      }
      toast.success(t('Image saved locally'))
    } catch {
      toast.error(t('Save failed'))
    }
  }

  const renderImageGrid = (items: GeneratedImage[]) => (
    <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
      {items.map((image, index) => (
        <div key={`${image.src}-${index}`} className='group space-y-2'>
          <button
            type='button'
            className='w-full text-left'
            onClick={() => !image.isExpired && setPreviewImage(image)}
          >
            <div className='bg-background relative overflow-hidden rounded-md border'>
            {image.isExpired ? (
              <ExpiredImagePlaceholder
                label={t('Image expired')}
                description={t('This temporary image link has expired.')}
              />
            ) : (
              <img
                src={image.src}
                alt={`generated-${index + 1}`}
                className='h-auto max-h-[70vh] w-full object-contain transition-transform duration-200 group-hover:scale-[1.02]'
                onError={() => handleImageLoadError(image)}
              />
            )}
            <div className='absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent px-3 py-2 opacity-0 transition-opacity duration-200 group-hover:opacity-100'>
              <span className='text-xs text-white'>
                {image.isExpired ? t('Image expired') : t('Click image to preview')}
              </span>
            </div>
            </div>
          </button>
          <div className='text-muted-foreground flex items-center justify-between gap-3 text-xs'>
            <span>{t('Image')} #{index + 1}</span>
            <div className='flex items-center gap-3'>
              {image.sourceType === 'url' && <span>{t('Temporary image URL')}</span>}
              {image.elapsed && <span>{image.elapsed}</span>}
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='h-7 w-7'
                onClick={() => handleSaveImage(image, index)}
                disabled={image.isExpired}
              >
                <Download className='h-4 w-4' />
                <span className='sr-only'>{t('Save image')}</span>
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
          <h1 className='text-2xl font-semibold tracking-tight'>{t('Text-to-Image')}</h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Generate images from prompts with model parameters')}
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t('Generation settings')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
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
                placeholder={t('Describe the image you want to generate')}
                rows={4}
              />
            </div>

            <Collapsible open={showAdvancedSettings} onOpenChange={setShowAdvancedSettings}>
              <div className='flex items-center justify-between rounded-md border border-dashed px-4 py-3'>
                <div>
                  <div className='text-sm font-medium'>{t('Advanced settings')}</div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Configure image size, count, steps, seed, and cfg.')}
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
                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('Image size')}</Label>
                    <Select value={size} onValueChange={(value) => setSize(value ?? IMAGE_SIZE_OPTIONS[0])}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        {IMAGE_SIZE_OPTIONS.map((option) => (
                          <SelectItem key={option} value={option}>
                            {option}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
                  <div className='space-y-2'>
                    <Label>{t('Image count')}</Label>
                    <Input
                      type='number'
                      min={1}
                      max={8}
                      value={n}
                      onChange={(e) => setN(Number(e.target.value || 1))}
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('num_inference_steps')}</Label>
                    <Input
                      type='number'
                      min={1}
                      value={steps}
                      onChange={(e) => setSteps(Number(e.target.value || 1))}
                    />
                  </div>
                  <div />
                </div>

                <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                  <div className='space-y-2'>
                    <Label>{t('seed')}</Label>
                    <Input
                      type='number'
                      value={seed}
                      onChange={(e) => setSeed(Number(e.target.value || 0))}
                    />
                  </div>
                  <div className='space-y-2'>
                    <Label>{t('cfg')}</Label>
                    <Input
                      type='number'
                      step='0.1'
                      value={cfg}
                      onChange={(e) => setCfg(Number(e.target.value || 0))}
                    />
                  </div>
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
            <CardTitle>{t('Generated images')}</CardTitle>
          </CardHeader>
          <CardContent>
            {generatedNotice && (
              <p className='mb-4 text-sm text-amber-600'>
                {generatedNotice}
              </p>
            )}
            {isGenerating ? (
              <div className='flex min-h-[280px] flex-col items-center justify-center rounded-lg border border-dashed bg-muted/20 px-6 py-10 text-center'>
                <Loader2 className='text-primary h-8 w-8 animate-spin' />
                <p className='mt-4 text-sm font-medium'>{t('Generating images...')}</p>
                <p className='text-muted-foreground mt-2 text-sm'>
                  {t('Please wait while the model generates your images.')}
                </p>
                {pendingRequest && (
                  <div className='text-muted-foreground mt-4 flex items-center gap-2 text-xs'>
                    <Clock3 className='h-3.5 w-3.5' />
                    <span>
                      {t('Current task')}: {pendingRequest.model} · {pendingRequest.size} · n=
                      {pendingRequest.n}
                    </span>
                  </div>
                )}
              </div>
            ) : images.length === 0 ? (
              <p className='text-muted-foreground text-sm'>
                {t('No images yet. Enter a prompt and click generate.')}
              </p>
            ) : (
              renderImageGrid(images)
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='flex flex-row items-center justify-between gap-3'>
            <CardTitle className='flex items-center gap-2'>
              <History className='h-4 w-4' />
              {t('Generation history')}
            </CardTitle>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={clearHistory}
              disabled={history.length === 0}
            >
              <Trash2 className='mr-2 h-4 w-4' />
              {t('Clear history')}
            </Button>
          </CardHeader>
          <CardContent className='space-y-4'>
            {history.length === 0 ? (
              <p className='text-muted-foreground text-sm'>
                {t('No generation history yet.')}
              </p>
            ) : (
              history.map((item) => (
                <div key={item.id} className='space-y-3 rounded-lg border p-4'>
                  <div className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
                    <div className='space-y-1'>
                      <div className='text-sm font-medium'>{item.request.model}</div>
                      <div className='text-muted-foreground text-xs'>
                        {new Date(item.createdAt).toLocaleString()}
                      </div>
                    </div>
                    <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                      <span>n={item.request.n}</span>
                      <span>{item.request.size}</span>
                      <span>{t('seed')}: {item.request.seed}</span>
                      <span>{t('cfg')}: {item.request.cfg}</span>
                    </div>
                  </div>
                  <p className='text-sm break-words'>{item.request.prompt}</p>
                  {item.images.some((image) => image.sourceType === 'url') && (
                    <p className='text-amber-600 text-xs'>
                      {t('Some historical images use temporary URLs and may expire.')}
                    </p>
                  )}
                  {renderImageGrid(item.images)}
                </div>
              ))
            )}
          </CardContent>
        </Card>

        <Dialog open={Boolean(previewImage)} onOpenChange={(open) => !open && setPreviewImage(null)}>
          <DialogContent className='w-[96vw] max-w-7xl sm:max-w-7xl'>
            <DialogHeader>
              <div className='flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
                <DialogTitle>{t('Image Preview')}</DialogTitle>
                <div className='flex flex-wrap items-center gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => updatePreviewScale((current) => current - PREVIEW_SCALE_STEP)}
                    disabled={previewScale <= MIN_PREVIEW_SCALE}
                  >
                    <ZoomOut className='mr-2 h-4 w-4' />
                    {t('Zoom out')}
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => previewImage && handleSaveImage(previewImage, 0)}
                    disabled={!previewImage || previewImage.isExpired}
                  >
                    <Download className='mr-2 h-4 w-4' />
                    {t('Save image')}
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => setPreviewScale(1)}
                    disabled={previewScale === 1}
                  >
                    <Search className='mr-2 h-4 w-4' />
                    {t('Reset zoom')}
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => updatePreviewScale((current) => current + PREVIEW_SCALE_STEP)}
                    disabled={previewScale >= MAX_PREVIEW_SCALE}
                  >
                    <ZoomIn className='mr-2 h-4 w-4' />
                    {t('Zoom in')}
                  </Button>
                  <span className='text-muted-foreground min-w-16 text-right text-sm'>
                    {Math.round(previewScale * 100)}%
                  </span>
                </div>
              </div>
            </DialogHeader>
            {previewImage && (
              <div
                className='max-h-[80vh] overflow-auto rounded-md border bg-muted/10'
                onWheel={handlePreviewWheel}
              >
                {previewImage.isExpired ? (
                  <div className='min-h-[60vh]'>
                    <ExpiredImagePlaceholder
                      label={t('Image expired')}
                      description={t('This temporary image link has expired.')}
                    />
                  </div>
                ) : (
                  <img
                    src={previewImage.src}
                    alt='preview'
                    className='mx-auto block max-w-none object-contain transition-transform duration-150'
                    style={{ transform: `scale(${previewScale})`, transformOrigin: 'center top' }}
                    onError={() => handleImageLoadError(previewImage)}
                  />
                )}
              </div>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </ScrollArea>
  )
}
