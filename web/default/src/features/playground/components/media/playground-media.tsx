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
import { useQuery } from '@tanstack/react-query'
import { Download, ImageIcon, Loader2, Play, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ModelGroupSelector } from '@/components/model-group-selector'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  createVideo,
  generateImages,
  getPlaygroundMediaHistory,
  getVideoTask,
  uploadPlaygroundAsset,
} from '../../api'
import { PLAYGROUND_MODES } from '../../constants'
import { usePlaygroundOptions } from '../../hooks/use-playground-options'
import type {
  GroupOption,
  ImageGenerationResult,
  ModelOption,
  PlaygroundConfig,
  PlaygroundMode,
  VideoTaskResponse,
} from '../../types'
import { getVideoModelProfile } from '../../video-model-profiles'
import { PlaygroundMediaHistory } from './playground-media-history'
import {
  VideoReferenceInputs,
  type VideoReferenceFiles,
  type VideoReferenceURLs,
} from './video-reference-inputs'

const VIDEO_POLL_INTERVAL_MS = 2000
const TERMINAL_VIDEO_STATUSES = new Set(['completed', 'failed', 'cancelled'])

type MediaConfig = Pick<PlaygroundConfig, 'group' | 'model'>

interface PlaygroundMediaProps {
  mode: Exclude<PlaygroundMode, 'chat'>
  groups: GroupOption[]
  setGroups: (groups: GroupOption[]) => void
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object') {
    const response = (error as { response?: { data?: unknown } }).response
    const data = response?.data
    if (data && typeof data === 'object') {
      const apiError = (data as { error?: { message?: unknown } }).error
      if (typeof apiError?.message === 'string') return apiError.message
      const message = (data as { message?: unknown }).message
      if (typeof message === 'string') return message
    }
  }
  return error instanceof Error && error.message ? error.message : fallback
}

function imageSource(image: ImageGenerationResult): string {
  if (image.url) {
    try {
      const url = new URL(image.url, window.location.origin)
      if (url.protocol === 'http:' || url.protocol === 'https:') {
        return image.url
      }
    } catch {
      return ''
    }
  }
  if (image.b64_json) return `data:image/png;base64,${image.b64_json}`
  return ''
}

