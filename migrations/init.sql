begin;

-- id и является ключом в s3
create table s3 (
    id uuid primary key default uuidv4(),
    hash text not null unique
);

create table  users (
    id uuid primary key default uuidv7(),
    name text not null,
    avatar_id uuid references s3(id),
    email text not null constraint users_email_lower check (email = lower(email)),
    password_hash text not null,
    deleted_at timestamptz default null,
    balance int not null default 0,
    hold int not null default 0 constraint users_hold_positive check (hold >= 0),
    created_at timestamptz not null default now(),
    constraint users_balance_greater_hold check (balance >= hold)
);

create unique index users_email_lower_key on users (lower(email)) where deleted_at is null;

create table user_sessions (
    id bigint generated always as identity primary key,
    user_id uuid not null references users(id),
    refresh_token_hash bytea not null unique constraint user_sessions_64 check (octet_length(refresh_token_hash) = 32),
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    revoked_at timestamptz default null,
    device jsonb default '{}'::jsonb
);

create index user_sessions_allow on user_sessions (user_id) WHERE revoked_at is null;

-- key - ключ от s3 хранилища
create table items (
    id bigint generated always as identity primary key,
    creator_id uuid not null references users(id),
    owner_id uuid not null references users(id),
    s3_id uuid not null references s3(id),
    type text not null constraint type_is check (type in ('image', '3d', 'audio', 'video', 'document', 'archive')),
    metadata jsonb default '{}'::jsonb,
    created_at timestamptz not null default now()
);

-- лот аукционна с пользователем и товаром, проверка что user_owner владеет item
-- Типо будет 1 лот и история торгов его
create table lots (
    id bigint generated always as identity primary key,
    seller_id uuid not null references users(id),
    item_id bigint not null references items(id),
    -- pending, пока не проверится нельзя выкладывать. reject - отклонено, approved - утверждено проверкой. cancelled - пользователь отменил аукцион
    status text not null default 'draft' constraint status_is check (status in ('draft', 'pending_review', 'approved', 'active', 'rejected', 'finished', 'cancelled')),
    winner_id uuid references users(id) default null,
    start_amount int not null check (start_amount > 0),
    min_step int not null check (min_step > 0),
    finish_amount int default null,
    title text not null,
    description jsonb default '{}'::jsonb,
    created_at timestamptz not null default now(),
    started_at timestamptz default null,
    ends_at timestamptz default null,
    finished_at timestamptz default null,
    constraint lots_start_covers_finish_amount check (finish_amount >= start_amount),
    constraint lots_active_has_started_at check (status <> 'active' or started_at is not null),
    constraint lots_active_has_ends_at check (status <> 'active' or ends_at is not null),
    constraint lots_finished_has_finished_at check (status <> 'finished' or finished_at is not null),
    constraint lots_winner_with_amount check ((winner_id is null) = (finish_amount is null))
);

create unique index ux_lot_item_active on lots (item_id) WHERE status = 'active';

-- amount на что изменился баланс
-- balance_after баланс после изменения
-- Получается для проверки сортируем id desk должно быть amount = balance_after (т.к. по умолчанию в самом начале 0 amount)
-- И после считать что balance_after = amount следующему
create table  ledger_entries (
    id bigint generated always as identity primary key,
    user_id uuid not null references users(id),
    amount int not null constraint amount_not_null check (amount <> 0),
    balance_after int not null constraint balance_after_greater_zero check (balance_after >= 0),
    type text not null constraint type_is check (type in ('top_up', 'payout_of_winnings', 'transfer_to_seller')),
    lot_id bigint default null references lots(id),
    created_at timestamptz not null default now()
);

-- История перемещения товара (от пользователя к пользователю)
create table history_item (
    id bigint generated always as identity primary key,
    item_id bigint not null references items(id),
    old_user uuid not null references users(id),
    new_user uuid not null references users(id),
    created_at timestamptz not null default now(),
    -- если при выйгрыше аукционаа то lot_id вписан
    lot_id bigint references lots(id) default null,
    -- если нет lot_id, description обязателен
    description text default null,
    constraint chk_different_user check (old_user <> new_user),
    constraint chk_description check (lot_id is not null or description is not null)
);

-- select coalesce(max(bids.amount), lot.start_amount) c условиями конечно же
create table bids (
    id bigint generated always as identity primary key,
    lot_id bigint not null references lots(id),
    user_id uuid not null references users(id),
    amount int not null constraint amount_greater_zero check (amount > 0),
    idempotency_key uuid not null unique,
    created_at timestamptz not null default now(),
    unique (id, lot_id)
);

create table hold (
    id bigint generated always as identity primary key,
    lot_id bigint not null references lots(id),
    user_id uuid not null references users(id),
    bid_id bigint not null unique references bids(id),
    amount int not null constraint amount_greater_zero check (amount > 0),
    status text not null default 'active' constraint status_is check (status in ('active', 'released', 'captured')),
    created_at timestamptz not null default now(),
    foreign key (lot_id, bid_id) references bids(lot_id, id)
);

create unique index ux_hold_user_lot_active on hold (lot_id, user_id) where status = 'active';

create table documents (
    id bigint generated always as identity primary key,
    name text not null,
    version text not null default '1.0',
    is_major bool not null default true,
    text text not null,
    published_at timestamptz not null default now(),
    superseded_at timestamptz default null,
    unique (name, version)
);
create unique index on documents (name) where superseded_at is null;

CREATE TABLE user_consents (
    document_id bigint not null references documents(id),
    user_id     uuid not null references users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (document_id, user_id)
);

create index ux_user_consents_user_id on user_consents (user_id);
commit;