import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatSession, Message } from '@/types'

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const currentSessionId = ref<string | null>(null)
  const messages = ref<Message[]>([])
  
  function setSessions(data: ChatSession[]) {
    sessions.value = data
  }
  
  function setCurrentSession(id: string) {
    currentSessionId.value = id
    messages.value = []
  }
  
  function addMessage(msg: Message) {
    messages.value.push(msg)
    const session = sessions.value.find(s => s.id === currentSessionId.value)
    if (session) {
      session.lastMessage = msg.content
      session.updatedAt = msg.createdAt
    }
  }
  
  function setMessages(data: Message[]) {
    messages.value = data
  }
  
  function clearUnread(sessionId: string) {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.unreadCount = 0
    }
  }
  
  return {
    sessions,
    currentSessionId,
    messages,
    setSessions,
    setCurrentSession,
    addMessage,
    setMessages,
    clearUnread
  }
})
