/**
 * DTO и сущности 1-в-1 соответствующие Go бэкенду.
 * Источник истины: docs/api-contract.md и Go пакеты (internal/user, internal/lot, internal/item, migrations/init.sql).
 */

export interface User {
  id: string; // uuid
  name: string;
  avatar_id?: string | null; // uuid
  email: string;
  balance: number; // int coins
  hold: number; // int coins (balance >= hold)
  created_at: string; // ISO 8601
  deleted_at?: string | null;
}

export interface UserDevice {
  browser?: string;
  os?: string;
}

export interface UserSession {
  refresh_id: number;
  created_at: string;
  expires_at: string;
  revoked_at?: string | null;
  device?: UserDevice;
}

export interface TokenResponse {
  access_token: string;
  refresh_session?: UserSession;
}

export type ItemType = 'image' | '3d' | 'audio' | 'video' | 'document' | 'archive';

export interface ItemMetadata {
  title?: string;
  description?: string;
  filename?: string;
  file_size?: string | number;
  file_format?: string;
  mime_type?: string;
  checksum?: string;
  sha256_hex?: string;
  // 3D
  polygon_count?: number;
  vertex_count?: number;
  has_texture?: boolean;
  has_rigging?: boolean;
  render_engine?: string;
  // Audio / Video
  duration?: number | string;
  sample_rate_hz?: number;
  bitrate_kbps?: number;
  channels?: number;
  width?: number;
  height?: number;
  // Document / Archive
  page_count?: number;
  file_count?: number;
  uncompressed_size_bytes?: number;
  [key: string]: unknown;
}

export interface Item {
  id: number; // bigint
  creator_id: string; // uuid
  owner_id: string; // uuid
  s3_id: string; // uuid
  type: ItemType;
  metadata?: ItemMetadata;
  created_at: string;
}

export interface UploadUrlRequest {
  file_size: number;
  sha_256_hex: string;
  filename: string;
}

export interface UploadUrlResponse {
  url: string;
  key: string; // uuid
}

export interface ConfirmUploadRequest {
  key: string;
}

export interface ConfirmUploadResponse {
  status: 'processing';
  message: string;
}

export interface UploadStatusResponse {
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';
  item?: Item;
  metadata?: ItemMetadata;
  error?: string;
}

export interface DownloadUrlResponse {
  url: string;
  expires_at?: string;
  daily_limit_reached?: boolean;
}

export type LotStatus =
  | 'draft'
  | 'pending_review'
  | 'approved'
  | 'active'
  | 'rejected'
  | 'finished'
  | 'cancelled';

export type MediaType = 'image' | 'video' | '3d';

export interface MediaItem {
  id: string; // uuid
  type: MediaType;
  blur_hash?: string;
  width?: number;
  height?: number;
  duration_sec?: number;
  thumbnail_id?: string | null;
  order: number;
  caption?: string;
}

export interface LotDescription {
  text: string;
  short_text?: string;
  media: MediaItem[];
  attributes?: Record<string, string>;
}

export interface SellerInfo {
  id: string; // uuid
  name: string;
  avatar_id?: string | null;
  rating?: number;
}

export interface WinnerInfo {
  id: string; // uuid
  name: string;
}

export interface ItemSummary {
  id: number;
  type: string;
  preview: string; // uuid
  blur_hash?: string;
}

export interface Lot {
  id: number; // bigint
  seller_id: string; // uuid
  seller: SellerInfo;
  item_id: number; // bigint
  item_summary?: ItemSummary;
  status: LotStatus;
  category: string;
  title: string;
  description: LotDescription;
  start_amount: number; // int coins
  current_bid: number; // int coins
  min_step: number; // int coins
  finish_amount?: number | null; // int coins
  bids_count: number;
  winner?: WinnerInfo | null;
  created_at: string;
  started_at?: string | null;
  ends_at?: string | null;
  finished_at?: string | null;
}

export interface LotFilter {
  query?: string;
  category?: string;
  status?: LotStatus | string;
  min_price?: number;
  max_price?: number;
  sort?: 'price_asc' | 'price_desc' | 'ending_soon' | 'popular' | 'newest' | string;
  page?: number;
  limit?: number;
}

export interface LotListResponse {
  items: Lot[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface Bid {
  id: number;
  lot_id: number;
  user_id: string;
  user_name: string;
  amount: number;
  idempotency_key: string;
  created_at: string;
}

export type HoldStatus = 'active' | 'released' | 'captured';

export interface Hold {
  id: number;
  lot_id: number;
  user_id: string;
  bid_id: number;
  amount: number;
  status: HoldStatus;
  created_at: string;
}

export type LedgerType = 'top_up' | 'payout_of_winnings' | 'transfer_to_seller';

export interface LedgerEntry {
  id: number;
  user_id: string;
  amount: number;
  balance_after: number;
  type: LedgerType;
  lot_id?: number | null;
  created_at: string;
}

export interface Document {
  id: number;
  name: string;
  version: string;
  is_major: boolean;
  text: string;
  published_at: string;
  superseded_at?: string | null;
}

export interface UserConsent {
  document_id: number;
  user_id: string;
  created_at: string;
}

export interface CreateLotInput {
  item_id: number;
  title: string;
  category: string;
  short_text?: string;
  text: string;
  start_amount: number;
  min_step: number;
  duration_minutes: number;
  started_at?: string;
  ends_at?: string;
}

export interface CreateItemInput {
  type: ItemType;
  name: string;
  description?: string;
  metadata?: ItemMetadata;
}

export interface PlaceBidResult {
  bid: Bid;
  lot: Lot;
  user_balance: number;
  user_hold: number;
}

export interface ServerTimeResponse {
  server_time: string; // ISO 8601
  timestamp_ms: number;
}

export interface HttpResponse<T> {
  data?: T;
  error?: string;
}
