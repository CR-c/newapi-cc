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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { getCurrencyDisplay } from '@/lib/currency'

const createSchema = (t: (key: string) => string) =>
  z.object({
    amount: z
      .number()
      .positive(t('Amount must be greater than 0'))
      .int(t('Amount must be a whole number')),
    bonus: z
      .number()
      .positive(t('Amount must be greater than 0'))
      .int(t('Amount must be a whole number')),
  })

export type AmountBonusData = z.infer<ReturnType<typeof createSchema>>

type AmountBonusDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (data: AmountBonusData) => void
  editData?: AmountBonusData | null
}

const FORM_ID = 'amount-bonus-form'

export function AmountBonusDialog(props: AmountBonusDialogProps) {
  const { t } = useTranslation()
  const schema = createSchema(t)
  const { meta } = getCurrencyDisplay()
  const unitLabel = meta.kind === 'tokens' ? t('Tokens') : 'USD'
  const form = useForm<AmountBonusData>({
    resolver: zodResolver(schema),
    defaultValues: { amount: 0, bonus: 0 },
  })

  useEffect(() => {
    form.reset(props.editData ?? { amount: 0, bonus: 0 })
  }, [form, props.editData, props.open])

  const handleSubmit = (values: AmountBonusData) => {
    props.onSave(values)
    props.onOpenChange(false)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={props.editData ? t('Edit gift tier') : t('Add gift tier')}
      description={t(
        'Set the gift quota credited for an exact recharge amount.'
      )}
      contentClassName='sm:max-w-[500px]'
      contentHeight='auto'
      footer={
        <>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button type='submit' form={FORM_ID}>
            {props.editData ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='amount'
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('Recharge Amount')} ({unitLabel})
                </FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    step={1}
                    disabled={!!props.editData}
                    {...field}
                    onChange={(event) =>
                      field.onChange(
                        Number.parseInt(event.target.value, 10) || 0
                      )
                    }
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'The gift applies only when this exact amount is recharged.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='bonus'
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('Gift')} ({unitLabel})
                </FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    step={1}
                    {...field}
                    onChange={(event) =>
                      field.onChange(
                        Number.parseInt(event.target.value, 10) || 0
                      )
                    }
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Gift balance is promotional credit and produces no revenue.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </Dialog>
  )
}
