export interface AdminStats {
  users: { total: number }
  rooms: { total: number }
  conferences: { active: number }
  messages: { today: number }
}

interface ApiStatsResponse {
  users?: { total?: number }
  rooms?: { total?: number }
  conferences?: { active?: number }
  messages?: { today?: number }
}

export function normalizeStats(data: ApiStatsResponse): AdminStats {
  return {
    users: { total: Number(data?.users?.total ?? 0) },
    rooms: { total: Number(data?.rooms?.total ?? 0) },
    conferences: { active: Number(data?.conferences?.active ?? 0) },
    messages: { today: Number(data?.messages?.today ?? 0) },
  }
}
