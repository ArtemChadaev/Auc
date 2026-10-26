const STORAGE_PREFIX = 'auc_participating_lots_';

/**
 * Получить список ID лотов, в которых участвовал пользователь (из localStorage)
 */
export function getParticipatingLotIds(userId?: string): number[] {
  if (typeof window === 'undefined') return [];
  const key = userId ? `${STORAGE_PREFIX}${userId}` : `${STORAGE_PREFIX}default`;
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(Number).filter((n) => !isNaN(n)) : [];
  } catch {
    return [];
  }
}

/**
 * Записать участие в лоте в локальное хранилище
 */
export function recordParticipatingLot(lotId: number, userId?: string): void {
  if (typeof window === 'undefined' || !lotId) return;
  const key = userId ? `${STORAGE_PREFIX}${userId}` : `${STORAGE_PREFIX}default`;
  try {
    const current = getParticipatingLotIds(userId);
    if (!current.includes(lotId)) {
      const updated = [lotId, ...current];
      localStorage.setItem(key, JSON.stringify(updated));
    }
  } catch {}
}
