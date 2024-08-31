create table user_passwords (
	user_id uuid primary key references users,
	algorithm text not null,
	salt bytea not null,
	min_memory int not null,
	iterations smallint not null,
	parallelism smallint not null,
	digest bytea not null,
	insert_time timestamptz not null default now(),
	update_time timestamptz not null default now()
);

create trigger on_user_password_update
before update on user_passwords
for each row execute procedure timestamp_update();

grant select, insert, update, delete on user_passwords to {{.app_user}};

---- create above / drop below ----

drop table user_passwords;
