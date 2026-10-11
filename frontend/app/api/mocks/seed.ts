import type { User, Lot, Item, Bid, Hold, LedgerEntry, Document } from '~/api/types';

export const SEED_USERS: User[] = [
  {
    id: 'user-me-1',
    name: 'Иван Разработчик',
    email: 'ivan@example.com',
    avatar_id: 'avatar-1',
    balance: 45000,
    hold: 5500,
    created_at: new Date(Date.now() - 30 * 86400000).toISOString(),
  },
  {
    id: 'user-alice',
    name: 'Алиса 3D-Артист',
    email: 'alice@example.com',
    avatar_id: 'avatar-2',
    balance: 120000,
    hold: 14500,
    created_at: new Date(Date.now() - 40 * 86400000).toISOString(),
  },
  {
    id: 'user-bob',
    name: 'Боб Sound Designer',
    email: 'bob@example.com',
    avatar_id: 'avatar-3',
    balance: 85000,
    hold: 0,
    created_at: new Date(Date.now() - 60 * 86400000).toISOString(),
  },
];

export const PRESET_AVATARS = [
  { id: 'avatar-1', name: 'Лиса-кодер', color: 'bg-orange-500' },
  { id: 'avatar-2', name: 'Кот-моделлер', color: 'bg-emerald-500' },
  { id: 'avatar-3', name: 'Медведь-архитектор', color: 'bg-blue-500' },
  { id: 'avatar-4', name: 'Сова-дизайнер', color: 'bg-purple-500' },
  { id: 'avatar-5', name: 'Волк-хакер', color: 'bg-slate-700' },
];

export const SEED_DOCUMENTS: Document[] = [
  {
    id: 1,
    name: 'terms',
    version: '1.0',
    is_major: true,
    published_at: new Date(Date.now() - 100 * 86400000).toISOString(),
    text: `# Пользовательское соглашение

1. Предмет соглашения
Настоящая платформа предназначена для аукциона цифровых файлов и виртуальных предметов (3D-модели, исходники, звуковые файлы, архивы, графические материалы). Все расчеты производятся в демонстрационных виртуальных Coins.

2. Права и скачивание файлов
Победитель аукциона получает право на скачивание исходного файла лота из защищённого хранилища (S3) после завершения торгов.

3. Правила ставок и Hold
При каждой ставке сумма замораживается на счёте покупателя (Hold). Если вашу ставку перебивают, сумма немедленно размораживается в реальном времени.`,
  },
  {
    id: 2,
    name: 'privacy',
    version: '1.0',
    is_major: true,
    published_at: new Date(Date.now() - 100 * 86400000).toISOString(),
    text: `# Политика конфиденциальности

1. Сбор и использование данных
Платформа собирает только минимально необходимые данные для аутентификации и сессий (Email, имя, аватар).

2. Защита цифрового контента
Загружаемые цифровые файлы хранятся в защищённом S3 хранилище и доступны для скачивания только подтверждённым владельцам.`,
  },
  {
    id: 3,
    name: 'rules',
    version: '1.0',
    is_major: true,
    published_at: new Date(Date.now() - 100 * 86400000).toISOString(),
    text: `# Правила аукциона цифровых файлов

1. Минимальный шаг
Ставка не может быть ниже текущей стоимости плюс минимальный шаг (min_step).

2. Защита от снайпинга (Anti-Sniping)
Если ставка сделана в последние 60 секунд до окончания торгов, сервер автоматически продлевает таймер на 2 минуты, гарантируя честность для всех участников.`,
  },
];

