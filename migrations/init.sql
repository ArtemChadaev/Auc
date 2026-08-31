create table if not exists users (
    id SERIAL PRIMARY KEY,
    name varchar(255) not null ,
    email varchar(255) not null,
    deleted_at bool not null default false
);
create table if not exists user_sessions (
    id SERIAL PRIMARY KEY,
    id_user INTEGER references users(id),
    refresh_token varchar(255) not null,
    created_at timestamp not null default now(),
    expires_at timestamp not null,
    revoked bool not null default false,
    device json
);
create table if not exists user_coin (
    id SERIAL primary key ,
    id_user INTEGER references users(id),
    created_at timestamp not null default now(),
    change integer not null,
    description_lot jsonb
);
-- description типо название ллота, кто как и почему и зачем
-- Типо сделать функцию или как она, просто сложить за все время от 1 пользователя user_coin операции change, и будет результат, проверку не забыть что не отрицательное число, и как то такую проверку сделать всегда при проверки операции

-- Поменять stavki на англ название норм
-- Подумать над тем что id user_stavki можно просто сложить от id lot + id_user эта комбинация должна быть уникальной и так
create table if not exists user_stavki (
    id SERIAL primary key ,
    id_user INTEGER references users(id),
--     id_lot
    created_at timestamp not null default now(),
    change_at timestamp,
    stavka integer not null,
    vin bool not null default false,
    description_lot jsonb not null
);
-- типо история смены ставки (всегда когда меняется change_at, если change_at пустой, то этой ставки в этой таблицы не будет) сама записаывается в бд без go вообще
create table if not exists history_user_stavki (
    id SERIAL primary key ,
    id_user_stavki integer references user_stavki(id),
    created_at timestamp not null default now(),
    stavka integer not null
);