/**
 * Денежные утилиты для работы с виртуальной валютой Coins (🪙).
 * Все суммы в системе - целые числа.
 */

export function formatCoins(amount: number | null | undefined): string {
  if (amount === null || amount === undefined || isNaN(amount)) {
    return '0 🪙';
  }
  const formatted = new Intl.NumberFormat('ru-RU').format(amount);
  return `${formatted} 🪙`;
}

export function formatCoinsOnlyNumber(amount: number | null | undefined): string {
  if (amount === null || amount === undefined || isNaN(amount)) {
    return '0';
  }
  return new Intl.NumberFormat('ru-RU').format(amount);
}
