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
import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AudioLines, History, Loader2, Trash2, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { useSpeechRecognitionStore } from '@/stores/speech-recognition-store'

import { getUserModels, recognizeSpeech } from './api'
import type { SpeechRecognitionRequest } from './types'

const ACCEPTED_AUDIO_TYPES = 'audio/*,.mp3,.flac,.wav,.m4a,.ogg,.aac,.opus,.webm'

function normalizeSpeechTranscript(content: string): string {
  return content
    .replace(/^\s*(?:language\s+[^<]*\s*)?<asr_text>\s*/i, '')
    .replace(/\s*<\/asr_text>\s*$/i, '')
    .trim()
}

function getAudioFormat(file: File): string {
  const extension = file.name.split('.').pop()?.trim().toLowerCase()
  if (extension) return extension

  const mimeType = file.type.split('/')[1]?.toLowerCase()
  return mimeType || 'mp3'
}

function readAudioAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('error', () => reject(new Error('Failed to read audio file')), { once: true })
    reader.addEventListener('load', () => {
      if (typeof reader.result !== 'string') {
        reject(new Error('Failed to read audio file'))
        return
      }
      const [, base64] = reader.result.split(',', 2)
      if (!base64) {
        reject(new Error('Failed to encode audio file'))
        return
      }
      resolve(base64)
    }, { once: true })
    reader.readAsDataURL(file)
  })
}

