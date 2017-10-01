create table camera (
    id serial primary key not null,
    name varchar(256) not null unique,
    source varchar(256) not null
);

create table period (
    id serial primary key,

    camera_id int not null references camera(id),

    codecs varchar(64) not null default '',
    width int not null default 0,
    height int not null default 0,
    timescale bigint not null default 0,
    frame_rate varchar(64) not null default '',
    time timestamp not null
);

create index on period (camera_id, time);

create table segment (
    id bigserial primary key,

    camera_id int not null references camera(id),
    period_id int not null references period(id),

    index int not null,
    len bigint not null,
    off bigint not null,
    time timestamp not null
);

create index on segment (camera_id, time);


==== MIGRATION FROM v1 ====

alter table segment add column index int not null default 0;
alter table segment alter column index drop default;

UPDATE period SET time = time - interval '2 hours';

UPDATE segment set index = s.rn
FROM
(
    SELECT
        id,
        row_number() over (partition by period_id order by id) AS rn
    FROM segment
) AS s
WHERE segment.id = s.id;

UPDATE segment set time = p.time + (segment.off::float/p.timescale || ' second')::interval
FROM period p
WHERE segment.period_id = p.id;
