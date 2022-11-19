begin;

create table if not exists currency_bans
(
    id serial primary key,
    currency varchar(3) not null unique check (currency <> ''),
    banned bool not null
);

commit;