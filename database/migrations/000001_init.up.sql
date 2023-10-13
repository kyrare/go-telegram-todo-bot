create table users
(
    id          int auto_increment,
    telegram_id int      not null,
    first_name  text     null,
    user_name   text     null,
    created_at  datetime not null default now(),
    updated_at  datetime not null default now(),
    primary key (id)
) ENGINE = InnoDB;

create unique index users_telegram_id_index
    on users (telegram_id);

create table todos
(
    id         int auto_increment,
    user_id    int      not null,
    text       text     not null,
    checked    tinyint  null,
    created_at datetime not null default now(),
    updated_at datetime not null default now(),
    primary key (id)
) ENGINE = InnoDB;

create index todos_user_id_created_at_index
    on todos (user_id, created_at);

alter table todos
    add constraint todos_users_id_fk
        foreign key (user_id) references users (id)
            on update cascade on delete cascade;