create table users (
  id uuid primary key,
  username text not null unique,
  insert_time timestamptz not null default now(),
  update_time timestamptz not null default now()
);

create trigger on_user_update
before update on users
for each row execute procedure timestamp_update();

grant select, insert, update, delete on table users to {{.app_user}};

---- create above / drop below ----

drop table users;
