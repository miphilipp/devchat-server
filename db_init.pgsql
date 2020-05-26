-- DROP TABLE public."user";
CREATE TABLE public."user" (
    name character varying(40) NOT NULL,
    email character varying(40) NOT NULL,
    password text NOT NULL,
    confirmation_uuid uuid,
    id SERIAL PRIMARY KEY,
    lastonline timestamp without time zone,
    failedloginattempts smallint NOT NULL DEFAULT 0,
    lockedoutsince timestamp without time zone,
    lastfailedlogin timestamp without time zone,
    isdeleted boolean NOT NULL DEFAULT false,
    recovery_uuid uuid,
    recovery_uuid_issue_date timestamp without time zone,
    CONSTRAINT user_email_key UNIQUE (email),
    CONSTRAINT user_name_key UNIQUE (name)
);

-- DROP INDEX public.lower_name_idx;
CREATE INDEX lower_name_idx ON public."user" USING btree
    (lower(name::text) COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;

-- DROP TABLE public.conversation;
CREATE TABLE public.conversation (
    title character varying(100) NOT NULL,
    repourl text,
    id SERIAL PRIMARY KEY
);


-- DROP TABLE public.group_association;
CREATE TABLE public.group_association (
    isadmin boolean NOT NULL DEFAULT false,
    userid integer REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    conversationid integer REFERENCES public.conversation MATCH SIMPLE ON DELETE CASCADE,
    joined timestamp without time zone,
    colorindex integer NOT NULL DEFAULT -1,
    hasleft boolean NOT NULL DEFAULT false,
    CONSTRAINT group_association_pkey PRIMARY KEY (userid, conversationid)
);

-- DROP TABLE public.message;
CREATE TABLE public.message (
    sentdate timestamp without time zone NOT NULL,
    conversationid integer NOT NULL REFERENCES public.conversation MATCH SIMPLE ON DELETE CASCADE,
    id BIGSERIAL PRIMARY KEY,
    iscomplete boolean not null default true,
    userid integer NOT NULL REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    type integer NOT NULL
);

-- DROP INDEX public.message_conversationid_idx;
CREATE INDEX message_conversationid_idx ON public.message USING btree
    (conversationid ASC NULLS LAST)
    TABLESPACE pg_default;

-- DROP INDEX public.message_userid_idx;
CREATE INDEX message_userid_idx ON public.message USING btree
    (userid ASC NULLS LAST)
    TABLESPACE pg_default;


-- DROP TABLE public.programming_language;
CREATE TABLE public.programming_language (
    name character varying(20) PRIMARY KEY,
    runnable boolean NOT NULL
);


-- DROP TABLE public.code_message;
CREATE TABLE public.code_message (
    id BIGINT PRIMARY KEY REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    language character varying(20) NOT NULL REFERENCES public.programming_language (name) MATCH SIMPLE,
    code text NOT NULL,
    title character varying(40) NOT NULL,
    lockedby bigint REFERENCES public.user (id) MATCH SIMPLE on delete set null
);

-- DROP INDEX public.code_message_language_idx;
CREATE INDEX code_message_language_idx
    ON public.code_message USING btree
    (language COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;


-- DROP TABLE public.text_message;
CREATE TABLE public.text_message (
    id BIGINT PRIMARY KEY REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    text text NOT NULL
);

-- DROP TABLE public.media_message;
CREATE TABLE public.media_message (
    text text,
    id bigint PRIMARY KEY REFERENCES public.message (id) MATCH SIMPLE ON DELETE CASCADE
);

-- DROP TABLE public.media_object;
CREATE TABLE public.media_object (
    id SERIAL PRIMARY KEY,
    filetype character varying(40) NOT NULL,
    message bigint NOT NULL REFERENCES public.media_message (id) MATCH SIMPLE ON DELETE CASCADE,
    name character varying(80) NOT NULL,
    meta json
);

CREATE OR REPLACE VIEW public.v_text_message AS
SELECT m.id, t.text, m.sentdate, m.conversationid, m.userid, m.type, u.name as author
FROM public.message m
JOIN public.text_message t ON m.id = t.id
JOIN public.user u ON m.userid = u.id;

CREATE OR REPLACE VIEW public.v_code_message AS
SELECT 
    m.id, 
    c.code, 
    m.sentdate, 
    m.conversationid, 
    m.userid, 
    m.type, 
    c.title, 
    c.language, 
    c.lockedby,
    u.name as author
FROM public.message m
JOIN public.code_message c ON m.id = c.id
JOIN public.user u ON m.userid = u.id;

CREATE OR REPLACE VIEW public.v_media_message AS
SELECT m.id, m.sentdate, m.conversationid, m.userid, m.type, mm.text, u.name as author, m.iscomplete
FROM public.message m
JOIN public.media_message mm ON m.id = mm.id
JOIN public.user u ON m.userid = u.id;

-- DROP TABLE public.message_status;
CREATE TABLE public.message_status (
    userid integer REFERENCES public."user" MATCH SIMPLE ON DELETE CASCADE,
    messageid bigint REFERENCES public.message MATCH SIMPLE ON DELETE CASCADE,
    conversationid integer NOT NULL REFERENCES public.conversation (id) MATCH SIMPLE ON DELETE CASCADE,
    hasread boolean NOT NULL DEFAULT false,
    CONSTRAINT message_status_pkey PRIMARY KEY (messageid, userid)
);


CREATE OR REPLACE VIEW public.v_invitation
 AS
 SELECT c.id AS conversationid,
    c.title AS conversationtitle,
    g.userid AS recipient
   FROM group_association g
     JOIN conversation c ON c.id = g.conversationid
  WHERE g.joined IS NULL;

CREATE OR REPLACE VIEW public.v_joined_member
 AS
 SELECT g.isadmin,
    g.userid,
    g.conversationid,
    g.colorindex,
    g.joined
   FROM group_association g
     JOIN "user" u ON u.id = g.userid
  WHERE g.hasleft = false AND g.joined IS NOT NULL AND u.isdeleted = false;

CREATE OR REPLACE VIEW public.v_admin
 AS
 SELECT group_association.isadmin,
    group_association.userid,
    group_association.conversationid,
    group_association.joined,
    group_association.colorindex
   FROM group_association
  WHERE group_association.isadmin = true AND group_association.joined IS NOT NULL;

CREATE OR REPLACE VIEW public.v_every_member
AS
 SELECT u.name,
    g.userid AS id,
    g.isadmin,
    g.colorindex,
    g.joined IS NOT NULL AS hasjoined,
    g.hasleft,
    u.isdeleted,
    g.conversationid
   FROM group_association g
     JOIN "user" u ON u.id = g.userid;


CREATE EXTENSION pgcrypto;
CREATE EXTENSION "uuid-ossp";