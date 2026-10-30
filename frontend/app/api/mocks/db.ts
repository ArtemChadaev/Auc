/**
 * In-memory база данных мок-сервера.
 * Точно воспроизводит правила и инварианты Go бэкенда (Hold, Balance, Anti-Sniping).
 */

import { ApiError } from '~/api/errors';
import type {
  User,
  Lot,
  Item,
  ItemType,
  Bid,
  Hold,
  LedgerEntry,
  Document,
  LotFilter,
  LotListResponse,
  PlaceBidResult,
  CreateLotInput,
  CreateItemInput,
  UploadUrlRequest,
  UploadUrlResponse,
  ConfirmUploadResponse,
  UploadStatusResponse,
  DownloadUrlResponse,
} from '~/api/types';
import {
  SEED_USERS,
  SEED_DOCUMENTS,
  SEED_ITEMS,
  SEED_LOTS,
  SEED_BIDS,
  SEED_HOLDS,
  SEED_LEDGER,
} from './seed';

class MockDatabase {
  private users: Map<string, User> = new Map();
  private lots: Map<number, Lot> = new Map();
  private items: Map<number, Item> = new Map();
  private bids: Bid[] = [];
  private holds: Hold[] = [];
  private ledger: LedgerEntry[] = [];
  private documents: Map<number, Document> = new Map();
  private consents: Set<string> = new Set(); // "docId:userId"
  private uploads: Map<
    string,
    {
      key: string;
      userId: string;
      filename: string;
      fileSize: number;
      sha256Hex: string;
      status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';
      item?: Item;
    }
  > = new Map();
  private downloadLimits: Set<string> = new Set(); // "userId:itemId:YYYY-MM-DD"
  private currentUserId: string = 'user-me-1';

  // Смещение серверного времени (для симуляции разницы часов)
  public serverTimeOffsetMs = 0;

  constructor() {
    this.reset();
  }

  public reset() {
    this.users.clear();
    SEED_USERS.forEach((u) => this.users.set(u.id, { ...u }));

    this.lots.clear();
    SEED_LOTS.forEach((l) => this.lots.set(l.id, JSON.parse(JSON.stringify(l))));

    this.items.clear();
    SEED_ITEMS.forEach((i) => this.items.set(i.id, { ...i }));

    this.bids = JSON.parse(JSON.stringify(SEED_BIDS));
    this.holds = JSON.parse(JSON.stringify(SEED_HOLDS));
    this.ledger = JSON.parse(JSON.stringify(SEED_LEDGER));

    this.documents.clear();
    SEED_DOCUMENTS.forEach((d) => this.documents.set(d.id, { ...d }));
    this.uploads.clear();
    this.downloadLimits.clear();
  }

  // --- Auth & Users ---
  public getCurrentUser(): User {
    const user = this.users.get(this.currentUserId);
    if (!user) {
      throw new ApiError('unauthorized', 'Пользователь не найден', 401);
    }
    return { ...user };
  }

  public setCurrentUserId(id: string) {
    if (this.users.has(id)) {
      this.currentUserId = id;
    }
  }

  public register(name: string, email: string, avatarId?: string): User {
    for (const u of this.users.values()) {
      if (u.email.toLowerCase() === email.toLowerCase()) {
        throw new ApiError('email_already_exists', 'Email уже занят', 409);
      }
    }
    const newUser: User = {
      id: `user-${Date.now()}`,
      name,
      email: email.toLowerCase(),
      avatar_id: avatarId || 'avatar-1',
      balance: 10000, // Стартовый приветственный баланс
      hold: 0,
      created_at: new Date().toISOString(),
    };
    this.users.set(newUser.id, newUser);
    this.currentUserId = newUser.id;

    this.ledger.push({
      id: Date.now(),
      user_id: newUser.id,
      amount: 10000,
      balance_after: 10000,
      type: 'top_up',
      lot_id: null,
      created_at: new Date().toISOString(),
    });

    return { ...newUser };
  }

  public login(email: string): User {
    for (const u of this.users.values()) {
      if (u.email.toLowerCase() === email.toLowerCase()) {
        this.currentUserId = u.id;
        return { ...u };
      }
    }
    throw new ApiError('unauthorized', 'Неверный email или пароль', 401);
  }

