alter table users add column system boolean not null default false;

---- create above / drop below ----

alter table users drop column system;
