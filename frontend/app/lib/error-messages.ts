/**
 * Словарь и маппинг ошибок для интерфейса.
 * Разделение на race (ожидаемые гонки), user (ошибки ввода), auth (авторизация), system (сеть/сбои).
 */

import type { ErrorKind } from '~/api/errors';

export interface ErrorDescriptor {
  title: string;
  description: string;
  kind: ErrorKind;
  severity: 'warning' | 'destructive' | 'info';
}

export const ERROR_MESSAGES: Record<string, ErrorDescriptor> = {
  unauthorized: {
    title: 'Требуется вход',
    description: 'Пожалуйста, войдите в аккаунт для выполнения этого действия.',
    kind: 'auth',
    severity: 'warning',
  },
  forbidden: {
    title: 'Доступ ограничен',
    description: 'У вас нет прав для совершения этой операции.',
    kind: 'auth',
    severity: 'destructive',
  },
  lot_not_found: {
    title: 'Лот не найден',
    description: 'Запрошенный лот не существует или был удален.',
    kind: 'user',
    severity: 'destructive',
  },
  lot_not_active: {
    title: 'Торги не активны',
    description: 'Аукцион по данному лоту уже завершён или ещё не начался.',
    kind: 'race',
    severity: 'warning',
  },
  outbid: {
    title: 'Вас опередили!',
    description: 'Пока вы делали ставку, другой участник предложил большую сумму. Текущая ставка обновлена.',
    kind: 'race',
    severity: 'warning',
  },
  bid_too_low: {
    title: 'Ставка слишком мала',
    description: 'Сумма вашей ставки должна быть не меньше текущей цены плюс минимальный шаг.',
    kind: 'user',
    severity: 'destructive',
  },
  insufficient_funds: {
    title: 'Недостаточно Coins',
    description: 'На вашем балансе не хватает свободных средств с учётом текущих заморозок (hold).',
    kind: 'user',
    severity: 'destructive',
  },
  cannot_bid_own_lot: {
    title: 'Нельзя ставить на свой лот',
    description: 'Вы являетесь продавцом этого лота и не можете участвовать в торгах.',
    kind: 'user',
    severity: 'warning',
  },
  already_highest_bidder: {
    title: 'Вы уже лидируете',
    description: 'Ваша ставка уже является наивысшей в данном аукционе.',
    kind: 'user',
    severity: 'info',
  },
  email_already_exists: {
    title: 'Email уже занят',
    description: 'Пользователь с таким адресом электронной почты уже зарегистрирован.',
    kind: 'user',
    severity: 'destructive',
  },
  invalid_request_body: {
    title: 'Некорректные данные',
    description: 'Проверьте правильность заполнения всех полей формы.',
    kind: 'user',
    severity: 'destructive',
  },
  network_error: {
    title: 'Проблема с соединением',
    description: 'Не удалось связаться с сервером. Проверьте интернет-соединение.',
    kind: 'system',
    severity: 'destructive',
  },
  system_error: {
    title: 'Сервис временно недоступен',
    description: 'Произошла непредвиденная ошибка. Мы уже работаем над ее устранением.',
    kind: 'system',
    severity: 'destructive',
  },

  // 429 Too Many Requests (Rate Limits)
  rate_limit: {
    title: 'Слишком много запросов (429)',
    description: 'Превышен лимит запросов к серверу. Пожалуйста, подождите немного перед повторной попыткой.',
    kind: 'race',
    severity: 'warning',
  },
  free_coin_limit: {
    title: 'Бонус уже получен',
    description: 'Бесплатные монеты (+1 000 🪙) можно получить только один раз в сутки (сброс в 00:00 UTC).',
    kind: 'user',
    severity: 'info',
  },
  photo_upload_limit: {
    title: 'Лимит загрузки фотографий',
    description: 'Вы можете загружать не более 30 фотографий в час. Пожалуйста, подождите немного.',
    kind: 'user',
    severity: 'warning',
  },
  download_limit: {
    title: 'Лимит скачиваний',
    description: 'Превышен суточный лимит скачиваний (максимум 1 скачивание в сутки).',
    kind: 'user',
    severity: 'warning',
  },
  bid_rate_limit: {
    title: 'Слишком частые ставки',
    description: 'Пожалуйста, подождите пару секунд перед размещением следующей ставки.',
    kind: 'race',
    severity: 'warning',
  },
};

export function getErrorMessage(code: string, fallbackMessage?: string): ErrorDescriptor {
  if (ERROR_MESSAGES[code]) {
    if (fallbackMessage && fallbackMessage !== code && !fallbackMessage.startsWith('http_error')) {
      return {
        ...ERROR_MESSAGES[code],
        description: fallbackMessage,
      };
    }
    return ERROR_MESSAGES[code];
  }

  // Динамический разбор текста ошибки при неспецифичном коде
  if (fallbackMessage) {
    const lower = fallbackMessage.toLowerCase();
    if (lower.includes('free coin') || lower.includes('once per day') || lower.includes('бесплатные монеты') || lower.includes('бонус')) {
      return ERROR_MESSAGES.free_coin_limit;
    }
    if (lower.includes('photo upload') || lower.includes('фото')) {
      return ERROR_MESSAGES.photo_upload_limit;
    }
    if (lower.includes('429') || lower.includes('rate limit') || lower.includes('слишком много запросов')) {
      return {
        title: 'Слишком много запросов (429)',
        description: fallbackMessage,
        kind: 'race',
        severity: 'warning',
      };
    }
  }

  return {
    title: 'Ошибка операции',
    description: fallbackMessage || `Код ошибки: ${code}`,
    kind: 'system',
    severity: 'destructive',
  };
}