  public updateUserName(name: string): User {
    const user = this.getCurrentUser();
    user.name = name;
    this.users.set(user.id, user);
    return { ...user };
  }

  public topUpDailyCoins(): { user: User; added: boolean } {
    const user = this.getCurrentUser();
    // Начисляем ежедневный бонус 1000 coins
    const bonus = 1000;
    user.balance += bonus;
    this.users.set(user.id, user);

    // Ledger only updated via server refresh

    return { user: { ...user }, added: true };
  }

  // --- Lots ---
  public listLots(filter: LotFilter): LotListResponse {
    let list = Array.from(this.lots.values());

    // Фильтрация по статусу
    if (filter.status === 'upcoming') {
      const nowMs = Date.now();
      list = list.filter((l) => {
        const startMs = l.started_at ? new Date(l.started_at).getTime() : 0;
        return startMs > nowMs && l.status !== 'finished' && l.status !== 'cancelled';
      });
    } else if (filter.status === 'active') {
      const nowMs = Date.now();
      list = list.filter((l) => {
        const startMs = l.started_at ? new Date(l.started_at).getTime() : 0;
        return l.status === 'active' && startMs <= nowMs;
      });
    } else if (filter.status) {
      list = list.filter((l) => l.status === filter.status);
    }

    // Фильтрация по поиску
    if (filter.query) {
      const q = filter.query.toLowerCase();
      list = list.filter(
        (l) =>
          l.title.toLowerCase().includes(q) ||
          l.category.toLowerCase().includes(q) ||
          l.description.short_text?.toLowerCase().includes(q)
      );
    }

    // Фильтрация по категории (по коду типа image, 3d, audio, video, document, archive или по названию)
    if (filter.category) {
      const cat = filter.category.toLowerCase();
      list = list.filter(
        (l) =>
          l.category.toLowerCase() === cat ||
          l.item_summary?.type.toLowerCase() === cat
      );
    }

    // Фильтрация по цене
    if (filter.min_price !== undefined) {
      list = list.filter((l) => l.current_bid >= filter.min_price!);
    }
    if (filter.max_price !== undefined) {
      list = list.filter((l) => l.current_bid <= filter.max_price!);
    }

    // Сортировка
    if (filter.sort === 'price_asc') {
      list.sort((a, b) => a.current_bid - b.current_bid);
    } else if (filter.sort === 'price_desc') {
      list.sort((a, b) => b.current_bid - a.current_bid);
    } else if (filter.sort === 'popular') {
      list.sort((a, b) => b.bids_count - a.bids_count);
    } else if (filter.sort === 'ending_soon') {
      list.sort((a, b) => {
        const timeA = a.ends_at ? new Date(a.ends_at).getTime() : Infinity;
        const timeB = b.ends_at ? new Date(b.ends_at).getTime() : Infinity;
        return timeA - timeB;
      });
    } else {
      // newest
      list.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    }

    const total = list.length;
    const page = filter.page || 1;
    const limit = filter.limit || 6;
    const total_pages = Math.ceil(total / limit) || 1;
    const offset = (page - 1) * limit;
    const items = list.slice(offset, offset + limit);

    return {
      items,
      total,
      page,
      limit,
      total_pages,
    };
  }

  public getLot(id: number): Lot {
    const lot = this.lots.get(id);
    if (!lot) {
      throw new ApiError('lot_not_found', 'Лот не найден', 404);
    }
    return JSON.parse(JSON.stringify(lot));
  }

  public createLot(input: CreateLotInput): Lot {
    const user = this.getCurrentUser();
    const id = Date.now();
    const startedAt = input.started_at || new Date().toISOString();
    const endsAt = input.ends_at || new Date(Date.now() + input.duration_minutes * 60000).toISOString();

    const newLot: Lot = {
      id,
      seller_id: user.id,
      seller: {
        id: user.id,
        name: user.name,
        avatar_id: user.avatar_id,
        rating: 5.0,
      },
      item_id: input.item_id,
      item_summary: {
        id: input.item_id,
        type: 'image',
        preview: '/placeholder.svg',
      },
      status: 'active',
      category: input.category,
      title: input.title,
      description: {
        text: input.text,
        short_text: input.short_text,
        media: [{ id: `media-${id}`, type: 'image', order: 1 }],
      },
      start_amount: input.start_amount,
      current_bid: input.start_amount,
      min_step: input.min_step,
      finish_amount: null,
      bids_count: 0,
      winner: null,
      created_at: startedAt,
      started_at: startedAt,
      ends_at: endsAt,
      finished_at: null,
    };

    this.lots.set(id, newLot);
    return JSON.parse(JSON.stringify(newLot));
  }

