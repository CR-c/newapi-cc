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
import {
  Clock3,
  Download,
  ImageIcon,
  Loader2,
  RefreshCw,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'

import type {
  ImageGenerationResult,
  PlaygroundMediaHistoryItem,
  PlaygroundMode,
} from '../../types'

interface PlaygroundMediaHistoryProps {
  mode: Exclude<PlaygroundMode, 'chat'>
  items: PlaygroundMediaHistoryItem[]
  isLoading: boolean
  isRefreshing: boolean
  isError: boolean
  onRefresh: () => void
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

function historyStatusLabel(
  status: string,
  t: (key: string) => string
): string {
  const normalized = status.toLowerCase()
  if (normalized === 'completed') return t('Completed')
  if (normalized === 'failed') return t('Failed')
  if (normalized === 'cancelled') return t('Cancelled')
  if (normalized === 'queued') return t('Queued')
  return t('Processing')
}

function HistoryTimestamp(props: { timestamp: number }) {
  const { t } = useTranslation()
  const value = new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(props.timestamp * 1000))

  return (
    <span className='text-muted-foreground inline-flex items-center gap-1 text-xs'>
      <Clock3 className='size-3.5' />
      <span className='sr-only'>{t('Created')}</span>
      {value}
    </span>
  )
}

function ImageHistoryItem(props: { item: PlaygroundMediaHistoryItem }) {
  const { t } = useTranslation()
  const images = props.item.images ?? []

  return (
    <article className='min-w-0 rounded-lg border p-3'>
      <div className='mb-3 flex min-w-0 items-start justify-between gap-3'>
        <div className='min-w-0'>
          <p className='truncate text-sm font-medium'>{props.item.model}</p>
          {props.item.prompt && (
            <p className='text-muted-foreground line-clamp-2 text-xs'>
              {props.item.prompt}
            </p>
          )}
        </div>
        <HistoryTimestamp timestamp={props.item.created_at} />
      </div>
      <div className='grid gap-3 sm:grid-cols-2'>
        {images.map((image, index) => {
          const src = imageSource(image)
          if (!src) return null
          const imageKey =
            image.url ||
            image.b64_json?.slice(0, 128) ||
            image.revised_prompt ||
            src
          return (
            <figure
              key={`${props.item.id}-${imageKey}`}
              className='min-w-0 space-y-2'
            >
              <div className='bg-muted/30 aspect-square overflow-hidden rounded-lg border'>
                <img
                  src={src}
                  alt={
                    image.revised_prompt ||
                    props.item.prompt ||
                    t('Generated image')
                  }
                  className='size-full object-contain'
                  loading='lazy'
                />
              </div>
              <Button
                variant='outline'
                size='sm'
                render={
                  <a
                    href={src}
                    download={`${props.item.id}-${index + 1}.png`}
                  />
                }
              >
                <Download />
                {t('Download')}
              </Button>
            </figure>
          )
        })}
      </div>
    </article>
  )
}

function VideoHistoryItem(props: { item: PlaygroundMediaHistoryItem }) {
  const { t } = useTranslation()
  const normalizedStatus = props.item.status.toLowerCase()
  const isPending = !['completed', 'failed', 'cancelled'].includes(
    normalizedStatus
  )

  return (
    <article className='min-w-0 rounded-lg border p-3'>
      <div className='flex min-w-0 items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-2'>
            <p className='truncate text-sm font-medium'>{props.item.model}</p>
            <Badge variant='secondary'>
              {historyStatusLabel(props.item.status, t)}
            </Badge>
          </div>
          <p className='text-muted-foreground mt-1 truncate font-mono text-xs'>
            {props.item.task_id}
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          <HistoryTimestamp timestamp={props.item.created_at} />
          {isPending && (
            <Loader2 className='text-muted-foreground size-4 animate-spin' />
          )}
        </div>
      </div>
      {isPending && <Progress className='mt-3' value={props.item.progress} />}
      {normalizedStatus === 'completed' && props.item.result_url && (
        <div className='mt-3 space-y-2'>
          <video
            className='aspect-video w-full rounded-lg border bg-black'
            src={props.item.result_url}
            controls
            preload='none'
          />
          <Button
            variant='outline'
            size='sm'
            render={
              <a
                href={props.item.result_url}
                download={`${props.item.task_id || 'video'}.mp4`}
              />
            }
          >
            <Download />
            {t('Download video')}
          </Button>
        </div>
      )}
      {normalizedStatus === 'failed' && (
        <p className='text-destructive mt-3 text-sm'>
          {props.item.error || t('Video generation failed')}
        </p>
      )}
    </article>
  )
}

export function PlaygroundMediaHistory(props: PlaygroundMediaHistoryProps) {
  const { t } = useTranslation()
  const HistoryIcon = props.mode === 'image' ? ImageIcon : Video
  let historyContent = (
    <div className='grid gap-3 md:grid-cols-2'>
      <Skeleton className='h-44 w-full' />
      <Skeleton className='h-44 w-full' />
    </div>
  )

  if (props.isError) {
    historyContent = (
      <div className='text-destructive flex min-h-28 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm'>
        {t('Failed to load media history')}
      </div>
    )
  } else if (!props.isLoading && props.items.length === 0) {
    historyContent = (
      <div className='text-muted-foreground flex min-h-28 items-center justify-center rounded-lg border border-dashed px-4 text-center text-sm'>
        {t('No media generated in the last 24 hours')}
      </div>
    )
  } else if (!props.isLoading) {
    historyContent = (
      <div className='grid min-w-0 gap-3 md:grid-cols-2'>
        {props.items.map((item) =>
          props.mode === 'image' ? (
            <ImageHistoryItem key={item.id} item={item} />
          ) : (
            <VideoHistoryItem key={item.id} item={item} />
          )
        )}
      </div>
    )
  }

  return (
    <section
      className='border-t pt-5'
      aria-labelledby={`${props.mode}-history-title`}
    >
      <div className='mb-3 flex items-center justify-between gap-3'>
        <h2
          id={`${props.mode}-history-title`}
          className='flex items-center gap-2 text-sm font-semibold'
        >
          <HistoryIcon className='size-4' />
          {t('Recent 24 hours')}
        </h2>
        <Button
          variant='ghost'
          size='icon-sm'
          onClick={props.onRefresh}
          disabled={props.isRefreshing}
          aria-label={t('Refresh history')}
          title={t('Refresh history')}
        >
          <RefreshCw className={props.isRefreshing ? 'animate-spin' : ''} />
        </Button>
      </div>

      {historyContent}
    </section>
  )
}
