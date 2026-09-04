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
import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Download, Loader2, Play, Volume2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'

import { getUserModels, synthesizeSpeech } from './api'
import type { SpeechSynthesisRequest } from './types'

const RESPONSE_FORMATS: SpeechSynthesisRequest['response_format'][] = ['wav', 'mp3', 'opus', 'aac', 'flac', 'pcm']

export function SpeechSynthesisPlayground() {
  const { t } = useTranslation()
  const [model, setModel] = useState('')
  const [input, setInput] = useState(t('Hello, this is a speech synthesis test.'))
  const [voice, setVoice] = useState('default')
  const [responseFormat, setResponseFormat] = useState<SpeechSynthesisRequest['response_format']>('wav')
  const [speed, setSpeed] = useState('1')
  const [audioUrl, setAudioUrl] = useState('')
  const [isSynthesizing, setIsSynthesizing] = useState(false)

  const { data: models = [], isLoading: isLoadingModels } = useQuery({
    queryKey: ['speech-synthesis-models'],
    queryFn: getUserModels,
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

  useEffect(() => () => {
    if (audioUrl) URL.revokeObjectURL(audioUrl)
  }, [audioUrl])

  const handleSynthesize = async () => {
    if (!model) {
      toast.error(t('Please select a model'))
      return
    }
    if (!input.trim()) {
      toast.error(t('Please enter text to synthesize'))
      return
    }
    const parsedSpeed = Number(speed)
    if (!Number.isFinite(parsedSpeed) || parsedSpeed <= 0) {
      toast.error(t('Please enter a valid speed'))
      return
    }

    setIsSynthesizing(true)
    try {
      const audio = await synthesizeSpeech({
        model,
        input: input.trim(),
        voice: voice.trim() || 'default',
        response_format: responseFormat,
        speed: parsedSpeed,
      })
      if (audio.size === 0) {
        throw new Error(t('No audio was returned'))
      }
      const nextAudioUrl = URL.createObjectURL(audio)
      setAudioUrl((currentAudioUrl) => {
        if (currentAudioUrl) URL.revokeObjectURL(currentAudioUrl)
        return nextAudioUrl
      })
    } catch (error) {
      const message = error instanceof Error ? error.message : t('Speech synthesis failed')
      toast.error(message)
    } finally {
      setIsSynthesizing(false)
    }
  }

  const handleDownload = () => {
    if (!audioUrl) return
    const link = document.createElement('a')
    link.href = audioUrl
    link.download = `speech.${responseFormat}`
    link.click()
  }

  return (
    <ScrollArea className='min-h-0 flex-1'>
      <div className='mx-auto flex w-full max-w-6xl flex-col gap-4 p-4 md:p-6'>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>{t('Speech Synthesis')}</h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Generate audio from text with a supported model.')}
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t('Generation settings')}</CardTitle>
            <CardDescription>{t('Select a model and configure the speech output.')}</CardDescription>
          </CardHeader>
          <CardContent className='space-y-5'>
            <div className='space-y-2'>
              <Label htmlFor='speech-synthesis-model'>{t('Model')}</Label>
              <Select value={model} onValueChange={(value) => setModel(value ?? '')} disabled={isLoadingModels || models.length === 0}>
                <SelectTrigger id='speech-synthesis-model'>
                  <SelectValue placeholder={t('Select a model')} />
                </SelectTrigger>
                <SelectContent>
                  {models.map((option) => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {!isLoadingModels && models.length === 0 && (
                <p className='text-muted-foreground text-sm'>{t('No speech synthesis models are available.')}</p>
              )}
            </div>

            <div className='space-y-2'>
              <Label htmlFor='speech-synthesis-input'>{t('Text')}</Label>
              <Textarea id='speech-synthesis-input' value={input} onChange={(event) => setInput(event.target.value)} rows={6} />
            </div>

            <div className='grid gap-4 sm:grid-cols-3'>
              <div className='space-y-2'>
                <Label htmlFor='speech-synthesis-voice'>{t('Voice')}</Label>
                <Input id='speech-synthesis-voice' value={voice} onChange={(event) => setVoice(event.target.value)} />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='speech-synthesis-format'>{t('Response format')}</Label>
                <Select value={responseFormat} onValueChange={(value) => setResponseFormat(value as SpeechSynthesisRequest['response_format'])}>
                  <SelectTrigger id='speech-synthesis-format'><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {RESPONSE_FORMATS.map((format) => <SelectItem key={format} value={format}>{format.toUpperCase()}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-2'>
                <Label htmlFor='speech-synthesis-speed'>{t('Speed')}</Label>
                <Input id='speech-synthesis-speed' type='number' min='0.25' step='0.05' value={speed} onChange={(event) => setSpeed(event.target.value)} />
              </div>
            </div>

            <Button className='w-full' onClick={handleSynthesize} disabled={isSynthesizing || !model || !input.trim()}>
              {isSynthesizing ? <Loader2 className='animate-spin' aria-hidden='true' /> : <Volume2 aria-hidden='true' />}
              {isSynthesizing ? t('Synthesizing...') : t('Synthesize speech')}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='flex flex-row items-center justify-between gap-3'>
            <CardTitle>{t('Generated audio')}</CardTitle>
            <Button type='button' variant='outline' size='sm' onClick={handleDownload} disabled={!audioUrl}>
              <Download aria-hidden='true' />
              {t('Download')}
            </Button>
          </CardHeader>
          <CardContent>
            {audioUrl ? (
              <audio className='w-full' controls src={audioUrl}>{t('Your browser does not support audio playback.')}</audio>
            ) : (
              <div className='text-muted-foreground flex min-h-40 flex-col items-center justify-center gap-3 text-center text-sm'>
                <Play className='size-10' aria-hidden='true' />
                <p>{t('Your generated audio will appear here.')}</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </ScrollArea>
  )
}