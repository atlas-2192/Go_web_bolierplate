create table walks (
  id uuid primary key,
	user_id uuid not null references users,
	duration interval not null,
	distance_in_miles numeric not null,
	finish_time timestamptz not null default now()
);

create index on walks (user_id);

grant select, insert, update, delete on walks to {{.app_user}};

---- create above / drop below ----

drop table walks;
