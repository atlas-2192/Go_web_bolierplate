create table login_sessions (
  id uuid primary key,
	user_id uuid not null references users,
	user_agent text,
	login_time timestamptz not null,
	login_request_id text not null,
	approximate_last_request_time timestamptz not null
);

create index on login_sessions (user_id);

grant select, insert, update, delete on login_sessions to {{.app_user}};

---- create above / drop below ----

drop table login_sessions;