export function PlaygroundMedia(props: PlaygroundMediaProps) {
  const { t } = useTranslation()
  const [config, setConfig] = useState<MediaConfig>({
    group: 'default',
    model: '',
  })
  const [models, setModels] = useState<ModelOption[]>([])
  const [prompt, setPrompt] = useState('')
  const [size, setSize] = useState(
    props.mode === PLAYGROUND_MODES.IMAGE ? '1024x1024' : '720x1280'
  )
  const [quality, setQuality] = useState('standard')
  const [seconds, setSeconds] = useState('4')
  const [aspectRatio, setAspectRatio] = useState('16:9')
  const [resolution, setResolution] = useState('720p')
  const [generateAudio, setGenerateAudio] = useState(false)
  const [watermark, setWatermark] = useState(false)
  const [referenceFiles, setReferenceFiles] = useState<VideoReferenceFiles>({
    images: [],
    videos: [],
    audios: [],
  })
  const [referenceURLs, setReferenceURLs] = useState<VideoReferenceURLs>({
    images: [],
    videos: [],
    audios: [],
  })
  const [images, setImages] = useState<ImageGenerationResult[]>([])
  const [task, setTask] = useState<VideoTaskResponse | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const historyQuery = useQuery({
    queryKey: ['playground-media-history', props.mode],
    queryFn: ({ signal }) => getPlaygroundMediaHistory(props.mode, signal),
    refetchInterval: (query) => {
      if (props.mode !== PLAYGROUND_MODES.VIDEO) return false
      const items = query.state.data ?? []
      const hasPendingTask = items.some(
        (item) => !TERMINAL_VIDEO_STATUSES.has(item.status.toLowerCase())
      )
      return hasPendingTask ? 5000 : false
    },
  })
  const refetchHistory = historyQuery.refetch
  const isImage = props.mode === PLAYGROUND_MODES.IMAGE
  const videoProfile = useMemo(
    () => getVideoModelProfile(config.model),
    [config.model]
  )
  const durationOptions = useMemo(() => {
    if (
      videoProfile.provider === 'grok' &&
      referenceFiles.images.length + referenceURLs.images.length > 1
    ) {
      return videoProfile.durations.filter((duration) => duration <= 10)
    }
    return videoProfile.durations
  }, [referenceFiles.images.length, referenceURLs.images.length, videoProfile])
  const minimumVideoDuration = durationOptions[0] ?? 1
  const maximumVideoDuration = durationOptions.at(-1) ?? 60
  const videoDuration = Number(seconds)
  const isVideoDurationValid =
    Number.isInteger(videoDuration) &&
    videoDuration >= minimumVideoDuration &&
    videoDuration <= maximumVideoDuration &&
    (durationOptions.length === 0 || durationOptions.includes(videoDuration))

  const updateConfig = useCallback(
    <K extends keyof MediaConfig>(key: K, value: MediaConfig[K]) => {
      setConfig((current) => ({ ...current, [key]: value }))
    },
    []
  )

  const { isLoadingModels } = usePlaygroundOptions({
    currentGroup: config.group,
    currentModel: config.model,
    mode: props.mode,
    setGroups: props.setGroups,
    setModels,
    updateConfig,
  })

  const taskId = task?.id || task?.task_id || ''
  const normalizedStatus = task?.status?.toLowerCase() || ''
  const isPolling =
    taskId !== '' &&
    normalizedStatus !== '' &&
    !TERMINAL_VIDEO_STATUSES.has(normalizedStatus)
  const progressValue = Math.min(100, Math.max(0, Number(task?.progress) || 0))
  const videoUrl = taskId ? `/v1/videos/${taskId}/content` : ''

  useEffect(() => {
    if (!isPolling || !taskId) return

    let stopped = false
    let controller: AbortController | null = null
    let timer: number | null = null

    const poll = async () => {
      controller = new AbortController()
      try {
        const nextTask = await getVideoTask(taskId, controller.signal)
        if (stopped) return
        setTask(nextTask)
        if (TERMINAL_VIDEO_STATUSES.has(nextTask.status.toLowerCase())) {
          void refetchHistory()
        } else {
          timer = window.setTimeout(poll, VIDEO_POLL_INTERVAL_MS)
        }
      } catch (error) {
        if (stopped || controller.signal.aborted) return
        toast.error(getErrorMessage(error, t('Failed to poll video task')))
        timer = window.setTimeout(poll, VIDEO_POLL_INTERVAL_MS)
      }
    }

    timer = window.setTimeout(poll, VIDEO_POLL_INTERVAL_MS)

    return () => {
      stopped = true
      if (timer !== null) window.clearTimeout(timer)
      controller?.abort()
    }
  }, [isPolling, refetchHistory, taskId, t])

  useEffect(
    () => () => {
      abortControllerRef.current?.abort()
    },
    []
  )

  useEffect(() => {
    setSeconds((current) => {
      const duration = Number(current)
      if (
        !Number.isFinite(duration) ||
        (durationOptions.length > 0 && !durationOptions.includes(duration)) ||
        duration < minimumVideoDuration
      ) {
        return String(minimumVideoDuration)
      }
      if (duration > maximumVideoDuration) {
        return String(maximumVideoDuration)
      }
      return current
    })
    setAspectRatio((current) =>
      videoProfile.aspectRatios.includes(current)
        ? current
        : (videoProfile.aspectRatios[0] ?? '16:9')
    )
    setResolution((current) =>
      videoProfile.resolutions.length === 0 ||
      videoProfile.resolutions.includes(current)
        ? current
        : (videoProfile.resolutions[0] ?? '720p')
    )
    setReferenceFiles((current) => ({
      images: current.images.slice(0, videoProfile.maxImages),
      videos: current.videos.slice(0, videoProfile.maxVideos),
      audios: current.audios.slice(0, videoProfile.maxAudios),
    }))
    setReferenceURLs((current) => ({
      images: current.images.slice(
        0,
        Math.max(
          0,
          videoProfile.maxImages -
            Math.min(referenceFiles.images.length, videoProfile.maxImages)
        )
      ),
      videos: current.videos.slice(
        0,
        Math.max(
          0,
          videoProfile.maxVideos -
            Math.min(referenceFiles.videos.length, videoProfile.maxVideos)
        )
      ),
      audios: current.audios.slice(
        0,
        Math.max(
          0,
          videoProfile.maxAudios -
            Math.min(referenceFiles.audios.length, videoProfile.maxAudios)
        )
      ),
    }))
    if (!videoProfile.supportsGenerateAudio) setGenerateAudio(false)
    if (!videoProfile.supportsWatermark) setWatermark(false)
  }, [
    durationOptions,
    maximumVideoDuration,
    minimumVideoDuration,
    referenceFiles.audios.length,
    referenceFiles.images.length,
    referenceFiles.videos.length,
    videoProfile,
  ])

  const hasRequiredReferences =
    !videoProfile.requiresImage ||
    referenceFiles.images.length + referenceURLs.images.length === 1

  const canSubmit =
    !isSubmitting &&
    !isLoadingModels &&
    config.model !== '' &&
    prompt.trim() !== '' &&
    (isImage || (isVideoDurationValid && hasRequiredReferences))

  const handleSubmit = async () => {
    if (!canSubmit) return
    abortControllerRef.current?.abort()
    const controller = new AbortController()
    abortControllerRef.current = controller
    setIsSubmitting(true)

    try {
      if (props.mode === PLAYGROUND_MODES.IMAGE) {
        const response = await generateImages(
          {
            model: config.model,
            group: config.group,
            prompt: prompt.trim(),
            size,
            quality,
            n: 1,
            response_format: 'url',
          },
          controller.signal
        )
        setImages(response.data ?? [])
        void refetchHistory()
        return
      }

      const [uploadedImages, uploadedVideos, uploadedAudios] =
        await Promise.all([
          Promise.all(
            referenceFiles.images.map((file) =>
              uploadPlaygroundAsset(file, 'image', controller.signal)
            )
          ),
          Promise.all(
            referenceFiles.videos.map((file) =>
              uploadPlaygroundAsset(file, 'video', controller.signal)
            )
          ),
          Promise.all(
            referenceFiles.audios.map((file) =>
              uploadPlaygroundAsset(file, 'audio', controller.signal)
            )
          ),
        ])
      const videoPayload = {
        model: config.model,
        group: config.group,
        prompt: prompt.trim(),
        seconds,
        ...(videoProfile.provider === 'generic'
          ? { size }
          : {
              aspect_ratio: aspectRatio,
              resolution:
                videoProfile.resolutions.length > 0 ? resolution : undefined,
              images: [
                ...referenceURLs.images,
                ...uploadedImages.map((asset) => asset.url),
              ],
              videos: [
                ...referenceURLs.videos,
                ...uploadedVideos.map((asset) => asset.url),
              ],
              audios: [
                ...referenceURLs.audios,
                ...uploadedAudios.map((asset) => asset.url),
              ],
              generate_audio: videoProfile.supportsGenerateAudio
                ? generateAudio
                : undefined,
              watermark: videoProfile.supportsWatermark ? watermark : undefined,
            }),
      }
      const response = await createVideo(videoPayload, controller.signal)
      setTask(response)
      void refetchHistory()
    } catch (error) {
      if (!controller.signal.aborted) {
        toast.error(getErrorMessage(error, t('Media generation failed')))
      }
    } finally {
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
      setIsSubmitting(false)
    }
  }

  const statusLabel = useMemo(() => {
    if (!task) return ''
    if (normalizedStatus === 'completed') return t('Completed')
    if (normalizedStatus === 'failed') return t('Failed')
    if (normalizedStatus === 'cancelled') return t('Cancelled')
    if (normalizedStatus === 'queued') return t('Queued')
    return t('Processing')
  }, [normalizedStatus, task, t])

  let resultContent = (
    <MediaEmpty icon={Play} label={t('Generated video will appear here')} />
  )
  if (isImage) {
    resultContent =
      images.length > 0 ? (
        <div className='grid gap-4 sm:grid-cols-2'>
          {images.map((image) => {
            const src = imageSource(image)
            if (!src) return null
            const imageKey =
              image.url || image.b64_json || image.revised_prompt || src
            return (
              <figure key={imageKey} className='space-y-2'>
                <div className='bg-muted/30 aspect-square overflow-hidden rounded-lg border'>
                  <img
                    src={src}
                    alt={image.revised_prompt || t('Generated image')}
                    className='size-full object-contain'
                  />
                </div>
                <Button
                  variant='outline'
                  size='sm'
                  render={<a href={src} download='generated-image.png' />}
                >
                  <Download />
                  {t('Download')}
                </Button>
              </figure>
            )
          })}
        </div>
      ) : (
        <MediaEmpty
          icon={ImageIcon}
          label={t('Generated images will appear here')}
        />
      )
  } else if (task) {
    resultContent = (
      <div className='space-y-4'>
        <div className='flex items-center justify-between gap-3'>
          <div>
            <p className='font-medium'>{statusLabel}</p>
            <p className='text-muted-foreground font-mono text-xs break-all'>
              {taskId}
            </p>
          </div>
          {isPolling && (
            <Loader2 className='text-muted-foreground animate-spin' />
          )}
        </div>
        <Progress
          value={normalizedStatus === 'completed' ? 100 : progressValue}
        />
        {normalizedStatus === 'completed' && videoUrl && (
          <>
            <video
              className='aspect-video w-full rounded-lg border bg-black'
              src={videoUrl}
              controls
              preload='metadata'
            />
            <Button
              variant='outline'
              render={
                <a href={videoUrl} download={`${taskId || 'video'}.mp4`} />
              }
            >
              <Download />
              {t('Download video')}
            </Button>
          </>
        )}
        {normalizedStatus === 'failed' && (
          <p className='text-destructive text-sm'>
            {task.error?.message || t('Video generation failed')}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className='mx-auto flex size-full max-w-5xl flex-col gap-5 overflow-y-auto px-4 py-5 sm:px-6'>
      <div className='grid gap-4 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]'>
        <section className='space-y-4 border-b pb-5 lg:border-r lg:border-b-0 lg:pr-5 lg:pb-0'>
          <ModelGroupSelector
            selectedModel={config.model}
            models={models}
            onModelChange={(value) => updateConfig('model', value)}
            selectedGroup={config.group}
            groups={props.groups}
            onGroupChange={(value) => updateConfig('group', value)}
            disabled={isSubmitting || isLoadingModels}
          />

          <div className='space-y-2'>
            <Label htmlFor={`${props.mode}-prompt`}>{t('Prompt')}</Label>
            <Textarea
              id={`${props.mode}-prompt`}
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              placeholder={
                isImage
                  ? t('Describe the image you want to create')
                  : t('Describe the video you want to create')
              }
              className='min-h-36 resize-y'
            />
          </div>

          {isImage ? (
            <div className='grid grid-cols-2 gap-3'>
              <div className='space-y-2'>
                <Label>{t('Size')}</Label>
                <Select
                  value={size}
                  onValueChange={(value) => value && setSize(value)}
                >
                  <SelectTrigger>
                    <SelectValue>{size}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {['1024x1024', '1024x1792', '1792x1024'].map((value) => (
                      <SelectItem key={value} value={value}>
                        {value}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-2'>
                <Label>{t('Quality')}</Label>
                <Select
                  value={quality}
                  onValueChange={(value) => value && setQuality(value)}
                >
                  <SelectTrigger>
                    <SelectValue>{t(quality)}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='standard'>{t('standard')}</SelectItem>
                    <SelectItem value='hd'>{t('hd')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          ) : (
            <div className='space-y-4'>
              <VideoReferenceInputs
                profile={videoProfile}
                files={referenceFiles}
                urls={referenceURLs}
                disabled={isSubmitting}
                onChange={setReferenceFiles}
                onURLsChange={setReferenceURLs}
              />
              <div className='grid grid-cols-2 gap-3'>
                <div className='space-y-2'>
                  <Label>{t('Duration')}</Label>
                  {durationOptions.length > 0 ? (
                    <Select
                      value={seconds}
                      onValueChange={(value) => value && setSeconds(value)}
                    >
                      <SelectTrigger>
                        <SelectValue>{seconds}s</SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {durationOptions.map((duration) => (
                          <SelectItem key={duration} value={String(duration)}>
                            {duration}s
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : (
                    <Input
                      type='number'
                      min={minimumVideoDuration}
                      max={maximumVideoDuration}
                      value={seconds}
                      onChange={(event) => setSeconds(event.target.value)}
                      aria-label={t('Duration in seconds')}
                    />
                  )}
                </div>
                <div className='space-y-2'>
                  <Label>{t('Aspect ratio')}</Label>
                  <Select
                    value={aspectRatio}
                    onValueChange={(value) => value && setAspectRatio(value)}
                  >
                    <SelectTrigger>
                      <SelectValue>{aspectRatio}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {videoProfile.aspectRatios.map((ratio) => (
                        <SelectItem key={ratio} value={ratio}>
                          {ratio}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {videoProfile.resolutions.length > 0 && (
                  <div className='space-y-2'>
                    <Label>{t('Resolution')}</Label>
                    <Select
                      value={resolution}
                      onValueChange={(value) => value && setResolution(value)}
                    >
                      <SelectTrigger>
                        <SelectValue>{resolution}</SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {videoProfile.resolutions.map((value) => (
                          <SelectItem key={value} value={value}>
                            {value}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </div>
              {(videoProfile.supportsGenerateAudio ||
                videoProfile.supportsWatermark) && (
                <div className='grid gap-3 sm:grid-cols-2'>
                  {videoProfile.supportsGenerateAudio && (
                    <div className='flex items-center justify-between gap-3 rounded-lg border px-3 py-2'>
                      <Label htmlFor='video-generate-audio'>
                        {t('Generate audio')}
                      </Label>
                      <Switch
                        id='video-generate-audio'
                        checked={generateAudio}
                        onCheckedChange={setGenerateAudio}
                      />
                    </div>
                  )}
                  {videoProfile.supportsWatermark && (
                    <div className='flex items-center justify-between gap-3 rounded-lg border px-3 py-2'>
                      <Label htmlFor='video-watermark'>{t('Watermark')}</Label>
                      <Switch
                        id='video-watermark'
                        checked={watermark}
                        onCheckedChange={setWatermark}
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          <Button
            className='w-full'
            disabled={!canSubmit}
            onClick={handleSubmit}
          >
            {isSubmitting ? <Loader2 className='animate-spin' /> : <Sparkles />}
            {isSubmitting ? t('Generating') : t('Generate')}
          </Button>
        </section>

        <section className='min-h-[22rem]'>{resultContent}</section>
      </div>
      <PlaygroundMediaHistory
        mode={props.mode}
        items={historyQuery.data ?? []}
        isLoading={historyQuery.isLoading}
        isRefreshing={historyQuery.isFetching}
        isError={historyQuery.isError}
        onRefresh={() => void refetchHistory()}
      />
    </div>
  )
}

function MediaEmpty(props: { icon: typeof ImageIcon; label: string }) {
  const Icon = props.icon
  return (
    <div className='text-muted-foreground flex min-h-[22rem] flex-col items-center justify-center gap-3 rounded-lg border border-dashed px-6 text-center'>
      <Icon className='size-9' />
      <p className='text-sm'>{props.label}</p>
    </div>
  )
}