  // --- Bids & Торги ---
  public listBids(lotId: number): Bid[] {
    return this.bids
      .filter((b) => b.lot_id === lotId)
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }

  public placeBid(lotId: number, amount: number, userId?: string): PlaceBidResult {
    const bidderId = userId || this.currentUserId;
    const user = this.users.get(bidderId);
    if (!user) {
      throw new ApiError('unauthorized', 'Не авторизован', 401);
    }

    const lot = this.lots.get(lotId);
    if (!lot) {
      throw new ApiError('lot_not_found', 'Лот не найден', 404);
    }

    // 1. Проверка статуса
    if (lot.status !== 'active') {
      throw new ApiError('lot_not_active', 'Аукцион по лоту не активен', 400);
    }

    // 2. Нельзя ставить на свой лот
    if (lot.seller_id === bidderId) {
      throw new ApiError('cannot_bid_own_lot', 'Нельзя делать ставку на собственный лот', 400);
    }

    // 3. Последняя ставка
    const lotBids = this.bids
      .filter((b) => b.lot_id === lotId)
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

    const highestBid = lotBids[0];

    // 4. Нельзя перебивать свою ставку, если вы уже лидер
    if (highestBid && highestBid.user_id === bidderId) {
      throw new ApiError('already_highest_bidder', 'Вы уже являетесь лидером в этом лоте', 400);
    }

    // 5. Минимальная допустимая сумма
    const minRequired = highestBid ? lot.current_bid + lot.min_step : lot.start_amount;
    if (amount < minRequired) {
      throw new ApiError(
        'bid_too_low',
        `Ставка слишком мала. Минимальная ставка: ${minRequired} Coins`,
        400,
        { min_bid: minRequired }
      );
    }

    // 6. Проверка баланса с учетом текущего hold
    const availableBalance = user.balance - user.hold;
    if (availableBalance < amount) {
      throw new ApiError(
        'insufficient_funds',
        `Недостаточно средств. Доступно: ${availableBalance} Coins`,
        400,
        { available: availableBalance, required: amount }
      );
    }

    // 7. Снятие hold с предыдущего лидера (если был)
    if (highestBid) {
      const prevBidder = this.users.get(highestBid.user_id);
      const prevHold = this.holds.find(
        (h) => h.lot_id === lotId && h.user_id === highestBid.user_id && h.status === 'active'
      );
      if (prevHold && prevBidder) {
        prevHold.status = 'released';
        prevBidder.hold = Math.max(0, prevBidder.hold - prevHold.amount);
      }
    }

    // 8. Фиксация нового Hold у текущего пользователя
    const newBidId = Date.now();
    user.hold += amount;

    const newHold: Hold = {
      id: Date.now() + 1,
      lot_id: lotId,
      user_id: bidderId,
      bid_id: newBidId,
      amount,
      status: 'active',
      created_at: new Date().toISOString(),
    };
    this.holds.push(newHold);

    // 9. Создание ставки
    const newBid: Bid = {
      id: newBidId,
      lot_id: lotId,
      user_id: bidderId,
      user_name: user.name,
      amount,
      idempotency_key: `idemp-${newBidId}`,
      created_at: new Date().toISOString(),
    };
    this.bids.unshift(newBid);

    // 10. Обновление лота
    lot.current_bid = amount;
    lot.bids_count += 1;

    // 11. Anti-sniping: если ставка сделана менее чем за 60 секунд до конца, продлеваем на 120 секунд!
    if (lot.ends_at) {
      const remainingMs = new Date(lot.ends_at).getTime() - Date.now();
      if (remainingMs > 0 && remainingMs < 60 * 1000) {
        lot.ends_at = new Date(new Date(lot.ends_at).getTime() + 120 * 1000).toISOString();
      }
    }

    return {
      bid: newBid,
      lot: JSON.parse(JSON.stringify(lot)),
      user_balance: user.balance,
      user_hold: user.hold,
    };
  }

