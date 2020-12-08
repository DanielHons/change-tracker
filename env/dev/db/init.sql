create user migration_user with password 'thisWillBeDifferent';

create user authenticator with password 'this_too';
create user api nologin;
create user anonymous nologin;

grant anonymous to authenticator;


alter schema data owner to migration_user;
alter schema api owner to migration_user;



-- This should never be on production, it helps developing api
grant migration_user to authenticator;