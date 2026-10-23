// 会话 ID 的可读化：Claude Code 上报的 metadata.user_id 形如
// user_<hash>_account__session_<uuid>，真正有辨识度的只是末尾的会话 UUID；
// Codex/Gemini 直接是 UUID。统一截取可读段的前 8 位用于列表展示。
const claudeSessionMarker = '_session_'

export const shortSessionId = (sessionId: string): string => {
  if (!sessionId) return ''
  const markerIndex = sessionId.lastIndexOf(claudeSessionMarker)
  const readable = markerIndex >= 0 ? sessionId.slice(markerIndex + claudeSessionMarker.length) : sessionId
  return readable.length <= 8 ? readable : readable.slice(0, 8)
}