  // --- Items & Upload Flow ---
  public listUserItems(userId?: string): Item[] {
    const uid = userId || this.currentUserId;
    return Array.from(this.items.values())
      .filter((i) => i.owner_id === uid)
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }

  public getItem(id: number): Item {
    const item = this.items.get(id);
    if (!item) {
      throw new ApiError('item_not_found', 'Предмет не найден', 404);
    }
    return JSON.parse(JSON.stringify(item));
  }

  public createItem(input: CreateItemInput): Item {
    const user = this.getCurrentUser();
    const id = Date.now();
    const item: Item = {
      id,
      creator_id: user.id,
      owner_id: user.id,
      s3_id: `s3-${id}`,
      type: input.type,
      metadata: {
        title: input.name,
        description: input.description,
        ...(input.metadata || {}),
      },
      created_at: new Date().toISOString(),
    };
    this.items.set(id, item);
    return item;
  }

  public createUploadUrl(req: UploadUrlRequest): UploadUrlResponse {
    const user = this.getCurrentUser();
    if (req.file_size <= 0) {
      throw new ApiError('invalid_file_size', 'Размер файла должен быть больше 0', 400);
    }
    if (!req.filename) {
      throw new ApiError('invalid_filename', 'Имя файла обязательно', 400);
    }
    if (!req.sha_256_hex || req.sha_256_hex.length !== 64) {
      throw new ApiError('invalid_sha256', 'Некорректный формат SHA-256 (требуется 64 hex символа)', 400);
    }

    const key = `upload-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const url = `https://s3.local-mock.internal/tmp/${key}`;

    this.uploads.set(key, {
      key,
      userId: user.id,
      filename: req.filename,
      fileSize: req.file_size,
      sha256Hex: req.sha_256_hex,
      status: 'pending',
    });

    return { url, key };
  }

  public confirmUpload(key: string): ConfirmUploadResponse {
    const upload = this.uploads.get(key);
    if (!upload) {
      throw new ApiError('upload_not_found', 'Сессия загрузки не найдена', 404);
    }

    upload.status = 'processing';

    // Фоновая генерация воркером Item и Metadata (имитация воркер-пула N=5)
    setTimeout(() => {
      if (upload.status === 'processing') {
        const id = Date.now();
        const type = this.detectItemType(upload.filename);
        const metadata = this.generateItemMetadata(
          upload.filename,
          upload.fileSize,
          upload.sha256Hex,
          type
        );

        const newItem: Item = {
          id,
          creator_id: upload.userId,
          owner_id: upload.userId,
          s3_id: `s3-${upload.key}`,
          type,
          metadata,
          created_at: new Date().toISOString(),
        };

        this.items.set(id, newItem);
        upload.status = 'completed';
        upload.item = newItem;
      }
    }, 1200);

    return {
      status: 'processing',
      message: 'file sample verified, background processing started',
    };
  }

  public getUploadStatus(key: string): UploadStatusResponse {
    const upload = this.uploads.get(key);
    if (!upload) {
      throw new ApiError('upload_not_found', 'Сессия загрузки не найдена', 404);
    }

    if (upload.status === 'completed' && upload.item) {
      return {
        status: 'completed',
        item: upload.item,
        metadata: upload.item.metadata,
      };
    }

    return {
      status: upload.status,
    };
  }

  public cancelUpload(key?: string): void {
    if (key) {
      const upload = this.uploads.get(key);
      if (upload) {
        upload.status = 'cancelled';
      }
    }
  }

