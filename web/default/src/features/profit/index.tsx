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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw, RotateCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  backfillProfitRecords,
  getModelCostRules,
  getProfitCostModelGroups,
  getProfitOverview,
  resetProfitAnalysisData,
  saveModelCostRule,
} from './api'
import { CostRules } from './components/cost-rules'
import { ProfitBreakdown } from './components/profit-breakdown'
import { ProfitSummary } from './components/profit-summary'
import type { ProfitQuery, SaveCostRuleInput } from './types'

type RangePreset = 'all' | '7d' | '30d' | '90d'

function rangeStart(preset: RangePreset): number | undefined {
  if (preset === 'all') return undefined
  const days = Number.parseInt(preset, 10)
  return Math.floor(Date.now() / 1000) - days * 24 * 60 * 60
}

export function Profit() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [range, setRange] = useState<RangePreset>('30d')
  const [modelFilter, setModelFilter] = useState('')
  const [userFilter, setUserFilter] = useState('')
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false)
  const query = useMemo<ProfitQuery>(() => {
    const userId = Number(userFilter)
    return {
      start_timestamp: rangeStart(range),
      model: modelFilter.trim() || undefined,
      user_id: Number.isInteger(userId) && userId > 0 ? userId : undefined,
    }
  }, [modelFilter, range, userFilter])

  const overviewQuery = useQuery({
    queryKey: ['profit-overview', query],
    queryFn: () => getProfitOverview(query),
  })
  const rulesQuery = useQuery({
    queryKey: ['profit-cost-rules'],
    queryFn: getModelCostRules,
  })
  const costModelsQuery = useQuery({
    queryKey: ['profit-cost-model-groups'],
    queryFn: getProfitCostModelGroups,
  })
  const saveMutation = useMutation({
    mutationFn: saveModelCostRule,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['profit-cost-rules'] }),
        queryClient.invalidateQueries({ queryKey: ['profit-overview'] }),
        queryClient.invalidateQueries({ queryKey: ['users'] }),
      ])
      toast.success(t('Cost rule saved'))
    },
    onError: () => toast.error(t('Failed to save cost rule')),
  })
  const backfillMutation = useMutation({
    mutationFn: backfillProfitRecords,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['profit-overview'] })
      toast.success(t('Profit records recalculated'))
    },
    onError: () => toast.error(t('Failed to recalculate profit records')),
  })
  const resetMutation = useMutation({
    mutationFn: resetProfitAnalysisData,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['profit-overview'] }),
        queryClient.invalidateQueries({ queryKey: ['profit-cost-model-groups'] }),
        queryClient.invalidateQueries({ queryKey: ['users'] }),
      ])
      setResetConfirmOpen(false)
      toast.success(t('Profit analysis data reset'))
    },
    onError: () => toast.error(t('Failed to reset profit analysis data')),
  })

  const saveRule = async (input: SaveCostRuleInput) => {
    await saveMutation.mutateAsync(input)
  }

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>
          {t('Profit Analysis')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button
            variant='outline'
            size='sm'
            onClick={() => backfillMutation.mutate()}
            disabled={backfillMutation.isPending || resetMutation.isPending}
          >
            <RefreshCw className='mr-2 h-4 w-4' />
            {t('Backfill missing costs')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            onClick={() => setResetConfirmOpen(true)}
            disabled={backfillMutation.isPending || resetMutation.isPending}
          >
            <RotateCcw className='mr-2 h-4 w-4' />
            {t('Reset analysis data')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-0 flex-col gap-4 overflow-auto pb-4'>
            <div className='border-border flex flex-wrap items-center gap-2 border px-3 py-2'>
              <NativeSelect
                value={range}
                onChange={(event) =>
                  setRange(event.target.value as RangePreset)
                }
                aria-label={t('Date range')}
              >
                <NativeSelectOption value='7d'>
                  {t('Last 7 days')}
                </NativeSelectOption>
                <NativeSelectOption value='30d'>
                  {t('Last 30 days')}
                </NativeSelectOption>
                <NativeSelectOption value='90d'>
                  {t('Last 90 days')}
                </NativeSelectOption>
                <NativeSelectOption value='all'>
                  {t('All time')}
                </NativeSelectOption>
              </NativeSelect>
              <Input
                className='w-36'
                inputMode='numeric'
                value={userFilter}
                onChange={(event) => setUserFilter(event.target.value)}
                placeholder={t('User ID')}
              />
              <Input
                className='min-w-52 flex-1'
                value={modelFilter}
                onChange={(event) => setModelFilter(event.target.value)}
                placeholder={t('Filter by model')}
              />
            </div>

            <Tabs defaultValue='overview' className='space-y-4'>
              <TabsList>
                <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
                <TabsTrigger value='costs'>
                  {t('Cost Configuration')}
                </TabsTrigger>
              </TabsList>
              <TabsContent value='overview' className='space-y-4'>
                {overviewQuery.isError ? (
                  <div className='border-border text-rose-600 border px-3 py-2 text-sm dark:text-rose-400'>
                    {t('Failed to load profit overview')}
                  </div>
                ) : null}
                <ProfitSummary summary={overviewQuery.data?.summary} />
                <ProfitBreakdown overview={overviewQuery.data} />
              </TabsContent>
              <TabsContent value='costs'>
                <CostRules
                  rules={rulesQuery.data ?? []}
                  modelGroups={costModelsQuery.data ?? []}
                  isSaving={saveMutation.isPending}
                  onSave={saveRule}
                />
              </TabsContent>
            </Tabs>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ConfirmDialog
        open={resetConfirmOpen}
        onOpenChange={setResetConfirmOpen}
        title={t('Reset profit analysis data?')}
        desc={t(
          'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.'
        )}
        destructive
        isLoading={resetMutation.isPending}
        handleConfirm={() => resetMutation.mutate()}
        confirmText={t('Reset')}
      />
    </>
  )
}
