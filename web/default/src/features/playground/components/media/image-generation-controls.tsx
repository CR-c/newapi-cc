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

import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import type { ImageModelProfile } from '../../image-model-profiles'
import { ImageReferenceInputs } from './video-reference-inputs'

export interface ImageGenerationSettings {
  aspectRatio: string
  resolution: string
  size: string
  quality: string
  count: number
  responseFormat: 'url' | 'b64_json'
  background: 'opaque' | 'transparent'
  useExactSize: boolean
  exactWidth: string
  exactHeight: string
  referenceFiles: File[]
  referenceURLs: string[]
}

interface ImageGenerationControlsProps {
  profile: ImageModelProfile
  settings: ImageGenerationSettings
  exactSizeValid: boolean
  disabled: boolean
  onChange: (next: ImageGenerationSettings) => void
}

export function ImageGenerationControls(props: ImageGenerationControlsProps) {
  const { t } = useTranslation()
  const update = <K extends keyof ImageGenerationSettings>(
    key: K,
    value: ImageGenerationSettings[K]
  ) => props.onChange({ ...props.settings, [key]: value })

  if (props.profile.provider === 'generic') {
    return (
      <div className='grid grid-cols-2 gap-3'>
        <div className='space-y-2'>
          <Label>{t('Size')}</Label>
          <Select
            value={props.settings.size}
            onValueChange={(value) => value && update('size', value)}
          >
            <SelectTrigger>
              <SelectValue>{props.settings.size}</SelectValue>
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
            value={props.settings.quality}
            onValueChange={(value) => value && update('quality', value)}
          >
            <SelectTrigger>
              <SelectValue>{t(props.settings.quality)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {props.profile.qualities.map((value) => (
                <SelectItem key={value} value={value}>
                  {t(value)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3'>
        <div className='space-y-2'>
          <Label>{t('Aspect ratio')}</Label>
          <Select
            value={props.settings.aspectRatio}
            disabled={props.settings.resolution === 'auto'}
            onValueChange={(value) => value && update('aspectRatio', value)}
          >
            <SelectTrigger>
              <SelectValue>{props.settings.aspectRatio}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {props.profile.aspectRatios.map((value) => (
                <SelectItem key={value} value={value}>
                  {value}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-2'>
          <Label>{t('Resolution')}</Label>
          {props.profile.fixedResolution ? (
            <Input value={props.profile.fixedResolution} disabled />
          ) : (
            <Select
              value={props.settings.resolution}
              onValueChange={(value) => value && update('resolution', value)}
            >
              <SelectTrigger>
                <SelectValue>{props.settings.resolution}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {props.profile.supportsAutoSize && (
                  <SelectItem value='auto'>{t('Auto')}</SelectItem>
                )}
                {props.profile.resolutions.map((value) => (
                  <SelectItem key={value} value={value}>
                    {value}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
      </div>

      {props.profile.supportsExactSize && (
        <div className='space-y-3'>
          <div className='flex items-center justify-between gap-3 border-y py-3'>
            <Label htmlFor='image-exact-size'>{t('Exact size')}</Label>
            <Switch
              id='image-exact-size'
              checked={props.settings.useExactSize}
              onCheckedChange={(checked) => update('useExactSize', checked)}
            />
          </div>
          {props.settings.useExactSize && (
            <div className='grid grid-cols-2 gap-3'>
              <div className='space-y-2'>
                <Label htmlFor='image-exact-width'>{t('Width')}</Label>
                <Input
                  id='image-exact-width'
                  type='number'
                  min={16}
                  step={16}
                  value={props.settings.exactWidth}
                  onChange={(event) => update('exactWidth', event.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='image-exact-height'>{t('Height')}</Label>
                <Input
                  id='image-exact-height'
                  type='number'
                  min={16}
                  step={16}
                  value={props.settings.exactHeight}
                  onChange={(event) =>
                    update('exactHeight', event.target.value)
                  }
                />
              </div>
              {!props.exactSizeValid && (
                <p className='text-destructive col-span-2 text-xs'>
                  {t('Width and height must be multiples of 16')}
                </p>
              )}
            </div>
          )}
        </div>
      )}

      <div className='grid grid-cols-2 gap-3'>
        {props.profile.qualities.length > 0 && (
          <div className='space-y-2'>
            <Label>{t('Quality')}</Label>
            <Select
              value={props.settings.quality}
              onValueChange={(value) => value && update('quality', value)}
            >
              <SelectTrigger>
                <SelectValue>{t(props.settings.quality)}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {props.profile.qualities.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(value)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {props.profile.backgrounds.length > 0 && (
          <div className='space-y-2'>
            <Label>{t('Background')}</Label>
            <Select
              value={props.settings.background}
              onValueChange={(value) =>
                value && update('background', value as 'opaque' | 'transparent')
              }
            >
              <SelectTrigger>
                <SelectValue>{t(props.settings.background)}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {props.profile.backgrounds.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(value)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div className='space-y-2'>
          <Label>{t('Image count')}</Label>
          <Select
            value={String(props.settings.count)}
            onValueChange={(value) => value && update('count', Number(value))}
          >
            <SelectTrigger>
              <SelectValue>{props.settings.count}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Array.from({ length: props.profile.maxImages }, (_, index) => {
                const value = index + 1
                return (
                  <SelectItem key={value} value={String(value)}>
                    {value}
                  </SelectItem>
                )
              })}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-2'>
          <Label>{t('Response format')}</Label>
          <Select
            value={props.settings.responseFormat}
            onValueChange={(value) =>
              value && update('responseFormat', value as 'url' | 'b64_json')
            }
          >
            <SelectTrigger>
              <SelectValue>
                {props.settings.responseFormat === 'url' ? 'URL' : 'Base64'}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {props.profile.responseFormats.map((value) => (
                <SelectItem key={value} value={value}>
                  {value === 'url' ? 'URL' : 'Base64'}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <ImageReferenceInputs
        files={props.settings.referenceFiles}
        urls={props.settings.referenceURLs}
        maxFiles={props.profile.maxReferences}
        disabled={props.disabled}
        onChange={(files) => update('referenceFiles', files)}
        onURLsChange={(urls) => update('referenceURLs', urls)}
      />
    </div>
  )
}
