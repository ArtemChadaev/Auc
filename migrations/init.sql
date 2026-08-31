-- balance сделать проверку (мб в бд) что не может быть меньше 0, пользователь может распоряжатся balanca - hold
-- Если ставит на ставку где уже есть прошлая, то balance - hold + hold_bid
create table  users (
    id bigint generated always as identity primary key,
    name text not null,
    email text not null,
    password_hash text not null,
    deleted_at timestamptz default null,
    balance int not null default 0,
    hold int not null default 0 constraint hold_greater_zero check (hold >= 0),
    created_at timestamptz not null default now(),
    constraint users_balance_greater_hold check (balance >= hold)
);

create unique index users_email_lower_key on users (lower(email));
-- refresh_token надо в хэш
create table user_sessions (
    id bigint generated always as identity primary key,
    user_id bigint not null references users(id),
    refresh_token_hash text not null unique,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    revoked_at timestamptz default null,
    device jsonb
);

-- так в виде item сделать файл, как набор (пак) фотографий (рисунков или еще чегото) или 3d модели, м.б. все сжатое в архиф
create table item (
    id bigint generated always as identity primary key,
    creator_id bigint not null references users(id),
    owner_id bigint not null references users(id),
-- Я просто хз как хранить картинки и 3d мб объекты, наверное ссылка на чтото
--     Статус на проверки bool добавить без этого нельзя выставить и никто кроме владельца не видит. м.б. реализовать с помщью ИИ еще
    created_at timestamptz not null default now()
);

-- лот аукционна с пользователем и товаром, проверка что user_owner владеет item
-- Типо будет 1 лот и история торгов его
create table lot (
    id bigint generated always as identity primary key,
    seller_id bigint not null references users(id),
    item_id bigint not null references item(id),
    status text not null default 'draft' constraint status_is check (status in ('draft', 'active', 'finished', 'cancelled')),
    winner_id bigint references users(id) default null,
    start_amount int not null check (start_amount > 0),
    min_step int not null check (min_step > 0),
    finish_amount int default null,
    description jsonb,
    created_at timestamptz not null default now(),
    started_at timestamptz default null,
    ends_at timestamptz default null,
    finished_at timestamptz default null,
    constraint lot_start_covers_finish_amount check (finish_amount >= start_amount),
    constraint lot_active_has_started_at check (status <> 'active' or started_at is not null),
    constraint lot_active_has_ends_at check (status <> 'active' or ends_at is not null),
    constraint lot_finished_has_finished_at check (status <> 'finished' or finished_at is not null)
);

create unique index ux_lot_item_active on lot (item_id) WHERE status = 'active';

-- amount на что изменился баланс
-- balance_after баланс после изменения
-- Получается для проверки сортируем id desk должно быть amount = balance_after (т.к. по умолчанию в самом начале 0 amount)
-- И после считать что balance_after = amount следующему
create table  user_coin (
    id bigint generated always as identity primary key,
    user_id bigint not null references users(id),
    amount int not null constraint amount_not_null check (amount <> 0),
    balance_after int not null constraint balance_after_greater_zero check (balance_after >= 0),
    type text not null constraint type_is check (type in ('top_up', 'payout_of_winnings', 'transfer_to_seller', 'refund')),
    lot_id bigint default null references lot(id),
    created_at timestamptz not null default now()
);

-- История перемещения товара (от пользователя к пользователю)
create table history_item (
    id bigint generated always as identity primary key,
    item_id bigint not null references item(id),
    old_user bigint not null references users(id),
    new_user bigint not null references users(id),
    created_at timestamptz not null default now(),
    -- если при выйгрыше аукционаа то lot_id вписан
    lot_id bigint references lot(id) default null,
    -- если нет lot_id, description обязателен
    description text default null,
    constraint chk_different_user check (old_user <> new_user),
    constraint chk_description check (lot_id is not null or description is not null)
);

-- select coalesce(max(bids.amount), lot.start_amount) c условиями конечно же
create table bids (
    id bigint generated always as identity primary key,
    lot_id bigint not null references lot(id),
    user_id bigint not null references users(id),
    amount int not null constraint amount_greater_zero check (amount > 0),
    idempotency_key uuid not null unique,
    created_at timestamptz not null default now()
);

create table hold (
    id bigint generated always as identity primary key,
    lot_id bigint not null references lot(id),
    user_id bigint not null references users(id),
    bid_id bigint not null unique references bids(id),
    amount int not null constraint amount_greater_zero check (amount > 0),
    status text not null default 'active' constraint status_is check (status in ('active', 'released', 'captured')),
    created_at timestamptz not null default now()
);

create unique index ux_hold_user_lot_active on hold (lot_id, user_id) where status = 'active';

-- TODO: Индексы сделать остальные для скорости!!!