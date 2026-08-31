create table tovar (
    id SERIAL primary key,
    tovar_creater_user INTEGER references users(id),
    tovar_vladelec_user INTEGER references users(id)
-- Я просто хз как хранить картинки и 3d мб объекты, наверное ссылка на чтото
);

-- История перемещения товара (от пользователя к пользователю)
create table history_tovar (
    id SERIAL primary key,
    id_tovar INTEGER references tovar(id),
    old_user INTEGER references users(id),
    new_user INTEGER references users(id),
    created_at timestamp not null default now()
);

-- лот аукционна с пользователем и товаром
create table lot (
    id SERIAL primary key,
    id_user INTEGER references users(id),
    id_tovar INTEGER references tovar(id),
    description jsonb
);
-- Ещё подумать как ставку делать на lot но наверное просто 2 поля ставка и пользователь этой ставки таблица чисто с последней ставкой