  public getItemDownloadUrl(id: number): DownloadUrlResponse {
    const user = this.getCurrentUser();
    const item = this.items.get(id);
    if (!item) {
      throw new ApiError('item_not_found', 'Предмет не найден', 404);
    }

    // Проверка: владелец (owner_id == uid)
    if (item.owner_id !== user.id) {
      throw new ApiError('forbidden', 'Скачивание разрешено только владельцу предмета', 403);
    }

    // Проверка суточного лимита (1 раз в день на пользователя и предмет)
    const today = new Date().toISOString().slice(0, 10);
    const limitKey = `${user.id}:${id}:${today}`;

    if (this.downloadLimits.has(limitKey)) {
      throw new ApiError(
        'download_limit_reached',
        'Превышен суточный лимит: скачивание этого предмета разрешено не более 1 раза в день',
        429
      );
    }

    this.downloadLimits.add(limitKey);

    const filename = (item.metadata?.filename as string) || (item.metadata?.title as string) || `asset-${id}.bin`;
    return {
      url: `https://s3.local-mock.internal/download/${item.s3_id}/${encodeURIComponent(filename)}?token=presigned-${Date.now()}`,
      expires_at: new Date(Date.now() + 3600000).toISOString(),
    };
  }

  private detectItemType(filename: string): ItemType {
    const ext = filename.split('.').pop()?.toLowerCase() || '';
    if (['fbx', 'obj', 'gltf', 'glb', 'blend', 'max', 'c4d'].includes(ext)) return '3d';
    if (['wav', 'flac', 'mp3', 'ogg', 'aiff'].includes(ext)) return 'audio';
    if (['psd', 'png', 'jpg', 'jpeg', 'tiff', 'svg', 'webp'].includes(ext)) return 'image';
    if (['zip', 'rar', '7z', 'tar', 'gz', 'unitypackage'].includes(ext)) return 'archive';
    if (['mp4', 'mov', 'mkv', 'avi'].includes(ext)) return 'video';
    return 'document';
  }

  private generateItemMetadata(
    filename: string,
    fileSize: number,
    sha256Hex: string,
    type: ItemType
  ) {
    const formatSize = (bytes: number) => {
      if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    };

    const baseMeta = {
      title: filename.replace(/\.[^/.]+$/, ''),
      filename,
      file_size: formatSize(fileSize),
      checksum: sha256Hex,
      sha256_hex: sha256Hex,
    };

    if (type === '3d') {
      return {
        ...baseMeta,
        file_format: filename.split('.').pop()?.toUpperCase() || '3D',
        polygon_count: 42000,
        vertex_count: 24500,
        has_texture: true,
        has_rigging: true,
        render_engine: 'PBR / Lumen Ready',
      };
    }
    if (type === 'audio') {
      return {
        ...baseMeta,
        file_format: filename.split('.').pop()?.toUpperCase() || 'AUDIO',
        duration: '03:45',
        sample_rate_hz: 48000,
        bitrate_kbps: 320,
        channels: 2,
      };
    }
    if (type === 'image') {
      return {
        ...baseMeta,
        file_format: filename.split('.').pop()?.toUpperCase() || 'IMAGE',
        width: 3840,
        height: 2160,
        blur_hash: 'LEHLh[WB2yk8pyoJadR*.7kCMdnj',
      };
    }
    if (type === 'archive') {
      return {
        ...baseMeta,
        file_format: 'ZIP Archive',
        file_count: 18,
        uncompressed_size_bytes: fileSize * 2,
      };
    }
    return {
      ...baseMeta,
      file_format: filename.split('.').pop()?.toUpperCase() || 'DOC',
      page_count: 24,
    };
  }

  // --- Transactions ---
  public listTransactions(userId?: string): LedgerEntry[] {
    const uid = userId || this.currentUserId;
    return this.ledger
      .filter((l) => l.user_id === uid)
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }

  // --- Documents ---
  public listDocuments(): Document[] {
    return Array.from(this.documents.values());
  }

  public getDocument(idOrName: string | number): Document {
    if (typeof idOrName === 'number') {
      const doc = this.documents.get(idOrName);
      if (!doc) throw new ApiError('not_found', 'Документ не найден', 404);
      return doc;
    }
    for (const d of this.documents.values()) {
      if (d.name === idOrName) return d;
    }
    throw new ApiError('not_found', 'Документ не найден', 404);
  }

  public recordConsent(documentId: number, userId?: string) {
    const uid = userId || this.currentUserId;
    this.consents.add(`${documentId}:${uid}`);
  }

  public hasConsent(documentId: number, userId?: string): boolean {
    const uid = userId || this.currentUserId;
    return this.consents.has(`${documentId}:${uid}`);
  }
}

export const mockDb = new MockDatabase();
