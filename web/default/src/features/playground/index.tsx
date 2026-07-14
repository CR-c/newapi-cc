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
import { ImageIcon, MessageSquare, Video } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { PlaygroundChat } from './components/chat/playground-chat'
import { PlaygroundInput } from './components/input/playground-input'
import { PlaygroundMedia } from './components/media/playground-media'
import { PLAYGROUND_MODES } from './constants'
import {
  useChatHandler,
  usePlaygroundConversation,
  usePlaygroundOptions,
  usePlaygroundState,
} from './hooks'

export function Playground() {
  const { t } = useTranslation()
  const [mode, setMode] = useState<'chat' | 'image' | 'video'>('chat')
  const {
    config,
    parameterEnabled,
    messages,
    isLoadingMessages,
    models,
    groups,
    updateMessages,
    setModels,
    setGroups,
    updateConfig,
    clearMessages,
  } = usePlaygroundState()

  const { sendChat, stopGeneration, isGenerating } = useChatHandler({
    config,
    parameterEnabled,
    onMessageUpdate: updateMessages,
  })

  const {
    editingMessageKey,
    handleSendMessage,
    handleRegenerateMessage,
    handleEditMessage,
    handleEditOpenChange,
    applyEdit,
    handleDeleteMessage,
  } = usePlaygroundConversation({
    messages,
    updateMessages,
    sendChat,
  })

  const handleClearMessages = () => {
    handleEditOpenChange(false)
    clearMessages()
  }

  const { isLoadingModels } = usePlaygroundOptions({
    currentGroup: config.group,
    currentModel: config.model,
    setGroups,
    setModels,
    updateConfig,
    mode: PLAYGROUND_MODES.CHAT,
  })

  return (
    <Tabs value={mode} onValueChange={(value) => setMode(value as typeof mode)} className='relative flex size-full min-h-0 flex-col overflow-hidden'>
      <div className='border-b px-4 py-2 sm:px-6'>
        <TabsList>
          <TabsTrigger value={PLAYGROUND_MODES.CHAT}><MessageSquare />{t('Chat')}</TabsTrigger>
          <TabsTrigger value={PLAYGROUND_MODES.IMAGE}><ImageIcon />{t('Image')}</TabsTrigger>
          <TabsTrigger value={PLAYGROUND_MODES.VIDEO}><Video />{t('Video')}</TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value={PLAYGROUND_MODES.CHAT} className='flex min-h-0 flex-1 flex-col overflow-hidden'>
        <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
          <PlaygroundChat
            messages={messages}
            isLoadingMessages={isLoadingMessages}
            onRegenerateMessage={handleRegenerateMessage}
            onEditMessage={handleEditMessage}
            onDeleteMessage={handleDeleteMessage}
            onSelectPrompt={handleSendMessage}
            isGenerating={isGenerating}
            editingKey={editingMessageKey}
            onCancelEdit={handleEditOpenChange}
            onSaveEdit={(newContent) => applyEdit(newContent, false)}
            onSaveEditAndSubmit={(newContent) => applyEdit(newContent, true)}
          />
        </div>
        <div className='mx-auto w-full max-w-4xl'>
          <PlaygroundInput
            disabled={isGenerating}
            groups={groups}
            groupValue={config.group}
            isGenerating={isGenerating}
            isModelLoading={isLoadingModels}
            modelValue={config.model}
            models={models}
            onGroupChange={(value) => updateConfig('group', value)}
            onClearMessages={handleClearMessages}
            onModelChange={(value) => updateConfig('model', value)}
            onStop={stopGeneration}
            onSubmit={handleSendMessage}
            hasMessages={messages.length > 0}
          />
        </div>
      </TabsContent>
      <TabsContent value={PLAYGROUND_MODES.IMAGE} className='min-h-0 overflow-hidden'>
        <PlaygroundMedia mode='image' groups={groups} setGroups={setGroups} />
      </TabsContent>
      <TabsContent value={PLAYGROUND_MODES.VIDEO} className='min-h-0 overflow-hidden'>
        <PlaygroundMedia mode='video' groups={groups} setGroups={setGroups} />
      </TabsContent>
    </Tabs>
  )
}
