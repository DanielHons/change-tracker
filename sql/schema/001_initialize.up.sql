
create table stages_reached_latest_versions
(
    component_name varchar(200) not null,
    stage_name     varchar(200) not null,
    version        varchar(100) not null,
    git_sha        varchar(100) not null,
    timestamp      timestamp    not null default now(),
    primary key (component_name, stage_name, timestamp)
);


create table stages_aliases
(
    component_name varchar(200) not null,
    plain_stage    varchar(200) not null,
    alias          varchar(200) not null,
    primary key (component_name, plain_stage)
);

create view latest_versions as
select s.*, coalesce(sa.alias, s.stage_name) as stage_alias
from stages_reached_latest_versions s
         left outer join stages_aliases sa
                         on s.component_name = sa.component_name and s.stage_name = sa.plain_stage;

create view latest_stages as
select l.*
from latest_versions l
         left outer join latest_versions r on
        r.stage_alias = l.stage_alias and r.component_name = l.component_name and r.timestamp > l.timestamp
         left outer join stages_aliases sa
                         on l.component_name = sa.component_name and l.stage_name = sa.plain_stage
where r.timestamp is null;



-- Can be used to find all components where the sha between stages differs like
-- select * from diff('prod','develop');
CREATE OR REPLACE FUNCTION diff(base text, compare text)
    RETURNS TABLE
            (
                component_name varchar,
                compare_sha    varchar,
                compare_version varchar,
                base_sha       varchar,
                base_version    varchar
            )
AS
$func$
BEGIN

    RETURN QUERY select lsa.component_name, lsa.git_sha,lsa.version, lsb.git_sha, lsb.version as present
                 from latest_stages lsa
                          left outer join latest_stages lsb on lsa.component_name = lsb.component_name
                     and lsb.stage_alias = base and lsa.stage_alias = compare
                 where lsa.stage_alias = compare and lsb.stage_name is null
                    or (lsb.stage_alias = base and lsb.git_sha != lsa.git_sha);


END
$func$ LANGUAGE plpgsql stable;





