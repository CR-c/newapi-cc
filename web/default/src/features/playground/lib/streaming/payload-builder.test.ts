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
import { describe, expect, test } from 'bun:test'

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../../constants'
import type { Message, ParameterEnabled, PlaygroundConfig } from '../../types'
import { buildChatCompletionPayload } from './payload-builder'

const sampleMessages: Message[] = [
  {
    key: 'user-1',
    from: 'user',
    versions: [{ id: 'v1', content: 'hello' }],
    status: 'complete',
  },
]

describe('buildChatCompletionPayload', () => {
  test('only includes enabled sampling parameters', () => {
    const config: PlaygroundConfig = {
      ...DEFAULT_CONFIG,
      temperature: 0.4,
      top_p: 0.8,
      frequency_penalty: 0.2,
      presence_penalty: 0.1,
      max_tokens: 1024,
      max_completion_tokens: 2048,
      seed: 42,
      reasoning_effort: 'high',
    }
    const parameterEnabled: ParameterEnabled = {
      ...DEFAULT_PARAMETER_ENABLED,
      temperature: true,
      top_p: false,
      frequency_penalty: false,
      presence_penalty: false,
      max_tokens: true,
      max_completion_tokens: true,
      seed: true,
      reasoning_effort: true,
    }

    const payload = buildChatCompletionPayload(
      sampleMessages,
      config,
      parameterEnabled
    )

    expect(payload).toMatchObject({
      model: config.model,
      group: config.group,
      stream: true,
      temperature: 0.4,
      max_tokens: 1024,
      max_completion_tokens: 2048,
      seed: 42,
      reasoning_effort: 'high',
    })
    expect(payload).not.toHaveProperty('top_p')
    expect(payload).not.toHaveProperty('frequency_penalty')
    expect(payload).not.toHaveProperty('presence_penalty')
    expect(payload.messages).toEqual([{ role: 'user', content: 'hello' }])
  })

  test('omits seed when enabled but null', () => {
    const payload = buildChatCompletionPayload(
      sampleMessages,
      { ...DEFAULT_CONFIG, seed: null },
      { ...DEFAULT_PARAMETER_ENABLED, seed: true }
    )

    expect(payload).not.toHaveProperty('seed')
  })
})