export function SpeechRecognitionPlayground() {
  const { t } = useTranslation()
  const { history, clearHistory } = useSpeechRecognitionStore()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [model, setModel] = useState('')
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [showAllModels, setShowAllModels] = useState(false)
  const [transcript, setTranscript] = useState('')
  const [isRecognizing, setIsRecognizing] = useState(false)

  const { data: models = [], isLoading: isLoadingModels } = useQuery({
    queryKey: ['speech-recognition-models', showAllModels],
    queryFn: () => getUserModels(showAllModels),
  })

  useEffect(() => {
    if (models.length === 0) {
      setModel('')
      return
    }
    if (!models.some((option) => option.value === model)) {
      setModel(models[0].value)
    }
  }, [model, models])

  const handleAudioChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] ?? null
    if (!nextFile) return
    setAudioFile(nextFile)
    setTranscript('')
    event.target.value = ''
  }

  const handleRecognize = async () => {
    if (!model) {
      toast.error(t('Please select a model'))
      return
    }
    if (!audioFile) {
      toast.error(t('Please upload an audio file'))
      return
    }

    setIsRecognizing(true)
    setTranscript('')
    try {
      const audioData = await readAudioAsBase64(audioFile)
      const payload: SpeechRecognitionRequest = {
        model,
        messages: [
          {
            role: 'user',
            content: [
              { type: 'text', text: t('Please transcribe this audio.') },
              {
                type: 'input_audio',
                input_audio: { data: audioData, format: getAudioFormat(audioFile) },
              },
            ],
          },
        ],
        max_tokens: 1024,
        stream: false,
      }
      const response = await recognizeSpeech(payload)
      const content = response.choices?.[0]?.message?.content
      const transcript = content ? normalizeSpeechTranscript(content) : ''
      if (!transcript) {
        throw new Error(t('No transcription was returned'))
      }
      setTranscript(transcript)
      useSpeechRecognitionStore.setState((state) => ({
        history: [
          {
            id: `${Date.now()}-${model}`,
            createdAt: Date.now(),
            model,
            audioFileName: audioFile.name,
            audioFormat: getAudioFormat(audioFile),
            transcript,
          },
          ...state.history,
        ].slice(0, 20),
      }))
    } catch (error) {
      const message = error instanceof Error ? error.message : t('Speech recognition failed')
      toast.error(message)
    } finally {
      setIsRecognizing(false)
    }
  }

  return (
    <ScrollArea className='min-h-0 flex-1'>
      <div className='mx-auto flex w-full max-w-6xl flex-col gap-4 p-4 md:p-6'>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>{t('Speech Recognition')}</h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Upload an audio file to transcribe it with a supported model.')}
          </p>
        </div>
      <Card>
        <CardHeader>
          <CardTitle>{t('Generation settings')}</CardTitle>
          <CardDescription>{t('Select a model and upload audio to transcribe.')}</CardDescription>
        </CardHeader>
        <CardContent className='space-y-5'>
          <div className='space-y-2'>
            <div className='flex items-center justify-between gap-2'>
              <Label htmlFor='speech-model'>{t('Model')}</Label>
              <div className='flex items-center gap-2'>
                <Label className='text-muted-foreground text-xs'>{t('Show all models')}</Label>
                <Switch checked={showAllModels} onCheckedChange={setShowAllModels} />
              </div>
            </div>
            <Select
              value={model}
              onValueChange={(value) => setModel(value ?? '')}
              disabled={isLoadingModels || models.length === 0}
            >
              <SelectTrigger id='speech-model'>
                <SelectValue placeholder={t('Select a model')} />
              </SelectTrigger>
              <SelectContent>
                {models.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {!isLoadingModels && models.length === 0 && (
              <p className='text-muted-foreground text-sm'>{t('No speech recognition models are available.')}</p>
            )}
          </div>

          <div className='space-y-2'>
            <Label htmlFor='speech-audio'>{t('Audio file')}</Label>
            <input
              ref={fileInputRef}
              id='speech-audio'
              type='file'
              accept={ACCEPTED_AUDIO_TYPES}
              className='sr-only'
              onChange={handleAudioChange}
            />
            {audioFile ? (
              <div className='border-input flex min-h-12 items-center gap-3 rounded-lg border bg-muted/30 px-3 py-2'>
                <AudioLines className='size-5 shrink-0 text-primary' aria-hidden='true' />
                <div className='min-w-0 flex-1'>
                  <p className='truncate text-sm font-medium'>{audioFile.name}</p>
                  <p className='text-muted-foreground text-xs'>{getAudioFormat(audioFile).toUpperCase()}</p>
                </div>
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  onClick={() => fileInputRef.current?.click()}
                >
                  {t('Change')}
                </Button>
              </div>
            ) : (
              <Button
                type='button'
                variant='outline'
                className='h-12 w-full border-dashed'
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload aria-hidden='true' />
                {t('Upload')}
              </Button>
            )}
          </div>

          <Button className='w-full' onClick={handleRecognize} disabled={isRecognizing || !audioFile || !model}>
            {isRecognizing ? <Loader2 className='animate-spin' aria-hidden='true' /> : <Upload aria-hidden='true' />}
            {isRecognizing ? t('Transcribing...') : t('Transcribe audio')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Transcription')}</CardTitle>
        </CardHeader>
        <CardContent>
          {transcript ? (
            <p className='whitespace-pre-wrap break-words leading-7'>{transcript}</p>
          ) : (
            <div className='text-muted-foreground flex min-h-52 flex-col items-center justify-center gap-3 text-center text-sm'>
              <AudioLines className='size-10' aria-hidden='true' />
              <p>{t('Your transcription will appear here.')}</p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className='flex flex-row items-center justify-between gap-3'>
          <CardTitle className='flex items-center gap-2'>
            <History className='h-4 w-4' />
            {t('Transcription history')}
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
            <p className='text-muted-foreground text-sm'>{t('No transcription history yet.')}</p>
          ) : (
            history.map((item) => (
              <div key={item.id} className='space-y-3 rounded-lg border p-4'>
                <div className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
                  <div className='min-w-0 space-y-1'>
                    <p className='truncate text-sm font-medium'>{item.audioFileName}</p>
                    <p className='text-muted-foreground text-xs'>{item.model}</p>
                  </div>
                  <div className='text-muted-foreground flex shrink-0 gap-3 text-xs'>
                    <span>{item.audioFormat.toUpperCase()}</span>
                    <span>{new Date(item.createdAt).toLocaleString()}</span>
                  </div>
                </div>
                <p className='whitespace-pre-wrap break-words text-sm leading-7'>{item.transcript}</p>
              </div>
            ))
          )}
        </CardContent>
      </Card>
      </div>
    </ScrollArea>
  )
}