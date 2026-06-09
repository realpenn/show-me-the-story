<script>
  import { onMount, afterUpdate } from 'svelte';
  import { api } from '../lib/api.js';
  import { chatSessions, currentChatSession, addToast, showConfirm, taskRunning } from '../lib/stores.js';

  export let contextPage = 'config';

  let chatInput = '';
  let messagesContainer;
  let showSessionList = false;

  $: sessions = ($chatSessions?.sessions || []);
  $: msgs = ($currentChatSession?.messages || []);
  $: streamingText = $currentChatSession?.streaming_text || '';
  $: pendingTools = $currentChatSession?.pending_tool_calls || [];

  onMount(async () => {
    try {
      chatSessions.set(await api('GET', '/api/chat/sessions'));
      if (!$currentChatSession) {
        if (sessions.length > 0) {
          await selectSession(sessions[0].id);
        } else {
          await createSession();
        }
      }
    } catch (e) {}
  });

  afterUpdate(() => {
    if (messagesContainer) messagesContainer.scrollTop = messagesContainer.scrollHeight;
  });

  export async function sendMessageToChat(text) {
    if (!$currentChatSession) {
      await createSession();
    }
    chatInput = text;
    await sendMessage();
  }

  async function createSession() {
    try {
      const session = await api('POST', '/api/chat/sessions');
      chatSessions.set(await api('GET', '/api/chat/sessions'));
      await selectSession(session.id);
    } catch (e) { addToast(e.message, 'error'); }
  }

  async function selectSession(id) {
    try {
      const session = await api('GET', '/api/chat/sessions/' + id);
      currentChatSession.set(session);
      showSessionList = false;
    } catch (e) { addToast(e.message, 'error'); }
  }

  async function deleteSession(id, e) {
    e.stopPropagation();
    showConfirm('确认删除此会话？', async () => {
      try {
        await api('DELETE', '/api/chat/sessions/' + id);
        chatSessions.set(await api('GET', '/api/chat/sessions'));
        if ($currentChatSession?.id === id) {
          currentChatSession.set(null);
          const updated = (await api('GET', '/api/chat/sessions')).sessions || [];
          if (updated.length > 0) await selectSession(updated[0].id);
        }
      } catch (e) { addToast(e.message, 'error'); }
    });
  }

  async function sendMessage() {
    if ($taskRunning) return;
    if (!$currentChatSession) { addToast('请先选择会话', 'error'); return; }
    const msg = chatInput.trim();
    if (!msg) return;
    chatInput = '';

    currentChatSession.update(s => {
      if (!s) return s;
      const messages = [...(s.messages || []), { role: 'user', content: msg, timestamp: new Date().toISOString() }];
      return { ...s, messages, streaming_text: '', pending_tool_calls: [] };
    });

    try {
      await api('POST', '/api/chat/sessions/' + $currentChatSession.id + '/messages', { content: msg, context_page: contextPage });
      const session = await api('GET', '/api/chat/sessions/' + $currentChatSession.id);
      currentChatSession.set(session);
      chatSessions.set(await api('GET', '/api/chat/sessions'));
    } catch (e) { addToast(e.message, 'error'); }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); }
  }
</script>

<div class="flex flex-col h-full">
  <!-- Session bar -->
  <div class="border-b border-base-content/10 px-3 py-2 flex items-center gap-2 shrink-0">
    <button class="btn btn-ghost btn-xs" on:click={() => showSessionList = !showSessionList}>
      {showSessionList ? '收起' : '会话列表'}
    </button>
    <span class="text-sm text-base-content/50 truncate flex-1">
      {$currentChatSession?.title || '未选择会话'}
    </span>
    <button class="btn btn-primary btn-xs" on:click={createSession} disabled={$taskRunning}>新建</button>
  </div>

  {#if showSessionList}
    <div class="border-b border-base-content/10 max-h-[200px] overflow-y-auto bg-base-200 shrink-0">
      {#each sessions as s}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
          class="px-3 py-2 border-b border-base-content/5 cursor-pointer hover:bg-base-300 transition-colors flex items-center gap-2"
          class:bg-base-300={$currentChatSession?.id === s.id}
          on:click={() => selectSession(s.id)}
        >
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium truncate">{s.title}</div>
            <div class="text-xs text-base-content/40">{new Date(s.updated_at).toLocaleString('zh-CN')}</div>
          </div>
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <span class="text-error text-sm opacity-0 hover:opacity-100 cursor-pointer" on:click={(e) => deleteSession(s.id, e)}>x</span>
        </div>
      {/each}
      {#if sessions.length === 0}
        <div class="px-3 py-2 text-sm text-base-content/40">暂无会话</div>
      {/if}
    </div>
  {/if}

  <!-- Messages -->
  <div bind:this={messagesContainer} class="flex-1 overflow-y-auto p-3 space-y-2">
    {#if !$currentChatSession}
      <div class="text-center text-base-content/40 py-8 text-base">选择或创建一个会话开始对话</div>
    {:else}
      {#each msgs as m}
        {#if m.role === 'user'}
          <div class="chat chat-end">
            <div class="chat-bubble chat-bubble-primary text-sm whitespace-pre-wrap max-w-[85%]">{m.content}</div>
          </div>
        {:else if m.role === 'assistant'}
          {#if m.tool_calls?.length > 0}
            {#each m.tool_calls as tc}
              <div class="chat chat-start">
                <div class="chat-bubble bg-base-300 text-xs font-mono max-w-[85%]">
                  <div class="text-warning font-semibold mb-0.5">🔧 {tc.name}</div>
                  <div class="text-base-content/50 break-all">{typeof tc.arguments === 'string' ? tc.arguments : JSON.stringify(tc.arguments)}</div>
                </div>
              </div>
            {/each}
          {/if}
          {#if m.content}
            <div class="chat chat-start">
              <div class="chat-bubble bg-base-300 text-sm whitespace-pre-wrap max-w-[85%]">{m.content}</div>
            </div>
          {/if}
        {:else if m.role === 'tool'}
          <div class="chat chat-start">
            <div class="chat-bubble bg-base-300 text-xs font-mono max-w-[85%]">
              <div class="text-info font-semibold mb-0.5">📋 工具结果</div>
              <div class="text-base-content/50 break-all">{m.tool_result || ''}</div>
            </div>
          </div>
        {/if}
      {/each}

      {#each pendingTools as tc}
        <div class="chat chat-start">
          <div class="chat-bubble bg-base-300 text-xs font-mono max-w-[85%]">
            <div class="text-warning font-semibold mb-0.5">🔧 调用 {tc.name}...</div>
            <div class="text-warning animate-pulse">执行中...</div>
          </div>
        </div>
      {/each}

      {#if streamingText}
        <div class="chat chat-start">
          <div class="chat-bubble bg-base-300 text-sm whitespace-pre-wrap max-w-[85%]">{streamingText}</div>
        </div>
      {/if}
    {/if}
  </div>

  <!-- Input -->
  {#if $currentChatSession}
    <div class="border-t border-base-content/10 p-2 flex gap-2 shrink-0">
      <textarea
        class="textarea textarea-sm flex-1 min-h-[36px] max-h-[100px] resize-none text-base"
        bind:value={chatInput}
        placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
        on:keydown={handleKeydown}
      ></textarea>
      <button class="btn btn-primary btn-sm" on:click={sendMessage} disabled={$taskRunning}>发送</button>
    </div>
  {/if}
</div>
