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
import { Plus, Trash2 } from 'lucide-react'
import { Fragment, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import type {
  ModelCostTier,
  ModelCostRule,
  ProfitCostModelGroup,
  SaveCostRuleInput,
} from '../types'

type CostTierDraft = ModelCostTier & { draftId: string }

export function CostRules(props: {
  rules: ModelCostRule[]
  modelGroups: ProfitCostModelGroup[]
  isSaving: boolean
  onSave: (input: SaveCostRuleInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const [modelName, setModelName] = useState('')
  const [purchasePrice, setPurchasePrice] = useState('')
  const [costTiers, setCostTiers] = useState<CostTierDraft[]>([])
  const activeRules = useMemo(
    () => props.rules.filter((rule) => rule.enabled),
    [props.rules]
  )
  const activeRuleByModel = useMemo(() => {
    const rules = new Map<string, ModelCostRule>()
    for (const rule of activeRules) {
      if (!rules.has(rule.model_name)) {
        rules.set(rule.model_name, rule)
      }
    }
    return rules
  }, [activeRules])
  const groupsByModel = useMemo(() => {
    const groups = new Map<string, string[]>()
    for (const group of props.modelGroups) {
      const groupName = group.group || t('Other')
      for (const model of group.models) {
        groups.set(model, [...(groups.get(model) ?? []), groupName])
      }
    }
    return groups
  }, [props.modelGroups, t])
  const modelOptions = useMemo(
    () =>
      [...groupsByModel.entries()]
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([name, groups]) => ({
          value: name,
          label: `${name} (${groups.join(', ')})`,
        })),
    [groupsByModel]
  )
  const parsedPurchasePrice = Number(purchasePrice)
  const canSave =
    modelName.trim() !== '' &&
    purchasePrice.trim() !== '' &&
    Number.isFinite(parsedPurchasePrice) &&
    parsedPurchasePrice > 0 &&
    costTiers.every(
      (tier) =>
        tier.key.trim() !== '' &&
        tier.label.trim() !== '' &&
        Number.isFinite(tier.purchase_price_cny) &&
        tier.purchase_price_cny > 0
    ) &&
    new Set(costTiers.map((tier) => tier.key.trim())).size === costTiers.length

  const selectModel = (value: string) => {
    setModelName(value)
    const activeRule = activeRuleByModel.get(value)
    setPurchasePrice(activeRule ? String(activeRule.purchase_price_cny) : '0')
    setCostTiers(
      (activeRule?.cost_tiers ?? []).map((tier) => ({
        ...tier,
        draftId: crypto.randomUUID(),
      }))
    )
  }

  const updateCostTier = <K extends keyof ModelCostTier>(
    index: number,
    field: K,
    value: ModelCostTier[K]
  ) => {
    setCostTiers((current) =>
      current.map((tier, tierIndex) =>
        tierIndex === index ? { ...tier, [field]: value } : tier
      )
    )
  }

  const submit = async () => {
    const input = {
      model_name: modelName.trim(),
      purchase_price_cny: parsedPurchasePrice,
      cost_tiers: costTiers.map((tier) => ({
        key: tier.key.trim(),
        label: tier.label.trim(),
        purchase_price_cny: tier.purchase_price_cny,
      })),
    }
    if (!canSave) return
    await props.onSave(input)
    setModelName('')
    setPurchasePrice('')
    setCostTiers([])
  }

  return (
    <div className='space-y-4'>
      <div className='border-border grid gap-3 border p-4 md:grid-cols-[minmax(220px,1fr)_minmax(160px,0.5fr)_auto] md:items-end'>
        <div className='space-y-1.5'>
          <Label htmlFor='cost-model'>{t('Model')}</Label>
          <Combobox
            id='cost-model'
            options={modelOptions}
            value={modelName}
            onValueChange={(value) => selectModel(value ?? '')}
            placeholder={t('Select a model to edit pricing')}
            emptyText={t('No models available')}
            allowCustomValue={false}
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor='cost-price'>{t('Purchase cost (CNY)')}</Label>
          <Input
            id='cost-price'
            type='number'
            inputMode='decimal'
            min='0'
            step='any'
            value={purchasePrice}
            onChange={(event) => setPurchasePrice(event.target.value)}
            placeholder='38.08'
          />
        </div>
        <Button onClick={submit} disabled={props.isSaving || !canSave}>
          {props.isSaving ? t('Saving...') : t('Save cost rule')}
        </Button>
        {modelName ? (
          <div className='border-border space-y-3 border-t pt-3 md:col-span-3'>
            <div className='flex items-center justify-between gap-3'>
              <Label>{t('Purchase cost tiers')}</Label>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() =>
                  setCostTiers((current) => [
                    ...current,
                    {
                      key: '',
                      label: '',
                      purchase_price_cny: 0,
                      draftId: crypto.randomUUID(),
                    },
                  ])
                }
              >
                <Plus />
                {t('Add cost tier')}
              </Button>
            </div>
            {costTiers.length === 0 ? (
              <div className='text-muted-foreground py-2 text-sm'>
                {t('No cost tiers')}
              </div>
            ) : (
              <div className='space-y-2'>
                {costTiers.map((tier, index) => (
                  <div
                    key={tier.draftId}
                    className='grid gap-2 md:grid-cols-[minmax(220px,1fr)_minmax(180px,0.8fr)_minmax(140px,0.45fr)_auto]'
                  >
                    <Input
                      value={tier.key}
                      onChange={(event) =>
                        updateCostTier(index, 'key', event.target.value)
                      }
                      aria-label={t('Cost tier key')}
                      placeholder='video:1080p:reference'
                    />
                    <Input
                      value={tier.label}
                      onChange={(event) =>
                        updateCostTier(index, 'label', event.target.value)
                      }
                      aria-label={t('Cost tier label')}
                      placeholder='1080p reference'
                    />
                    <Input
                      type='number'
                      inputMode='decimal'
                      min='0'
                      step='any'
                      value={tier.purchase_price_cny || ''}
                      onChange={(event) =>
                        updateCostTier(
                          index,
                          'purchase_price_cny',
                          Number(event.target.value)
                        )
                      }
                      aria-label={t('Purchase cost (CNY)')}
                      placeholder='20.15'
                    />
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      onClick={() =>
                        setCostTiers((current) =>
                          current.filter((_, tierIndex) => tierIndex !== index)
                        )
                      }
                      aria-label={t('Delete cost tier')}
                      title={t('Delete cost tier')}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : null}
      </div>

      <div className='border-border border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Group')}</TableHead>
              <TableHead>{t('Model')}</TableHead>
              <TableHead className='text-right'>
                {t('Purchase cost (CNY)')}
              </TableHead>
              <TableHead>{t('Purchase cost tiers')}</TableHead>
              <TableHead className='text-right'>{t('Version')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {props.modelGroups.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={5}
                  className='text-muted-foreground py-8 text-center'
                >
                  {t('No models available')}
                </TableCell>
              </TableRow>
            ) : (
              props.modelGroups.map((group) => (
                <Fragment key={`group-${group.group || 'other'}`}>
                  <TableRow className='bg-muted/40 hover:bg-muted/40'>
                    <TableCell
                      colSpan={5}
                      className='text-muted-foreground text-xs font-medium tracking-wide uppercase'
                    >
                      {group.group || t('Other')}
                    </TableCell>
                  </TableRow>
                  {group.models.map((name) => {
                    const rule = activeRuleByModel.get(name)
                    return (
                      <TableRow
                        key={`${group.group || 'other'}-${name}`}
                        className='cursor-pointer'
                        onClick={() => selectModel(name)}
                      >
                        <TableCell className='text-muted-foreground'>
                          {group.group || t('Other')}
                        </TableCell>
                        <TableCell className='font-medium'>{name}</TableCell>
                        <TableCell className='text-right'>
                          ¥ {(rule?.purchase_price_cny ?? 0).toFixed(4)}
                        </TableCell>
                        <TableCell>
                          {(rule?.cost_tiers ?? []).length === 0 ? (
                            <span className='text-muted-foreground'>
                              {t('No cost tiers')}
                            </span>
                          ) : (
                            <div className='space-y-1'>
                              {(rule?.cost_tiers ?? []).map((tier) => (
                                <div
                                  key={tier.key}
                                  className='flex min-w-0 items-center justify-between gap-3 text-xs'
                                >
                                  <span className='truncate' title={tier.key}>
                                    {tier.label}
                                  </span>
                                  <span className='shrink-0 tabular-nums'>
                                    ¥ {tier.purchase_price_cny.toFixed(4)}
                                  </span>
                                </div>
                              ))}
                            </div>
                          )}
                        </TableCell>
                        <TableCell className='text-right'>
                          {rule ? `v${rule.version}` : t('Default')}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </Fragment>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
