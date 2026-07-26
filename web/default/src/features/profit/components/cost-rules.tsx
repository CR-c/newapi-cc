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
import { ChevronDown, Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import type {
  ModelCostTier,
  ModelCostRule,
  ProfitCostModelGroup,
  SaveCostRuleInput,
} from '../types'

type CostTierDraft = ModelCostTier & { draftId: string }

type RowDraft = {
  modelName: string
  groups: string[]
  purchasePrice: string
  costTiers: CostTierDraft[]
  expanded: boolean
  baselinePrice: number
  baselineTiers: ModelCostTier[]
  version?: number
  configured: boolean
}

function tiersEqual(
  left: Array<Pick<ModelCostTier, 'key' | 'label' | 'purchase_price_cny'>>,
  right: Array<Pick<ModelCostTier, 'key' | 'label' | 'purchase_price_cny'>>
): boolean {
  if (left.length !== right.length) return false
  return left.every((tier, index) => {
    const other = right[index]
    return (
      tier.key === other.key &&
      tier.label === other.label &&
      tier.purchase_price_cny === other.purchase_price_cny
    )
  })
}

function toTierPayload(tiers: CostTierDraft[]): ModelCostTier[] {
  return tiers.map((tier) => ({
    key: tier.key.trim(),
    label: tier.label.trim(),
    purchase_price_cny: tier.purchase_price_cny,
  }))
}

export function CostRules(props: {
  rules: ModelCostRule[]
  modelGroups: ProfitCostModelGroup[]
  isSaving: boolean
  onSave: (input: SaveCostRuleInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [missingOnly, setMissingOnly] = useState(false)
  const [drafts, setDrafts] = useState<RowDraft[]>([])
  const [savingModel, setSavingModel] = useState<string | null>(null)

  const activeRuleByModel = useMemo(() => {
    const rules = new Map<string, ModelCostRule>()
    for (const rule of props.rules) {
      if (!rule.enabled) continue
      if (!rules.has(rule.model_name)) {
        rules.set(rule.model_name, rule)
      }
    }
    return rules
  }, [props.rules])

  useEffect(() => {
    const groupsByModel = new Map<string, string[]>()
    for (const group of props.modelGroups) {
      const groupName = group.group || t('Other')
      for (const model of group.models) {
        groupsByModel.set(model, [
          ...(groupsByModel.get(model) ?? []),
          groupName,
        ])
      }
    }
    for (const rule of props.rules) {
      if (!rule.enabled) continue
      if (!groupsByModel.has(rule.model_name)) {
        groupsByModel.set(rule.model_name, [t('Other')])
      }
    }

    setDrafts((current) => {
      const expandedByModel = new Map(
        current.map((row) => [row.modelName, row.expanded])
      )
      const priceByModel = new Map(
        current.map((row) => [row.modelName, row.purchasePrice])
      )
      const tiersByModel = new Map(
        current.map((row) => [row.modelName, row.costTiers])
      )
      const dirtyModels = new Set(
        current
          .filter((row) => {
            const price = Number(row.purchasePrice)
            const dirtyPrice =
              row.purchasePrice.trim() !== '' &&
              Number.isFinite(price) &&
              price !== row.baselinePrice
            const dirtyTiers = !tiersEqual(
              toTierPayload(row.costTiers),
              row.baselineTiers
            )
            return dirtyPrice || dirtyTiers
          })
          .map((row) => row.modelName)
      )

      return [...groupsByModel.entries()]
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([modelName, groups]) => {
          const rule = activeRuleByModel.get(modelName)
          const baselineTiers = rule?.cost_tiers ?? []
          const keepLocal = dirtyModels.has(modelName)
          let purchasePrice = ''
          if (keepLocal) {
            purchasePrice =
              priceByModel.get(modelName) ??
              String(rule?.purchase_price_cny ?? '')
          } else if (rule) {
            purchasePrice = String(rule.purchase_price_cny)
          }
          let costTiers = baselineTiers.map((tier) => ({
            ...tier,
            draftId: crypto.randomUUID(),
          }))
          if (keepLocal) {
            costTiers =
              tiersByModel.get(modelName) ??
              baselineTiers.map((tier) => ({
                ...tier,
                draftId: crypto.randomUUID(),
              }))
          }
          return {
            modelName,
            groups: [...new Set(groups)],
            purchasePrice,
            costTiers,
            expanded: expandedByModel.get(modelName) ?? false,
            baselinePrice: rule?.purchase_price_cny ?? 0,
            baselineTiers,
            version: rule?.version,
            configured: Boolean(rule && rule.purchase_price_cny > 0),
          } satisfies RowDraft
        })
    })
  }, [activeRuleByModel, props.modelGroups, props.rules, t])

  const visibleRows = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return drafts.filter((row) => {
      if (missingOnly && row.configured) return false
      if (!keyword) return true
      return (
        row.modelName.toLowerCase().includes(keyword) ||
        row.groups.some((group) => group.toLowerCase().includes(keyword))
      )
    })
  }, [drafts, missingOnly, search])

  const missingCount = drafts.filter((row) => !row.configured).length

  const updateRow = (modelName: string, patch: Partial<RowDraft>) => {
    setDrafts((current) =>
      current.map((row) =>
        row.modelName === modelName ? { ...row, ...patch } : row
      )
    )
  }

  const isRowDirty = (row: RowDraft): boolean => {
    const price = Number(row.purchasePrice)
    if (
      row.purchasePrice.trim() === '' ||
      !Number.isFinite(price) ||
      price <= 0
    ) {
      return false
    }
    if (price !== row.baselinePrice) return true
    return !tiersEqual(toTierPayload(row.costTiers), row.baselineTiers)
  }

  const canSaveRow = (row: RowDraft): boolean => {
    const price = Number(row.purchasePrice)
    if (
      row.purchasePrice.trim() === '' ||
      !Number.isFinite(price) ||
      price <= 0
    ) {
      return false
    }
    const tiers = toTierPayload(row.costTiers)
    if (
      tiers.some(
        (tier) =>
          tier.key === '' ||
          tier.label === '' ||
          !Number.isFinite(tier.purchase_price_cny) ||
          tier.purchase_price_cny <= 0
      )
    ) {
      return false
    }
    if (new Set(tiers.map((tier) => tier.key)).size !== tiers.length) {
      return false
    }
    return isRowDirty(row)
  }

  const saveRow = async (row: RowDraft) => {
    if (!canSaveRow(row) || props.isSaving) return
    const price = Number(row.purchasePrice)
    setSavingModel(row.modelName)
    try {
      await props.onSave({
        model_name: row.modelName,
        purchase_price_cny: price,
        cost_tiers: toTierPayload(row.costTiers),
      })
    } finally {
      setSavingModel(null)
    }
  }

  return (
    <div className='space-y-3'>
      <div className='border-border flex flex-col gap-3 border px-3 py-3 sm:flex-row sm:items-center sm:justify-between'>
        <div className='space-y-1'>
          <h2 className='text-sm font-medium'>{t('Cost Configuration')}</h2>
          <p className='text-muted-foreground text-xs'>
            {t('Edit purchase cost inline, then save the row.')}
            {missingCount > 0 ? (
              <span className='text-amber-600 dark:text-amber-400'>
                {' '}
                · {t('{{count}} models missing cost', { count: missingCount })}
              </span>
            ) : null}
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <Input
            className='w-56'
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t('Search model or group')}
          />
          <Button
            type='button'
            size='sm'
            variant={missingOnly ? 'default' : 'outline'}
            onClick={() => setMissingOnly((value) => !value)}
          >
            {t('Missing only')}
          </Button>
        </div>
      </div>

      <div className='border-border border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className='w-10' />
              <TableHead>{t('Model')}</TableHead>
              <TableHead>{t('Group')}</TableHead>
              <TableHead className='w-44'>{t('Purchase cost (CNY)')}</TableHead>
              <TableHead className='w-28'>{t('Status')}</TableHead>
              <TableHead className='w-28 text-right'>{t('Action')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {visibleRows.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className='text-muted-foreground py-8 text-center'
                >
                  {t('No models available')}
                </TableCell>
              </TableRow>
            ) : (
              visibleRows.map((row) => {
                const dirty = isRowDirty(row)
                const saving = savingModel === row.modelName
                return (
                  <TableRow
                    key={row.modelName}
                    className={cn(!row.configured && 'bg-amber-500/5')}
                  >
                    <TableCell className='align-top'>
                      <Button
                        type='button'
                        size='icon'
                        variant='ghost'
                        className='h-8 w-8'
                        onClick={() =>
                          updateRow(row.modelName, {
                            expanded: !row.expanded,
                          })
                        }
                        aria-label={t('Purchase cost tiers')}
                      >
                        <ChevronDown
                          className={cn(
                            'h-4 w-4 transition-transform',
                            row.expanded && 'rotate-180'
                          )}
                        />
                      </Button>
                    </TableCell>
                    <TableCell className='align-top'>
                      <div className='space-y-1'>
                        <div className='font-medium'>{row.modelName}</div>
                        {row.version ? (
                          <div className='text-muted-foreground text-xs'>
                            v{row.version}
                          </div>
                        ) : null}
                        {row.expanded ? (
                          <div className='border-border mt-3 space-y-2 border-t pt-3'>
                            <div className='flex items-center justify-between gap-2'>
                              <span className='text-muted-foreground text-xs font-medium'>
                                {t('Purchase cost tiers')}
                              </span>
                              <Button
                                type='button'
                                size='sm'
                                variant='outline'
                                onClick={() =>
                                  updateRow(row.modelName, {
                                    costTiers: [
                                      ...row.costTiers,
                                      {
                                        key: '',
                                        label: '',
                                        purchase_price_cny: 0,
                                        draftId: crypto.randomUUID(),
                                      },
                                    ],
                                  })
                                }
                              >
                                <Plus className='h-3.5 w-3.5' />
                                {t('Add cost tier')}
                              </Button>
                            </div>
                            {row.costTiers.length === 0 ? (
                              <p className='text-muted-foreground text-xs'>
                                {t('No cost tiers')}
                              </p>
                            ) : (
                              row.costTiers.map((tier, index) => (
                                <div
                                  key={tier.draftId}
                                  className='grid gap-2 md:grid-cols-[1fr_1fr_120px_auto]'
                                >
                                  <Input
                                    value={tier.key}
                                    onChange={(event) => {
                                      const next = [...row.costTiers]
                                      next[index] = {
                                        ...tier,
                                        key: event.target.value,
                                      }
                                      updateRow(row.modelName, {
                                        costTiers: next,
                                      })
                                    }}
                                    placeholder='video:1080p:reference'
                                    aria-label={t('Cost tier key')}
                                  />
                                  <Input
                                    value={tier.label}
                                    onChange={(event) => {
                                      const next = [...row.costTiers]
                                      next[index] = {
                                        ...tier,
                                        label: event.target.value,
                                      }
                                      updateRow(row.modelName, {
                                        costTiers: next,
                                      })
                                    }}
                                    placeholder='1080p reference'
                                    aria-label={t('Cost tier label')}
                                  />
                                  <Input
                                    type='number'
                                    inputMode='decimal'
                                    min='0'
                                    step='any'
                                    value={tier.purchase_price_cny || ''}
                                    onChange={(event) => {
                                      const next = [...row.costTiers]
                                      next[index] = {
                                        ...tier,
                                        purchase_price_cny: Number(
                                          event.target.value
                                        ),
                                      }
                                      updateRow(row.modelName, {
                                        costTiers: next,
                                      })
                                    }}
                                    aria-label={t('Purchase cost (CNY)')}
                                  />
                                  <Button
                                    type='button'
                                    size='icon'
                                    variant='ghost'
                                    onClick={() =>
                                      updateRow(row.modelName, {
                                        costTiers: row.costTiers.filter(
                                          (_, tierIndex) => tierIndex !== index
                                        ),
                                      })
                                    }
                                    aria-label={t('Delete cost tier')}
                                  >
                                    <Trash2 className='h-4 w-4' />
                                  </Button>
                                </div>
                              ))
                            )}
                          </div>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell className='text-muted-foreground align-top text-xs'>
                      <div className='flex max-w-64 flex-wrap gap-1'>
                        {row.groups.map((group) => (
                          <Badge
                            key={`${row.modelName}-${group}`}
                            variant='outline'
                          >
                            {group}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className='align-top'>
                      <Input
                        type='number'
                        inputMode='decimal'
                        min='0'
                        step='any'
                        value={row.purchasePrice}
                        onChange={(event) =>
                          updateRow(row.modelName, {
                            purchasePrice: event.target.value,
                          })
                        }
                        placeholder='0.01'
                        aria-label={`${row.modelName} ${t('Purchase cost (CNY)')}`}
                      />
                    </TableCell>
                    <TableCell className='align-top'>
                      {row.configured ? (
                        <Badge variant='outline' className='text-emerald-600'>
                          {t('Configured')}
                        </Badge>
                      ) : (
                        <Badge variant='outline' className='text-amber-600'>
                          {t('Missing cost')}
                        </Badge>
                      )}
                      {dirty ? (
                        <div className='text-muted-foreground mt-1 text-xs'>
                          {t('Unsaved')}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell className='align-top text-right'>
                      <Button
                        type='button'
                        size='sm'
                        disabled={!canSaveRow(row) || props.isSaving || saving}
                        onClick={() => void saveRow(row)}
                      >
                        <Save className='h-3.5 w-3.5' />
                        {saving ? t('Saving...') : t('Save')}
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