export const SEED_ITEMS: Item[] = [
  {
    id: 101,
    creator_id: 'user-me-1',
    owner_id: 'user-me-1',
    s3_id: 's3-item-1',
    type: '3d',
    metadata: {
      title: '3D-модель: Киберпанк мех',
      file_size: '124 MB',
      file_format: '.FBX / .OBJ',
      checksum: 'sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    },
    created_at: new Date(Date.now() - 15 * 86400000).toISOString(),
  },
  {
    id: 102,
    creator_id: 'user-alice',
    owner_id: 'user-alice',
    s3_id: 's3-item-2',
    type: 'image',
    metadata: {
      title: 'Исходник арта: Неоновый Город (PSD 8K)',
      file_size: '480 MB',
      file_format: '.PSD / .TIFF',
    },
    created_at: new Date(Date.now() - 12 * 86400000).toISOString(),
  },
  {
    id: 103,
    creator_id: 'user-bob',
    owner_id: 'user-bob',
    s3_id: 's3-item-3',
    type: 'audio',
    metadata: {
      title: 'Мастер-пак саундтреков (WAV 96/24)',
      file_size: '650 MB',
      file_format: '.WAV / Stems',
    },
    created_at: new Date(Date.now() - 10 * 86400000).toISOString(),
  },
  {
    id: 104,
    creator_id: 'user-alice',
    owner_id: 'user-alice',
    s3_id: 's3-item-4',
    type: 'archive',
    metadata: {
      title: 'Игровые шейдеры для Unreal Engine 5',
      file_size: '310 MB',
      file_format: '.ZIP / .uasset',
    },
    created_at: new Date(Date.now() - 5 * 86400000).toISOString(),
  },
  {
    id: 105,
    creator_id: 'user-me-1',
    owner_id: 'user-me-1',
    s3_id: 's3-item-5',
    type: 'document',
    metadata: {
      title: 'Геймдизайн-документ и Библия Лора',
      file_size: '32 MB',
      file_format: '.PDF / .DOCX',
    },
    created_at: new Date(Date.now() - 2 * 86400000).toISOString(),
  },
  {
    id: 106,
    creator_id: 'user-alice',
    owner_id: 'user-alice',
    s3_id: 's3-item-6',
    type: '3d',
    metadata: {
      title: 'Low-poly набор космических кораблей (GLTF / Blender)',
      file_size: '86 MB',
      file_format: '.GLTF / .BLEND',
    },
    created_at: new Date(Date.now() - 4 * 86400000).toISOString(),
  },
  {
    id: 107,
    creator_id: 'user-bob',
    owner_id: 'user-bob',
    s3_id: 's3-item-7',
    type: 'audio',
    metadata: {
      title: 'Эмбиент-пак: Звуки космоса и заброшенных станций',
      file_size: '420 MB',
      file_format: '.FLAC / .WAV',
    },
    created_at: new Date(Date.now() - 3 * 86400000).toISOString(),
  },
  {
    id: 108,
    creator_id: 'user-me-1',
    owner_id: 'user-me-1',
    s3_id: 's3-item-8',
    type: 'archive',
    metadata: {
      title: 'Коллекция шейдеров постобработки (Unity URP / HDRP)',
      file_size: '180 MB',
      file_format: '.unitypackage / .HLSL',
    },
    created_at: new Date(Date.now() - 6 * 86400000).toISOString(),
  },
  {
    id: 109,
    creator_id: 'user-alice',
    owner_id: 'user-alice',
    s3_id: 's3-item-9',
    type: 'image',
    metadata: {
      title: 'Концепт-арты персонажей: Фэнтези бестиарий (PSD 4K)',
      file_size: '340 MB',
      file_format: '.PSD / .PNG',
    },
    created_at: new Date(Date.now() - 7 * 86400000).toISOString(),
  },
  {
    id: 110,
    creator_id: 'user-bob',
    owner_id: 'user-bob',
    s3_id: 's3-item-10',
    type: 'document',
    metadata: {
      title: 'Набор смарт-контрактов и скриптов бекенда (Go / Sol)',
      file_size: '15 MB',
      file_format: '.ZIP / .GO',
    },
    created_at: new Date(Date.now() - 1 * 86400000).toISOString(),
  },
  {
    id: 111,
    creator_id: 'user-alice',
    owner_id: 'user-alice',
    s3_id: 's3-item-11',
    type: '3d',
    metadata: {
      title: 'Модульный sci-fi коридор со сменными текстурами',
      file_size: '260 MB',
      file_format: '.FBX / .MAX',
    },
    created_at: new Date(Date.now() - 8 * 86400000).toISOString(),
  },
  {
    id: 112,
    creator_id: 'user-me-1',
    owner_id: 'user-me-1',
    s3_id: 's3-item-12',
    type: 'archive',
    metadata: {
      title: 'UI Кит для игр в стиле Glassmorphism & Cyber',
      file_size: '95 MB',
      file_format: '.FIG / .SVG',
    },
    created_at: new Date(Date.now() - 9 * 86400000).toISOString(),
  },
];

const now = Date.now();

export const SEED_LOTS: Lot[] = [
  // Лот 1: Короткое время окончания (90 секунд), для тестирования Anti-Sniping и завершения!
  {
    id: 1,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 102,
    item_summary: {
      id: 102,
      type: 'image',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Цифровой Арт',
    title: 'Исходник арта: Неоновый Город (PSD 8K)',
    description: {
      text: 'Многослойный исходник коммерческой иллюстрации в разрешении 8192×4608 px. Включает 45 раздельных слоев, цветокоррекцию и маски рендеринга.',
      short_text: 'Многослойный PSD 8K файл (480 МБ), 45 слоев, коммерческая лицензия.',
      media: [
        {
          id: 'media-1',
          type: 'image',
          order: 1,
          caption: 'Превью рендера',
        },
      ],
      attributes: {
        'Тип файла': 'Цифровой исходник (.PSD)',
        'Размер файла': '480 MB',
        'Разрешение': '8192 × 4608 px',
        'Цветовой профиль': 'Display P3',
      },
    },
    start_amount: 5000,
    current_bid: 7800,
    min_step: 200,
    finish_amount: null,
    bids_count: 9,
    winner: null,
    created_at: new Date(now - 3600000).toISOString(),
    started_at: new Date(now - 3600000).toISOString(),
    ends_at: new Date(now + 95 * 1000).toISOString(), // ~1.5 минуты!
    finished_at: null,
  },

  // Лот 2: Короткое время окончания (3 минуты)
  {
    id: 2,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 103,
    item_summary: {
      id: 103,
      type: 'audio',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Аудио и Музыка',
    title: 'Мастер-пак саундтреков (WAV 96/24 + Stems)',
    description: {
      text: 'Полный архив саундтрека в формате несжатого аудио 96kHz / 24bit. Включает раздельные дорожки (Stems) инструментов для динамического звука в играх.',
      short_text: 'Несжатое аудио 96/24, раздельные Stems дорожки, архив 650 МБ.',
      media: [
        {
          id: 'media-2',
          type: 'image',
          order: 1,
          caption: 'Спектрограмма и обложка',
        },
      ],
      attributes: {
        'Тип файла': 'Архив аудио (.ZIP / .WAV)',
        'Размер файла': '650 MB',
        'Формат': '96 kHz / 24 bit WAV',
        'Треков в архиве': '12 + stems',
      },
    },
    start_amount: 10000,
    current_bid: 14500,
    min_step: 500,
    finish_amount: null,
    bids_count: 14,
    winner: null,
    created_at: new Date(now - 7200000).toISOString(),
    started_at: new Date(now - 7200000).toISOString(),
    ends_at: new Date(now + 190 * 1000).toISOString(), // ~3 минуты
    finished_at: null,
  },

  // Лот 3: Активный, где лидирует текущий пользователь!
  {
    id: 3,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 104,
    item_summary: {
      id: 104,
      type: 'archive',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Игровые ассеты',
    title: 'Пак шейдеров воды и стекла для Unreal Engine 5',
    description: {
      text: 'Оптимизированные HLSL-шейдеры нового поколения с трассировкой лучей Lumen и поддержкой Nanite. Готовый пакет uasset.',
      short_text: 'Готовый архив для UE5 (.uasset), 310 МБ, исходные шейдеры.',
      media: [{ id: 'media-3', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Пакет ассетов (.uasset / .ZIP)',
        'Размер файла': '310 MB',
        'Движок': 'Unreal Engine 5.3+',
      },
    },
    start_amount: 3000,
    current_bid: 5500, // эта ставка сделана user-me-1 (hold = 5500)
    min_step: 250,
    finish_amount: null,
    bids_count: 6,
    winner: null,
    created_at: new Date(now - 86400000).toISOString(),
    started_at: new Date(now - 86400000).toISOString(),
    ends_at: new Date(now + 45 * 60000).toISOString(),
    finished_at: null,
  },

  // Лот 4: Собственный лот текущего пользователя
  {
    id: 4,
    seller_id: 'user-me-1',
    seller: {
      id: 'user-me-1',
      name: 'Иван Разработчик',
      avatar_id: 'avatar-1',
      rating: 5.0,
    },
    item_id: 101,
    item_summary: {
      id: 101,
      type: '3d',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: '3D Модели',
    title: '3D-модель: Киберпанк Боевой Мех (FBX, 4K PBR)',
    description: {
      text: 'Низкополигональная оптимизированная модель с полным риггингом и PBR-текстурами (BaseColor, Metallic, Roughness, Normal, Emissive).',
      short_text: 'Low-poly риггинг мех (.FBX), 4K PBR текстуры, 124 МБ.',
      media: [{ id: 'media-4', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': '3D Сцена (.FBX / .OBJ)',
        'Размер файла': '124 MB',
        'Полигонов': '42 000 Tris',
        'Текстуры': '4K PBR UDIM',
      },
    },
    start_amount: 15000,
    current_bid: 21000,
    min_step: 1000,
    finish_amount: null,
    bids_count: 12,
    winner: null,
    created_at: new Date(now - 12000000).toISOString(),
    started_at: new Date(now - 12000000).toISOString(),
    ends_at: new Date(now + 3 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 5: Активный лот
  {
    id: 5,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 105,
    item_summary: {
      id: 105,
      type: 'document',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Документы и Скрипты',
    title: 'Геймдизайн-документ и Библия Лора Sci-Fi RPG',
    description: {
      text: 'Полная документация игрового мира на 180 страниц с диаграммами механик, квестовыми ветками и диалогами.',
      short_text: '180 страниц документации лора и механик (.PDF / .DOCX, 32 МБ).',
      media: [{ id: 'media-5', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Документ (.PDF)',
        'Размер файла': '32 MB',
        'Страниц': '180',
      },
    },
    start_amount: 8000,
    current_bid: 11200,
    min_step: 400,
    finish_amount: null,
    bids_count: 8,
    winner: null,
    created_at: new Date(now - 18000000).toISOString(),
    started_at: new Date(now - 18000000).toISOString(),
    ends_at: new Date(now + 24 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 6: Завершенный лот
  {
    id: 6,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 101,
    item_summary: {
      id: 101,
      type: '3d',
      preview: '/placeholder.svg',
    },
    status: 'finished',
    category: '3D Модели',
    title: 'Высокополигональный скульптинг: Дракон (ZBrush ZTL)',
    description: {
      text: 'Исходный ZTL файл для ZBrush с поддержкой всех уровней сабдивов (SubTool).',
      short_text: 'Исходник ZBrush ZTL, 1.2 ГБ.',
      media: [{ id: 'media-6', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': '3D Скульптинг (.ZTL)',
        'Размер файла': '1.2 GB',
      },
    },
    start_amount: 12000,
    current_bid: 18500,
    min_step: 500,
    finish_amount: 18500,
    bids_count: 17,
    winner: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
    },
    created_at: new Date(now - 86400000 * 3).toISOString(),
    started_at: new Date(now - 86400000 * 3).toISOString(),
    ends_at: new Date(now - 86400000).toISOString(),
    finished_at: new Date(now - 86400000).toISOString(),
  },

  // Лот 7: Завершенный лот, выигранный текущим пользователем (можно скачать файл!)
  {
    id: 7,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 103,
    item_summary: {
      id: 103,
      type: 'audio',
      preview: '/placeholder.svg',
    },
    status: 'finished',
    category: 'Аудио и Музыка',
    title: 'SFX-библиотека футуристического оружия (950 звуков)',
    description: {
      text: 'Комплект звуковых эффектов для игр: лазеры, плазменные пушки, взрывы и щиты. 24bit/96kHz WAV с метаданными UCS.',
      short_text: '950 SFX звуков (.ZIP, 820 МБ), готово к интеграции.',
      media: [{ id: 'media-7', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Аудио библиотека (.ZIP / .WAV)',
        'Размер файла': '820 MB',
        'Количество файлов': '950 WAV',
      },
    },
    start_amount: 25000,
    current_bid: 34000,
    min_step: 1000,
    finish_amount: 34000,
    bids_count: 15,
    winner: {
      id: 'user-me-1',
      name: 'Иван Разработчик',
    },
    created_at: new Date(now - 86400000 * 5).toISOString(),
    started_at: new Date(now - 86400000 * 5).toISOString(),
    ends_at: new Date(now - 86400000 * 2).toISOString(),
    finished_at: new Date(now - 86400000 * 2).toISOString(),
  },

  // Лот 8: Активный
  {
    id: 8,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 106,
    item_summary: {
      id: 106,
      type: '3d',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: '3D Модели',
    title: 'Low-poly набор космических кораблей (GLTF / Blender)',
    description: {
      text: 'Пак из 8 уникальных космических истребителей, оптимизированных для мобильных и инди-проектов.',
      short_text: '8 моделей истребителей (.GLTF / .BLEND), 86 МБ, готово к игре.',
      media: [{ id: 'media-8', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': '3D Модели (.GLTF / .BLEND)',
        'Размер файла': '86 MB',
        'Моделей': '8 шт',
      },
    },
    start_amount: 4000,
    current_bid: 6200,
    min_step: 200,
    finish_amount: null,
    bids_count: 5,
    winner: null,
    created_at: new Date(now - 14400000).toISOString(),
    started_at: new Date(now - 14400000).toISOString(),
    ends_at: new Date(now + 12 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 9: Активный
  {
    id: 9,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 107,
    item_summary: {
      id: 107,
      type: 'audio',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Аудио и Музыка',
    title: 'Эмбиент-пак: Звуки космоса и заброшенных станций',
    description: {
      text: 'Глубокие аналоговые дроны, синтезаторные текстуры и металлические реверберации для создания атмосферы.',
      short_text: 'Космический эмбиент (.FLAC / .WAV), 420 МБ, 20 лупов.',
      media: [{ id: 'media-9', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Аудио архив (.FLAC / .WAV)',
        'Размер файла': '420 MB',
        'Количество лупов': '20 треков',
      },
    },
    start_amount: 2500,
    current_bid: 3900,
    min_step: 150,
    finish_amount: null,
    bids_count: 7,
    winner: null,
    created_at: new Date(now - 20000000).toISOString(),
    started_at: new Date(now - 20000000).toISOString(),
    ends_at: new Date(now + 18 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 10: Активный
  {
    id: 10,
    seller_id: 'user-me-1',
    seller: {
      id: 'user-me-1',
      name: 'Иван Разработчик',
      avatar_id: 'avatar-1',
      rating: 5.0,
    },
    item_id: 108,
    item_summary: {
      id: 108,
      type: 'archive',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Игровые ассеты',
    title: 'Коллекция шейдеров постобработки (Unity URP / HDRP)',
    description: {
      text: 'Эффекты CRT-монитора, ночного видения, тепловизора и искажений времени с полной документацией внедрения.',
      short_text: 'Пакет для Unity (.unitypackage), 180 МБ, 12 шейдеров.',
      media: [{ id: 'media-10', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Архив ассетов (.unitypackage)',
        'Размер файла': '180 MB',
        'Поддержка': 'Unity 2022+ URP/HDRP',
      },
    },
    start_amount: 7000,
    current_bid: 9500,
    min_step: 500,
    finish_amount: null,
    bids_count: 4,
    winner: null,
    created_at: new Date(now - 25000000).toISOString(),
    started_at: new Date(now - 25000000).toISOString(),
    ends_at: new Date(now + 8 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 11: Активный
  {
    id: 11,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 109,
    item_summary: {
      id: 109,
      type: 'image',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Цифровой Арт',
    title: 'Концепт-арты персонажей: Фэнтези бестиарий (PSD 4K)',
    description: {
      text: '15 детализированных существ с разбором силуэтов, анатомических деталей и вариантов окраса.',
      short_text: '15 исходников персонажей (.PSD 4K), 340 МБ.',
      media: [{ id: 'media-11', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Цифровой исходник (.PSD)',
        'Размер файла': '340 MB',
        'Разрешение': '3840 × 2160 px',
      },
    },
    start_amount: 11000,
    current_bid: 16400,
    min_step: 600,
    finish_amount: null,
    bids_count: 11,
    winner: null,
    created_at: new Date(now - 30000000).toISOString(),
    started_at: new Date(now - 30000000).toISOString(),
    ends_at: new Date(now + 36 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 12: Активный
  {
    id: 12,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 110,
    item_summary: {
      id: 110,
      type: 'document',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Документы и Скрипты',
    title: 'Набор смарт-контрактов и скриптов бекенда (Go / Sol)',
    description: {
      text: 'Проверенные архитектурные шаблоны микросервисов на Go и оптимизированные смарт-контракты с тестами.',
      short_text: 'Исходный код (.ZIP / .GO), 15 МБ, чистая архитектура.',
      media: [{ id: 'media-12', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Архив исходного кода (.ZIP)',
        'Размер файла': '15 MB',
        'Языки': 'Go 1.24, Solidity',
      },
    },
    start_amount: 18000,
    current_bid: 24000,
    min_step: 1000,
    finish_amount: null,
    bids_count: 8,
    winner: null,
    created_at: new Date(now - 1200000).toISOString(),
    started_at: new Date(now - 1200000).toISOString(),
    ends_at: new Date(now + 5 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 13: Активный
  {
    id: 13,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 111,
    item_summary: {
      id: 111,
      type: '3d',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: '3D Модели',
    title: 'Модульный sci-fi коридор со сменными текстурами',
    description: {
      text: 'Готовый модульный конструктор уровня: полы, стены, потолки, консоли и двери с анимацией открытия.',
      short_text: 'Модульный 3D конструктор (.FBX / .MAX), 260 МБ.',
      media: [{ id: 'media-13', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': '3D Конструктор (.FBX)',
        'Размер файла': '260 MB',
        'Деталей': '48 модулей',
      },
    },
    start_amount: 8500,
    current_bid: 12100,
    min_step: 400,
    finish_amount: null,
    bids_count: 6,
    winner: null,
    created_at: new Date(now - 15000000).toISOString(),
    started_at: new Date(now - 15000000).toISOString(),
    ends_at: new Date(now + 16 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 14: Активный
  {
    id: 14,
    seller_id: 'user-me-1',
    seller: {
      id: 'user-me-1',
      name: 'Иван Разработчик',
      avatar_id: 'avatar-1',
      rating: 5.0,
    },
    item_id: 112,
    item_summary: {
      id: 112,
      type: 'archive',
      preview: '/placeholder.svg',
    },
    status: 'active',
    category: 'Игровые ассеты',
    title: 'UI Кит для игр в стиле Glassmorphism & Cyber',
    description: {
      text: 'Векторные элементы интерфейса: кнопки, окна инвентаря, полосы здоровья, мини-карты и 120 векторных иконок.',
      short_text: 'UI Kit Figma + SVG (.FIG / .SVG), 95 МБ, 120 иконок.',
      media: [{ id: 'media-14', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Дизайн ассеты (.FIG / .SVG)',
        'Размер файла': '95 MB',
        'Иконок': '120 векторных',
      },
    },
    start_amount: 5000,
    current_bid: 7600,
    min_step: 300,
    finish_amount: null,
    bids_count: 9,
    winner: null,
    created_at: new Date(now - 6000000).toISOString(),
    started_at: new Date(now - 6000000).toISOString(),
    ends_at: new Date(now + 28 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 15: Предстоящий аукцион (старт через 3 часа)
  {
    id: 15,
    seller_id: 'user-alice',
    seller: {
      id: 'user-alice',
      name: 'Алиса 3D-Артист',
      avatar_id: 'avatar-2',
      rating: 4.9,
    },
    item_id: 101,
    item_summary: {
      id: 101,
      type: '3d',
      preview: '/placeholder.svg',
    },
    status: 'approved',
    category: '3D Модели',
    title: 'Эксклюзив: Анимированный мех Новой Эры (GLTF 4K)',
    description: {
      text: 'Эксклюзивная премьера нового сета роботов с 24 процедурными анимациями и звуковым сопровождением.',
      short_text: 'Премьерный лот: 24 анимации (.GLTF / .FBX), 4K текстуры.',
      media: [{ id: 'media-15', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': '3D Сцена (.GLTF)',
        'Размер файла': '210 MB',
        'Анимаций': '24 клипа',
      },
    },
    start_amount: 20000,
    current_bid: 20000,
    min_step: 1000,
    finish_amount: null,
    bids_count: 0,
    winner: null,
    created_at: new Date(now - 3600000).toISOString(),
    started_at: new Date(now + 3 * 3600000 + 45 * 60000).toISOString(), // Старт через ~3.7 часа
    ends_at: new Date(now + 27 * 3600000).toISOString(),
    finished_at: null,
  },

  // Лот 16: Предстоящий аукцион (старт завтра в 08:00)
  {
    id: 16,
    seller_id: 'user-bob',
    seller: {
      id: 'user-bob',
      name: 'Боб Sound Designer',
      avatar_id: 'avatar-3',
      rating: 4.8,
    },
    item_id: 103,
    item_summary: {
      id: 103,
      type: 'audio',
      preview: '/placeholder.svg',
    },
    status: 'approved',
    category: 'Аудио',
    title: 'Саундтрек к блокбастеру «Хроники Бездны» (Orchestral Stems)',
    description: {
      text: 'Оркестровый саундтрек, записанный с живым симфоническим оркестром. 48 дорожек 192kHz/32bit float.',
      short_text: 'Живой симфонический оркестр (.WAV 192/32), 1.8 ГБ.',
      media: [{ id: 'media-16', type: 'image', order: 1 }],
      attributes: {
        'Тип файла': 'Оркестровый пак (.WAV)',
        'Размер файла': '1.8 GB',
        'Качество': '192 kHz / 32-bit float',
      },
    },
    start_amount: 35000,
    current_bid: 35000,
    min_step: 1500,
    finish_amount: null,
    bids_count: 0,
    winner: null,
    created_at: new Date(now - 7200000).toISOString(),
    started_at: new Date(now + 14 * 3600000).toISOString(), // Старт через 14 часов
    ends_at: new Date(now + 38 * 3600000).toISOString(),
    finished_at: null,
  },
];

export const SEED_BIDS: Bid[] = [
  {
    id: 1,
    lot_id: 1,
    user_id: 'user-bob',
    user_name: 'Боб Sound Designer',
    amount: 5200,
    idempotency_key: 'idemp-1',
    created_at: new Date(now - 3000000).toISOString(),
  },
  {
    id: 2,
    lot_id: 1,
    user_id: 'user-me-1',
    user_name: 'Иван Разработчик',
    amount: 6000,
    idempotency_key: 'idemp-2',
    created_at: new Date(now - 2500000).toISOString(),
  },
  {
    id: 3,
    lot_id: 1,
    user_id: 'user-bob',
    user_name: 'Боб Sound Designer',
    amount: 7800,
    idempotency_key: 'idemp-3',
    created_at: new Date(now - 400000).toISOString(),
  },
  {
    id: 4,
    lot_id: 3,
    user_id: 'user-alice',
    user_name: 'Алиса 3D-Артист',
    amount: 4500,
    idempotency_key: 'idemp-4',
    created_at: new Date(now - 4000000).toISOString(),
  },
  {
    id: 5,
    lot_id: 3,
    user_id: 'user-me-1',
    user_name: 'Иван Разработчик',
    amount: 5500,
    idempotency_key: 'idemp-5',
    created_at: new Date(now - 2000000).toISOString(),
  },
];

export const SEED_HOLDS: Hold[] = [
  {
    id: 1,
    lot_id: 3,
    user_id: 'user-me-1',
    bid_id: 5,
    amount: 5500,
    status: 'active',
    created_at: new Date(now - 2000000).toISOString(),
  },
  {
    id: 2,
    lot_id: 1,
    user_id: 'user-bob',
    bid_id: 3,
    amount: 7800,
    status: 'active',
    created_at: new Date(now - 400000).toISOString(),
  },
];

export const SEED_LEDGER: LedgerEntry[] = [
  {
    id: 1,
    user_id: 'user-me-1',
    amount: 50000,
    balance_after: 50000,
    type: 'top_up',
    lot_id: null,
    created_at: new Date(now - 86400000 * 2).toISOString(),
  },
  {
    id: 2,
    user_id: 'user-me-1',
    amount: 1000,
    balance_after: 51000,
    type: 'top_up',
    lot_id: null,
    created_at: new Date(now - 86400000).toISOString(),
  },
  {
    id: 3,
    user_id: 'user-me-1',
    amount: -6000,
    balance_after: 45000,
    type: 'transfer_to_seller',
    lot_id: 7,
    created_at: new Date(now - 86400000 * 2).toISOString(),
  },
];
