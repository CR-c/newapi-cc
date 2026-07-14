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
  FileAudio,
  FileVideo,
  ImageIcon,
  Link,
  Plus,
  Upload,
  X,
} from 'lucide-react'
import { useEffect, useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import type { PlaygroundAssetKind } from '../../types'
import type { VideoModelProfile } from '../../video-model-profiles'

export interface VideoReferenceFiles {
  images: File[]
  videos: File[]
  audios: File[]
}

export interface VideoReferenceURLs {
  images: string[]
  videos: string[]
  audios: string[]
}

interface VideoReferenceInputsProps {
  profile: VideoModelProfile
  files: VideoReferenceFiles
  urls: VideoReferenceURLs
  disabled: boolean
  onChange: (files: VideoReferenceFiles) => void
  onURLsChange: (urls: VideoReferenceURLs) => void
}

interface ReferenceFileInputProps {
  kind: PlaygroundAssetKind
  files: File[]
  urls: string[]
  maxFiles: number
  accept: string
  disabled: boolean
  onChange: (files: File[]) => void
  onURLsChange: (urls: string[]) => void
  allowImageDataURL: boolean
}

const kindLabels: Record<PlaygroundAssetKind, string> = {
  image: 'Reference images',
  video: 'Reference videos',
  audio: 'Reference audio',
}

const kindActions: Record<PlaygroundAssetKind, string> = {
  image: 'Add images',
  video: 'Add videos',
  audio: 'Add audio',
}

function ReferenceFileInput(props: ReferenceFileInputProps) {
  const { t } = useTranslation()
  const inputId = `${useId()}-${props.kind}`
  const urlInputId = `${inputId}-url`
  const [urlValue, setURLValue] = useState('')
  const previewURLs = useMemo(
    () =>
      props.kind === 'image'
        ? props.files.map((file) => URL.createObjectURL(file))
        : [],
    [props.files, props.kind]
  )

  useEffect(
    () => () => {
      previewURLs.forEach((url) => URL.revokeObjectURL(url))
    },
    [previewURLs]
  )

  const addFiles = (fileList: FileList | null) => {
    if (!fileList) return
    const remaining = props.maxFiles - props.files.length - props.urls.length
    if (remaining <= 0) {
      toast.error(t('Reference limit reached'))
      return
    }
    const knownFiles = new Set(
      props.files.map(
        (file) => `${file.name}-${file.size}-${file.lastModified}`
      )
    )
    const selected = [...fileList]
      .filter((file) => {
        const key = `${file.name}-${file.size}-${file.lastModified}`
        if (knownFiles.has(key)) {
          return false
        }
        knownFiles.add(key)
        return true
      })
      .slice(0, remaining)
    if (selected.length < fileList.length) {
      toast.error(t('Reference limit reached'))
    }
    props.onChange([...props.files, ...selected])
  }

  const addURL = () => {
    const value = urlValue.trim()
    const isImageDataURL =
      props.kind === 'image' &&
      props.allowImageDataURL &&
      /^data:image\/(?:jpeg|png|webp);base64,/i.test(value)
    let isHTTPURL = false
    try {
      const parsed = new URL(value)
      isHTTPURL = parsed.protocol === 'http:' || parsed.protocol === 'https:'
    } catch {
      isHTTPURL = false
    }
    if (!isImageDataURL && !isHTTPURL) {
      toast.error(t('Must be a valid URL'))
      return
    }
    if (
      props.files.length + props.urls.length >= props.maxFiles ||
      props.urls.includes(value)
    ) {
      toast.error(t('Reference limit reached'))
      return
    }
    props.onURLsChange([...props.urls, value])
    setURLValue('')
  }

  const referenceCount = props.files.length + props.urls.length

  let FileIcon = FileAudio
  if (props.kind === 'image') {
    FileIcon = ImageIcon
  } else if (props.kind === 'video') {
    FileIcon = FileVideo
  }

  return (
    <div className='space-y-2'>
      <div className='flex items-center justify-between gap-3'>
        <Label htmlFor={inputId}>
          {t(kindLabels[props.kind])} ({referenceCount}/{props.maxFiles})
        </Label>
        <Button
          variant='outline'
          size='sm'
          disabled={props.disabled || referenceCount >= props.maxFiles}
          render={<label htmlFor={inputId} />}
        >
          <Upload />
          {t(kindActions[props.kind])}
        </Button>
        <Input
          id={inputId}
          type='file'
          accept={props.accept}
          multiple={props.maxFiles > 1}
          className='sr-only'
          disabled={props.disabled}
          onChange={(event) => {
            addFiles(event.target.files)
            event.target.value = ''
          }}
        />
      </div>
      <div className='flex gap-2'>
        <Input
          id={urlInputId}
          type='text'
          value={urlValue}
          placeholder={t('URL')}
          disabled={props.disabled || referenceCount >= props.maxFiles}
          onChange={(event) => setURLValue(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              addURL()
            }
          }}
          aria-label={`${t(kindLabels[props.kind])} ${t('URL')}`}
        />
        <Button
          variant='outline'
          size='icon'
          disabled={
            props.disabled ||
            referenceCount >= props.maxFiles ||
            !urlValue.trim()
          }
          onClick={addURL}
          aria-label={t('Add')}
          title={t('Add')}
        >
          <Plus />
        </Button>
      </div>
      {props.urls.length > 0 && (
        <div className='space-y-2'>
          {props.urls.map((url) => (
            <div
              key={url}
              className='flex min-w-0 items-center gap-2 rounded-lg border p-2'
            >
              <div className='bg-muted flex size-9 shrink-0 items-center justify-center rounded-md'>
                <Link className='text-muted-foreground size-4' />
              </div>
              <p className='min-w-0 flex-1 truncate text-xs' title={url}>
                {url}
              </p>
              <Button
                variant='ghost'
                size='icon-sm'
                disabled={props.disabled}
                onClick={() =>
                  props.onURLsChange(props.urls.filter((item) => item !== url))
                }
                aria-label={t('Remove reference')}
                title={t('Remove reference')}
              >
                <X />
              </Button>
            </div>
          ))}
        </div>
      )}
      {props.files.length > 0 && (
        <div className='grid gap-2 sm:grid-cols-2'>
          {props.files.map((file, index) => {
            const key = `${file.name}-${file.size}-${file.lastModified}`
            return (
              <div
                key={key}
                className='flex min-w-0 items-center gap-2 rounded-lg border p-2'
              >
                {props.kind === 'image' ? (
                  <img
                    src={previewURLs[index]}
                    alt={file.name}
                    className='size-12 shrink-0 rounded-md border object-cover'
                  />
                ) : (
                  <div className='bg-muted flex size-12 shrink-0 items-center justify-center rounded-md'>
                    <FileIcon className='text-muted-foreground size-5' />
                  </div>
                )}
                <div className='min-w-0 flex-1'>
                  <p className='truncate text-xs font-medium'>{file.name}</p>
                  <p className='text-muted-foreground text-xs'>
                    {(file.size / 1024 / 1024).toFixed(1)} MB
                  </p>
                </div>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  disabled={props.disabled}
                  onClick={() =>
                    props.onChange(props.files.filter((item) => item !== file))
                  }
                  aria-label={t('Remove reference')}
                  title={t('Remove reference')}
                >
                  <X />
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function VideoReferenceInputs(props: VideoReferenceInputsProps) {
  return (
    <div className='space-y-4'>
      {props.profile.maxImages > 0 && (
        <ReferenceFileInput
          kind='image'
          files={props.files.images}
          urls={props.urls.images}
          maxFiles={props.profile.maxImages}
          accept='image/jpeg,image/png,image/webp'
          disabled={props.disabled}
          onChange={(images) => props.onChange({ ...props.files, images })}
          onURLsChange={(images) =>
            props.onURLsChange({ ...props.urls, images })
          }
          allowImageDataURL={props.profile.provider === 'grok'}
        />
      )}
      {props.profile.maxVideos > 0 && (
        <ReferenceFileInput
          kind='video'
          files={props.files.videos}
          urls={props.urls.videos}
          maxFiles={props.profile.maxVideos}
          accept='video/mp4,video/quicktime,video/webm'
          disabled={props.disabled}
          onChange={(videos) => props.onChange({ ...props.files, videos })}
          onURLsChange={(videos) =>
            props.onURLsChange({ ...props.urls, videos })
          }
          allowImageDataURL={false}
        />
      )}
      {props.profile.maxAudios > 0 && (
        <ReferenceFileInput
          kind='audio'
          files={props.files.audios}
          urls={props.urls.audios}
          maxFiles={props.profile.maxAudios}
          accept='audio/mpeg,audio/mp4,audio/wav,audio/aac,audio/ogg'
          disabled={props.disabled}
          onChange={(audios) => props.onChange({ ...props.files, audios })}
          onURLsChange={(audios) =>
            props.onURLsChange({ ...props.urls, audios })
          }
          allowImageDataURL={false}
        />
      )}
    </div>
  )
}
