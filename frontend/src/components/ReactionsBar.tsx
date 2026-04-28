interface ReactionItem {
  emoji: string
  count: number
  user_ids: string[]
}

interface ReactionsBarProps {
  reactions: ReactionItem[]
  currentUserId?: string
  onToggle: (emoji: string) => void
}

export default function ReactionsBar({ reactions, currentUserId, onToggle }: ReactionsBarProps) {
  if (!reactions || reactions.length === 0) return null

  return (
    <div className="reactions-bar">
      {reactions.map(r => {
        const isActive = currentUserId ? r.user_ids.includes(currentUserId) : false
        return (
          <button
            key={r.emoji}
            className={`reaction-chip${isActive ? ' active' : ''}`}
            onClick={() => onToggle(r.emoji)}
            title={`${r.count}`}
            type="button"
          >
            <span className="reaction-chip-emoji">{r.emoji}</span>
            <span className="reaction-chip-count">{r.count}</span>
          </button>
        )
      })}
    </div>
  )
}
